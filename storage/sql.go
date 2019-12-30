package storage

import (
	"database/sql"
	"fmt"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/feed"
	_ "github.com/mattn/go-sqlite3/driver"
	"github.com/pkg/errors"
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
	sql := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS subscription (
	  id TEXT NOT NULL,
	  chat_id BIGINT NOT NULL,
	  source TEXT NOT NULL,
      name TEXT NOT NULL,
	  item %s NOT NULL,
	  "offset" BIGINT NOT NULL DEFAULT 0,
	  updated %s,
	  error TEXT
	)`, s.quirks.JSONType(), s.quirks.TimeType())
	s.exec(sql)
	sql = `
	CREATE UNIQUE INDEX IF NOT EXISTS i__subscription__id 
	ON subscription(id, chat_id, source)`
	s.exec(sql)
	return s
}

func (s *SQL) selectQuery(query string, args ...interface{}) *feed.Subscription {
	sql := `
	SELECT id, chat_id, source, name, item, "offset" 
	FROM subscription ` + query + ` LIMIT 1`
	rows := s.query(sql, args...)
	defer rows.Close()
	if !rows.Next() {
		return nil
	}
	sub := new(feed.Subscription)
	sub.Item = []byte{}
	if err := rows.Scan(
		&sub.ID.ID,
		&sub.ID.ChatID,
		&sub.ID.Source,
		&sub.Name,
		&sub.Item,
		&sub.Offset); err != nil {
		panic(err)
	}
	return sub
}

func (s *SQL) Create(sub *feed.Subscription) bool {
	sql := `
	INSERT INTO subscription (id, chat_id, source, name, item, error) 
	VALUES ($1, $2, $3, $4, $5, '__notstarted')
	ON CONFLICT DO NOTHING`
	return s.update(sql, sub.ID.ID, sub.ID.ChatID, sub.ID.Source, sub.Name, sub.Item) == 1
}

func (s *SQL) Get(id feed.ID) *feed.Subscription {
	sql := `WHERE id = $1 AND chat_id = $2 AND source = $3`
	return s.selectQuery(sql, id.ID, id.ChatID, id.Source)
}

func (s *SQL) Advance(chatID telegram.ID) *feed.Subscription {
	sql := `
	WHERE chat_id = $1 
	  AND error IS NULL 
	ORDER BY CASE 
	  WHEN updated IS NULL 
		THEN 0 
	  ELSE 1 
	END, updated`
	return s.selectQuery(sql, chatID)
}

func (s *SQL) Change(id feed.ID, change feed.Change) bool {
	field := "error"
	var value interface{} = nil
	cond := "error IS NULL"
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
	WHERE id = $2 
      AND chat_id = $3 
      AND source = $4 
      AND %s`, field, s.quirks.Now(), cond)
	return s.update(sql, value, id.ID, id.ChatID, id.Source) == 1
}

func (s *SQL) Active() []telegram.ID {
	sql := `
	SELECT DISTINCT chat_id 
	FROM subscription
	WHERE error IS NULL
	ORDER BY chat_id`
	rows := s.query(sql)
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
