package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/services"
	"github.com/jfk9w/hikkabot/subscription"
	"github.com/jfk9w/hikkabot/util"
	_ "github.com/mattn/go-sqlite3/driver"
	"github.com/pkg/errors"
	"github.com/segmentio/ksuid"
)

type SQLConfig struct {
	Driver     string
	Datasource string
}

func (c SQLConfig) validate() {
	if c.Driver == "" {
		panic("driver must not be empty")
	}
	if c.Datasource == "" {
		panic("datasource must not be empty")
	}
	if _, ok := KnownSQLQuirks[c.Driver]; !ok {
		panic(errors.Errorf("unknown driver: %s", c.Driver))
	}
}

type SQL struct {
	db     *sql.DB
	quirks SQLQuirks
}

func NewSQL(config SQLConfig) *SQL {
	config.validate()
	db, err := sql.Open(config.Driver, config.Datasource)
	if err != nil {
		panic(err)
	}
	return (&SQL{db, KnownSQLQuirks[config.Driver]}).init()
}

func (s *SQL) Close() error {
	return s.db.Close()
}

func (s *SQL) query(query string, args ...interface{}) *sql.Rows {
	rows, err := s.db.Query(query, args...)
	for i := 0; i < 5; i++ {
		if s.quirks.RetryQueryOrExec(err, i) {
			rows, err = s.db.Query(query, args...)
		} else {
			break
		}
	}
	if err != nil {
		panic(err)
	}
	return rows
}

func (s *SQL) exec(query string, args ...interface{}) sql.Result {
	res, err := s.db.Exec(query, args...)
	for i := 0; i < 5; i++ {
		if s.quirks.RetryQueryOrExec(err, i) {
			res, err = s.db.Exec(query, args...)
		} else {
			break
		}
	}
	if err != nil {
		panic(err)
	}
	return res
}

func (s *SQL) update(query string, args ...interface{}) int64 {
	result := s.exec(query, args...)
	rows, err := result.RowsAffected()
	if err != nil {
		panic(err)
	}
	return rows
}

func (s *SQL) init() *SQL {
	s.exec(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS subscription (
  id TEXT NOT NULL,
  secondary_id TEXT NOT NULL,
  chat_id BIGINT NOT NULL,
  service TEXT NOT NULL,
  item %s NOT NULL,
  "offset" BIGINT NOT NULL DEFAULT 0,
  updated %s,
  error TEXT
)`, s.quirks.JSONType(), s.quirks.TimeType()))
	s.exec(`CREATE UNIQUE INDEX IF NOT EXISTS i__subscription__id ON subscription(id)`)
	s.exec(`CREATE UNIQUE INDEX IF NOT EXISTS i__subscription__secondary_id ON subscription(secondary_id, chat_id, service)`)

	return s
}

func (s *SQL) itemData(query string, args ...interface{}) *subscription.ItemData {
	rows := s.query(`SELECT id, secondary_id, chat_id, service, item, "offset" FROM subscription `+query+` LIMIT 1`, args...)
	if !rows.Next() {
		_ = rows.Close()
		return nil
	}

	idata := new(subscription.ItemData)
	var (
		serviceID string
		itemJSON  json.RawMessage
	)

	util.Check(rows.Scan(&idata.PrimaryID, &idata.SecondaryID, &idata.ChatID, &serviceID, &itemJSON, &idata.Offset))
	_ = rows.Close()

	service, ok := services.Map[serviceID]
	if !ok {
		panic("unsupported service " + serviceID)
	}

	item := service()
	err := json.Unmarshal(itemJSON, item)
	if err != nil {
		return nil
	}

	idata.Item = item
	return idata
}

func (s *SQL) AddItem(chatID telegram.ID, item subscription.Item) (*subscription.ItemData, bool) {
	idata := &subscription.ItemData{
		Item:        item,
		PrimaryID:   ksuid.New().String(),
		SecondaryID: item.ID(),
		ChatID:      chatID,
	}

	itemJSON, err := json.Marshal(item)
	util.Check(err)

	return idata, s.update(`
INSERT INTO subscription (id, secondary_id, chat_id, service, item, error) 
VALUES ($1, $2, $3, $4, $5, '__notstarted')
ON CONFLICT DO NOTHING`,
		idata.PrimaryID, idata.SecondaryID, idata.ChatID, idata.Service(), itemJSON) == 1
}

func (s *SQL) GetItem(primaryID string) (*subscription.ItemData, bool) {
	item := s.itemData(`WHERE id = $1`, primaryID)
	return item, item != nil
}

func (s *SQL) GetNextItem(chatID telegram.ID) (*subscription.ItemData, bool) {
	item := s.itemData(`WHERE chat_id = $1 AND error IS NULL ORDER BY CASE WHEN updated IS NULL THEN 0 ELSE 1 END, updated`, chatID)
	return item, item != nil
}

func (s *SQL) UpdateOffset(primaryID string, offset int64) bool {
	return s.update(fmt.Sprintf(`
UPDATE subscription
SET "offset" = $1, updated = %s
WHERE id = $2 AND error IS NULL`, s.quirks.Now()),
		offset, primaryID) == 1
}

func (s *SQL) UpdateError(primaryID string, err error) bool {
	return s.update(fmt.Sprintf(`
UPDATE subscription
SET error = $1, updated = %s
WHERE id = $2 AND error IS NULL`, s.quirks.Now()), err.Error(), primaryID) == 1
}

func (s *SQL) ResetError(primaryID string) bool {
	return s.update(fmt.Sprintf(`
UPDATE subscription
SET error = NULL, updated = %s
WHERE id = $1 AND error IS NOT NULL`, s.quirks.Now()), primaryID) == 1
}

func (s *SQL) GetActiveChats() []telegram.ID {
	rows := s.query(`
	SELECT DISTINCT chat_id 
	FROM subscription
	WHERE error IS NULL
	ORDER BY chat_id`)
	chatIDs := make([]telegram.ID, 0)
	for rows.Next() {
		var chatID telegram.ID
		util.Check(rows.Scan(&chatID))
		chatIDs = append(chatIDs, chatID)
	}
	_ = rows.Close()
	return chatIDs
}
