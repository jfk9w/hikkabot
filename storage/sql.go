package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/feed"
	"github.com/jfk9w/hikkabot/services"
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
	*sql.DB
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

func (s *SQL) query(query string, args ...interface{}) *sql.Rows {
	rows, err := s.Query(query, args...)
	for i := 0; i < 5; i++ {
		if s.quirks.RetryQueryOrExec(err, i) {
			rows, err = s.Query(query, args...)
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
	res, err := s.Exec(query, args...)
	for i := 0; i < 5; i++ {
		if s.quirks.RetryQueryOrExec(err, i) {
			res, err = s.Exec(query, args...)
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
	s.exec(fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS subscription (
	  id TEXT NOT NULL,
	  secondary_id TEXT NOT NULL,
	  chat_id BIGINT NOT NULL,
	  service TEXT NOT NULL,
	  item %s NOT NULL,
	  "offset" BIGINT NOT NULL DEFAULT 0,
	  updated %s,
	  error TEXT
	)`, s.quirks.JSONType(), s.quirks.TimeType()))
	s.exec(`
	CREATE UNIQUE INDEX IF NOT EXISTS i__subscription__id 
	ON subscription(id)`)
	s.exec(`
	CREATE UNIQUE INDEX IF NOT EXISTS i__subscription__secondary_id 
	ON subscription(secondary_id, chat_id, service)`)
	return s
}

func (s *SQL) itemData(query string, args ...interface{}) *feed.ItemData {
	rows := s.query(`
	SELECT id, secondary_id, chat_id, service, item, "offset" 
	FROM subscription `+query+` LIMIT 1`, args...)
	defer rows.Close()
	if !rows.Next() {
		return nil
	}
	idata := new(feed.ItemData)
	var (
		serviceID string
		itemJSON  json.RawMessage
	)
	util.Check(rows.Scan(&idata.PrimaryID, &idata.SecondaryID, &idata.ChatID, &serviceID, &itemJSON, &idata.Offset))
	service, ok := services.Map[serviceID]
	if !ok {
		panic("unsupported service " + serviceID)
	}
	item := service()
	err := json.Unmarshal(itemJSON, item)
	if err != nil {
		panic(err)
	}
	idata.Item = item
	return idata
}

func (s *SQL) Create(chatID telegram.ID, item feed.Item) (*feed.ItemData, bool) {
	idata := &feed.ItemData{
		Item:        item,
		PrimaryID:   ksuid.New().String(),
		SecondaryID: item.ID(),
		ChatID:      chatID,
	}
	itemJSON, err := json.Marshal(item)
	if err != nil {
		panic(err)
	}
	return idata, s.update(`
	INSERT INTO subscription (id, secondary_id, chat_id, service, item, error) 
	VALUES ($1, $2, $3, $4, $5, '__notstarted')
	ON CONFLICT DO NOTHING`,
		idata.PrimaryID, idata.SecondaryID, idata.ChatID, idata.Service(), itemJSON) == 1
}

func (s *SQL) Get(primaryID string) (*feed.ItemData, bool) {
	item := s.itemData(`
	WHERE id = $1`, primaryID)
	return item, item != nil
}

func (s *SQL) Advance(chatID telegram.ID) (*feed.ItemData, bool) {
	item := s.itemData(`
	WHERE chat_id = $1 
	  AND error IS NULL 
	ORDER BY CASE 
	  WHEN updated IS NULL 
		THEN 0 
	  ELSE 1 
	END, updated`, chatID)
	return item, item != nil
}

func (s *SQL) Update(id string, change feed.Change) bool {
	var (
		field             = "error"
		value interface{} = nil
		cond              = "error IS NULL"
	)

	if change.Offset != 0 {
		field = `"offset"`
		value = change.Offset
	} else if change.Error == nil {
		cond = "error IS NOT NULL"
	} else {
		value = change.Error.Error()
	}

	sql := fmt.Sprintf(`
	UPDATE subscription
	SET %s = $1, updated = %s
	WHERE id = $2 AND %s`, field, s.quirks.Now(), cond)
	return s.update(sql, value, id) == 1
}

func (s *SQL) Active() []telegram.ID {
	rows := s.query(`
	SELECT DISTINCT chat_id 
	FROM subscription
	WHERE error IS NULL
	ORDER BY chat_id`)
	defer rows.Close()
	chatIDs := make([]telegram.ID, 0)
	for rows.Next() {
		chatID := new(telegram.ID)
		if err := rows.Scan(chatID); err != nil {
			panic(err)
		}
		chatIDs = append(chatIDs, *chatID)
	}
	return chatIDs
}
