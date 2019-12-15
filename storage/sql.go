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

func (c SQLConfig) validate() error {
	if c.Driver == "" {
		return errors.New("driver must not be empty")
	}

	if c.Datasource == "" {
		return errors.New("datasource must not be empty")
	}

	if _, ok := sqlDialects[c.Driver]; !ok {
		return errors.Errorf("unknown sql dialect: %s", c.Driver)
	}

	return nil
}

type SQL struct {
	db      *sql.DB
	dialect sqlDialect
}

func NewSQL(config SQLConfig) *SQL {
	util.Check(config.validate())
	db, err := sql.Open(config.Driver, config.Datasource)
	util.Check(err)
	return (&SQL{db, sqlDialects[config.Driver]}).init()
}

func (s *SQL) Close() error {
	return s.db.Close()
}

func (s *SQL) query(query string, args ...interface{}) (*sql.Rows, error) {
	return s.db.Query(query, args...)
}

func (s *SQL) mustQuery(query string, args ...interface{}) *sql.Rows {
	rows, err := s.query(query, args...)
	util.Check(err)
	return rows
}

func (s *SQL) exec(query string, args ...interface{}) (sql.Result, error) {
	return s.db.Exec(query, args...)
}

func (s *SQL) mustExec(query string, args ...interface{}) sql.Result {
	result, err := s.exec(query, args...)
	util.Check(err)
	return result
}

func (s *SQL) update(query string, args ...interface{}) (rows int64, err error) {
	result, err := s.exec(query, args...)
	if err != nil {
		return
	}

	rows, err = result.RowsAffected()
	return
}

func (s *SQL) mustUpdate(query string, args ...interface{}) int64 {
	rows, err := s.update(query, args...)
	util.Check(err)
	return rows
}

func (s *SQL) init() *SQL {
	s.mustExec(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS subscription (
  id TEXT NOT NULL,
  secondary_id TEXT NOT NULL,
  chat_id BIGINT NOT NULL,
  service TEXT NOT NULL,
  item %s NOT NULL,
  "offset" BIGINT NOT NULL DEFAULT 0,
  updated %s,
  error TEXT
)`, s.dialect.jsonType(), s.dialect.timeType()))

	s.mustExec(`CREATE UNIQUE INDEX IF NOT EXISTS i__subscription__id ON subscription(id)`)
	s.mustExec(`CREATE UNIQUE INDEX IF NOT EXISTS i__subscription__secondary_id ON subscription(secondary_id, chat_id, service)`)

	return s
}

func (s *SQL) itemData(query string, args ...interface{}) *subscription.ItemData {
	rows := s.mustQuery(`SELECT id, secondary_id, chat_id, service, item, "offset" FROM subscription `+query+` LIMIT 1`, args...)
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

	return idata, s.mustUpdate(`
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
	return s.mustUpdate(fmt.Sprintf(`
UPDATE subscription
SET "offset" = $1, updated = %s
WHERE id = $2 AND error IS NULL`, s.dialect.now()),
		offset, primaryID) == 1
}

func (s *SQL) UpdateError(primaryID string, err error) bool {
	return s.mustUpdate(fmt.Sprintf(`
UPDATE subscription
SET error = $1, updated = %s
WHERE id = $2 AND error IS NULL`, s.dialect.now()), err.Error(), primaryID) == 1
}

func (s *SQL) ResetError(primaryID string) bool {
	return s.mustUpdate(fmt.Sprintf(`
UPDATE subscription
SET error = NULL, updated = %s
WHERE id = $1 AND error IS NOT NULL`, s.dialect.now()), primaryID) == 1
}

func (s *SQL) GetActiveChats() []telegram.ID {
	rows := s.mustQuery(`
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
