package storage

import (
	_sql "database/sql"
	"fmt"
	"time"
	"unicode/utf8"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/feed"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"golang.org/x/exp/utf8string"
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
	*_sql.DB
	SQLQuirks
}

func NewSQL(config SQLConfig) *SQL {
	config.validate()
	db, err := _sql.Open(config.Driver, config.Datasource)
	if err != nil {
		panic(err)
	}
	return (&SQL{db, KnownSQLQuirks[config.Driver]}).init()
}

func (s *SQL) query(query string, args ...interface{}) *_sql.Rows {
	rows, err := s.Query(query, args...)
	for i := 0; i < 5; i++ {
		if s.RetryQueryOrExec(err, i) {
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

func (s *SQL) exec(query string, args ...interface{}) _sql.Result {
	res, err := s.Exec(query, args...)
	for i := 0; i < 5; i++ {
		if s.RetryQueryOrExec(err, i) {
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
	  id VARCHAR(50) NOT NULL,
	  chat_id BIGINT NOT NULL,
	  source VARCHAR(20) NOT NULL,
      name VARCHAR(50) NOT NULL,
	  item %s NOT NULL,
	  updated %s,
	  error VARCHAR(100)
	)`, s.JSONType(), s.TimeType())
	s.exec(sql)
	sql = `
	CREATE UNIQUE INDEX IF NOT EXISTS i__subscription__id 
	ON subscription(id, chat_id, source)`
	s.exec(sql)
	sql = fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS reddit_posts (
	  time %s NOT NULL,
	  id VARCHAR(50) NOT NULL,
	  chat_id BIGINT NOT NULL,
	  name VARCHAR(20) NOT NULL,
	  ups INTEGER NOT NULL,
	  sent INTEGER NOT NULL
	)`, s.TimeType())
	s.exec(sql)
	sql = fmt.Sprintf(`
	CREATE UNIQUE INDEX IF NOT EXISTS i__reddit_posts__name
	ON reddit_posts(id, chat_id, name)`)
	s.exec(sql)
	sql = fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS files (
	  time %s NOT NULL,
	  url TEXT NOT NULL,
      hash CHAR(32) NOT NULL,
      collisions INTEGER NOT NULL DEFAULT 0
	)`, s.TimeType())
	s.exec(sql)
	sql = `
	CREATE UNIQUE INDEX IF NOT EXISTS i__files__hash
    ON files(hash)`
	s.exec(sql)
	return s
}

func (s *SQL) selectQuery(query string, args ...interface{}) []feed.Subscription {
	sql := `
	SELECT id, chat_id, source, name, item 
	FROM subscription ` + query
	rows := s.query(sql, args...)
	defer rows.Close()
	res := make([]feed.Subscription, 0)
	for rows.Next() {
		sub := feed.Subscription{}
		bytes := make([]byte, 0)
		if err := rows.Scan(
			&sub.ID.ID,
			&sub.ID.ChatID,
			&sub.ID.Source,
			&sub.Name,
			&bytes); err != nil {
			panic(err)
		}
		sub.RawData = feed.NewRawData(bytes)
		res = append(res, sub)
	}

	return res
}

func (s *SQL) Create(sub *feed.Subscription) bool {
	if utf8.RuneCountInString(sub.ID.ID) > 50 {
		panic("too long id: " + sub.ID.ID)
	}
	if utf8.RuneCountInString(sub.ID.Source) > 20 {
		panic("too long source: " + sub.ID.Source)
	}
	if utf8.RuneCountInString(sub.Name) > 50 {
		panic("too long name: " + sub.Name)
	}
	sql := `
	INSERT INTO subscription (id, chat_id, source, name, item, error) 
	VALUES ($1, $2, $3, $4, $5, '__notstarted')
	ON CONFLICT DO NOTHING`
	return s.update(sql, sub.ID.ID, sub.ID.ChatID, sub.ID.Source, sub.Name, sub.RawData.Bytes()) == 1
}

func (s *SQL) Get(id feed.ID) *feed.Subscription {
	sql := `WHERE id = $1 AND chat_id = $2 AND source = $3 LIMIT 1`
	res := s.selectQuery(sql, id.ID, id.ChatID, id.Source)
	if len(res) == 0 {
		return nil
	} else {
		return &res[0]
	}
}

func (s *SQL) Advance(chatID telegram.ID) *feed.Subscription {
	sql := `
	WHERE chat_id = $1 
	  AND error IS NULL 
	ORDER BY CASE 
	  WHEN updated IS NULL 
		THEN 0 
	  ELSE 1 
	END, updated
	LIMIT 1`
	res := s.selectQuery(sql, chatID)
	if len(res) == 0 {
		return nil
	} else {
		return &res[0]
	}
}

func (s *SQL) Change(id feed.ID, change feed.Change) bool {
	field := "error"
	var value interface{} = nil
	cond := "error IS NULL"
	if change.RawData != nil {
		field = "item"
		value = change.RawData
	} else if change.Error == nil {
		cond = "error IS NOT NULL"
	} else {
		msg := change.Error.Error()
		if utf8.RuneCountInString(msg) > 100 {
			msg = utf8string.NewString(msg).Slice(0, 100)
		}
		value = msg
	}

	sql := fmt.Sprintf(`
	UPDATE subscription
	SET %s = $1, updated = %s
	WHERE id = $2 
      AND chat_id = $3 
      AND source = $4 
      AND %s`, field, s.Now(), cond)
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

func (s *SQL) List(chatID telegram.ID, active bool) []feed.Subscription {
	return s.selectQuery(`WHERE chat_id = ? AND (error IS NULL) = ?`, chatID, active)
}

func (s *SQL) Clear(chatID telegram.ID, error string) int {
	sql := `
	DELETE FROM subscription 
	WHERE chat_id = ? 
	  AND error IS NOT NULL 
      AND error LIKE ?`
	return int(s.update(sql, chatID, error))
}

func (s *SQL) RedditUpPivot(id feed.ID, percentile float64, period time.Duration) int {
	sql := fmt.Sprintf(`
	with t as (select name, ups
               from reddit_posts
			   where id = $1 AND chat_id = $2 AND time > %s),
     u as (select name, max(ups) as ups from t group by name),
     v as (select ups, dense_rank() over (order by ups) as rank from u),
     w as (select ups, rank * 100 / (select count(distinct rank) from v) as percentile from v),
     x as (select max(ups) as ups, percentile from w group by percentile)
	select ups from x 
    where percentile >= $3
    order by percentile 
    limit 1`, s.Ago(period))
	res := s.query(sql, id.ID, id.ChatID, int(percentile*100))
	defer res.Close()
	var threshold int
	if res.Next() {
		if err := res.Scan(&threshold); err != nil {
			panic(err)
		}
	}

	return threshold
}

func (s *SQL) RedditPost(id feed.ID, name string, ups int, sent bool, period time.Duration) bool {
	sql := fmt.Sprintf(`
	SELECT sent
	FROM reddit_posts
	WHERE id = ? AND chat_id = ? AND name = ? AND time > %s
	LIMIT 1`, s.Ago(period))
	res := s.query(sql, id.ID, id.ChatID, name)
	state := -1
	if res.Next() {
		if err := res.Scan(&state); err != nil {
			res.Close()
			panic(err)
		}
	}
	res.Close()

	switch state {
	case 1:
		// post has already been sent
		sql := fmt.Sprintf(`
		UPDATE reddit_posts
		SET time = %s, ups = $1, sent = sent OR $2
		WHERE id = $3 AND chat_id = $4 AND name = $5`, s.Now())
		s.update(sql, ups, sent, id.ID, id.ChatID, name)
		return false
	case 0:
		// post has not been sent
		sql := fmt.Sprintf(`
		UPDATE reddit_posts
		SET time = %s, ups = $1, sent = sent OR $2
		WHERE id = $3 AND chat_id = $4 AND name = $5`, s.Now())
		s.update(sql, ups, sent, id.ID, id.ChatID, name)
		return sent
	default:
		// post has not been seen
		sql := fmt.Sprintf(`
		INSERT INTO reddit_posts (time, id, chat_id, name, ups, sent)
		VALUES (%s, $1, $2, $3, $4, $5)
		ON CONFLICT DO NOTHING`, s.Now())
		s.update(sql, id.ID, id.ChatID, name, ups, sent)
		return sent
	}
}

func (s *SQL) Delete(id feed.ID) bool {
	sql := `
	DELETE FROM subscription
	WHERE id = ? AND chat_id = ? AND source = ? AND error IS NOT NULL`
	return s.update(sql, id.ID, id.ChatID, id.Source) == 1
}

func (s *SQL) FileHash(url, hash string) bool {
	sql := fmt.Sprintf(`
	INSERT INTO files (time, url, hash) 
	VALUES (%s, ?, ?)
	ON CONFLICT DO NOTHING`, s.Now())
	if s.update(sql, url, hash) == 0 {
		sql := `UPDATE files
		SET collisions = collisions + 1
		WHERE hash = ?`
		s.update(sql, hash)
		return true
	}

	return false
}
