package service

import (
	"database/sql"

	"github.com/jfk9w-go/lego"
	"github.com/jfk9w-go/telegram-bot-api"
)

type SqlStorage sql.DB

func NewSqlStorage(driverName string, dataSourceName string) *SqlStorage {
	db, err := sql.Open(driverName, dataSourceName)
	lego.Check(err)
	return (*SqlStorage)(db).init()
}

func (storage *SqlStorage) db() *sql.DB {
	return (*sql.DB)(storage)
}

func (storage *SqlStorage) query(query string, args ...interface{}) (*sql.Rows, error) {
	return storage.db().Query(query, args...)
}

func (storage *SqlStorage) mustQuery(query string, args ...interface{}) *sql.Rows {
	rows, err := storage.query(query, args...)
	lego.Check(err)
	return rows
}

func (storage *SqlStorage) exec(query string, args ...interface{}) (sql.Result, error) {
	return storage.db().Exec(query, args...)
}

func (storage *SqlStorage) mustExec(query string, args ...interface{}) sql.Result {
	result, err := storage.exec(query, args...)
	lego.Check(err)
	return result
}

func (storage *SqlStorage) update(query string, args ...interface{}) (rows int64, err error) {
	result, err := storage.exec(query, args...)
	if err != nil {
		return
	}

	rows, err = result.RowsAffected()
	return
}

func (storage *SqlStorage) mustUpdate(query string, args ...interface{}) int64 {
	rows, err := storage.update(query, args...)
	lego.Check(err)
	return rows
}

//language=SQL
func (storage *SqlStorage) selectSingle(query string, args ...interface{}) *Subscription {
	rows := storage.mustQuery(`SELECT id, secondary_id, chat_id, type, name, options, offset FROM subscription `+query+` LIMIT 1`, args...)
	if !rows.Next() {
		_ = rows.Close()
		return nil
	}

	s := new(Subscription)
	lego.Check(rows.Scan(&s.ID, &s.SecondaryID, &s.ChatID, &s.Type, &s.Name, &s.Options, &s.Offset))
	_ = rows.Close()

	return s
}

//language=SQL
func (storage *SqlStorage) init() *SqlStorage {
	storage.mustExec(`CREATE TABLE IF NOT EXISTS subscription (
  id TEXT NOT NULL,
  secondary_id TEXT NOT NULL,
  chat_id INTEGER NOT NULL,
  type TEXT NOT NULL,
  name TEXT NOT NULL,
  options TEXT NOT NULL,
  offset INTEGER NOT NULL DEFAULT 0,
  updated TEXT,
  error TEXT
)`)

	storage.mustExec(`CREATE UNIQUE INDEX IF NOT EXISTS i__subscription__id ON subscription(id)`)
	storage.mustExec(`CREATE UNIQUE INDEX IF NOT EXISTS i__subscription__secondary_id ON subscription(secondary_id, chat_id, type)`)

	return storage
}

//language=SQL
func (storage *SqlStorage) Active() []telegram.ID {
	rows := storage.mustQuery(`SELECT DISTINCT chat_id 
FROM subscription
WHERE error IS NULL
ORDER BY chat_id ASC`)

	chatIds := make([]telegram.ID, 0)
	for rows.Next() {
		var chatId telegram.ID
		lego.Check(rows.Scan(&chatId))
		chatIds = append(chatIds, chatId)
	}

	_ = rows.Close()
	return chatIds
}

//language=SQL
func (storage *SqlStorage) Insert(s *Subscription) bool {
	return storage.mustUpdate(`INSERT OR IGNORE INTO subscription (id, secondary_id, chat_id, name, type, options) 
VALUES (?, ?, ?, ?, ?, ?)`, s.ID, s.SecondaryID, s.ChatID, s.Name, s.Type, s.Options) == 1
}

//language=SQL
func (storage *SqlStorage) Query(chatId telegram.ID) *Subscription {
	return storage.selectSingle(`WHERE chat_id = ? ORDER BY updated IS NULL DESC, updated ASC`, chatId)
}

//language=SQL
func (storage *SqlStorage) Update(id string, offset Offset) bool {
	return storage.mustUpdate(`UPDATE subscription
SET offset = ?, updated = datetime('now') 
WHERE id = ? AND error IS NULL`, offset, id) == 1
}

//language=SQL
func (storage *SqlStorage) Suspend(id string, err error) *Subscription {
	s := storage.selectSingle(`WHERE id = ? AND error IS NULL`, id, err)
	if s == nil {
		return nil
	}

	if storage.mustUpdate(`UPDATE subscription
SET error = ?
WHERE id = ? AND error IS NULL`, err, id) == 0 {
		return nil
	}

	return s
}
