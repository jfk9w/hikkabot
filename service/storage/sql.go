package storage

import (
	"database/sql"

	"github.com/jfk9w-go/hikkabot/service/dvach"

	"github.com/segmentio/ksuid"

	"github.com/jfk9w-go/hikkabot/service"
	"github.com/jfk9w-go/lego"
	telegram "github.com/jfk9w-go/telegram-bot-api"
)

type SQLStorage sql.DB

func SQL(driverName string, dataSourceName string) *SQLStorage {
	db, err := sql.Open(driverName, dataSourceName)
	lego.Check(err)
	return (*SQLStorage)(db).init()
}

func (storage *SQLStorage) db() *sql.DB {
	return (*sql.DB)(storage)
}

func (storage *SQLStorage) query(query string, args ...interface{}) (*sql.Rows, error) {
	return storage.db().Query(query, args...)
}

func (storage *SQLStorage) mustQuery(query string, args ...interface{}) *sql.Rows {
	rows, err := storage.query(query, args...)
	lego.Check(err)
	return rows
}

func (storage *SQLStorage) exec(query string, args ...interface{}) (sql.Result, error) {
	return storage.db().Exec(query, args...)
}

func (storage *SQLStorage) mustExec(query string, args ...interface{}) sql.Result {
	result, err := storage.exec(query, args...)
	lego.Check(err)
	return result
}

func (storage *SQLStorage) update(query string, args ...interface{}) (rows int64, err error) {
	result, err := storage.exec(query, args...)
	if err != nil {
		return
	}

	rows, err = result.RowsAffected()
	return
}

func (storage *SQLStorage) mustUpdate(query string, args ...interface{}) int64 {
	rows, err := storage.update(query, args...)
	lego.Check(err)
	return rows
}

//language=SQL
func (storage *SQLStorage) selectFeed(query string, args ...interface{}) *service.Feed {
	rows := storage.mustQuery(`SELECT id, secondary_id, chat_id, service_id, name, options, offset FROM feed `+query+` LIMIT 1`, args...)
	if !rows.Next() {
		_ = rows.Close()
		return nil
	}

	s := new(service.Feed)
	lego.Check(rows.Scan(&s.ID, &s.SecondaryID, &s.ChatID, &s.ServiceID, &s.Name, &s.OptionsBytes, &s.Offset))
	_ = rows.Close()

	return s
}

//language=SQL
func (storage *SQLStorage) init() *SQLStorage {
	storage.mustExec(`CREATE TABLE IF NOT EXISTS feed (
  id TEXT NOT NULL,
  secondary_id TEXT NOT NULL,
  chat_id INTEGER NOT NULL,
  service_id TEXT NOT NULL,
  name TEXT NOT NULL,
  options TEXT NOT NULL,
  offset INTEGER NOT NULL DEFAULT 0,
  updated TEXT,
  error TEXT
)`)

	storage.mustExec(`CREATE TABLE IF NOT EXISTS dvach_post_ref (
  chat_id INTEGER NOT NULL,
  board_id TEXT NOT NULL,
  thread_id INTEGER NOT NULL,
  num INTEGER NOT NULL,
  username TEXT NOT NULL,
  message_id INTEGER NOT NULL
)`)

	storage.mustExec(`CREATE UNIQUE INDEX IF NOT EXISTS i__feed__id ON feed(id)`)
	storage.mustExec(`CREATE UNIQUE INDEX IF NOT EXISTS i__feed__secondary_id ON feed(secondary_id, chat_id, service_id)`)
	storage.mustExec(`CREATE UNIQUE INDEX IF NOT EXISTS i__dvach_post_ref__id ON dvach_post_ref(chat_id, board_id, thread_id, num)`)
	storage.mustExec(`CREATE UNIQUE INDEX IF NOT EXISTS i__dvach_post_ref__message_id ON dvach_post_ref(chat_id, message_id)`)

	return storage
}

//language=SQL
func (storage *SQLStorage) ActiveSubscribers() []telegram.ID {
	rows := storage.mustQuery(`SELECT DISTINCT chat_id 
FROM feed
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
func (storage *SQLStorage) InsertFeed(s *service.Feed) bool {
	s.ID = ksuid.New().String()
	return storage.mustUpdate(`INSERT OR IGNORE INTO feed (id, secondary_id, chat_id, name, service_id, options) 
VALUES (?, ?, ?, ?, ?, ?)`, s.ID, s.SecondaryID, s.ChatID, s.Name, s.ServiceID, s.OptionsBytes) == 1
}

//language=SQL
func (storage *SQLStorage) NextFeed(chatID telegram.ID) *service.Feed {
	return storage.selectFeed(`WHERE chat_id = ? ORDER BY updated IS NULL DESC, updated ASC`, chatID)
}

//language=SQL
func (storage *SQLStorage) UpdateFeedOffset(id string, offset int64) bool {
	return storage.mustUpdate(`UPDATE feed
SET offset = ?, updated = datetime('now') 
WHERE id = ? AND error IS NULL`, offset, id) == 1
}

//language=SQL
func (storage *SQLStorage) SuspendFeed(id string, err error) *service.Feed {
	s := storage.selectFeed(`WHERE id = ? AND error IS NULL`, id, err)
	if s == nil {
		return nil
	}

	if storage.mustUpdate(`UPDATE feed
SET error = ?
WHERE id = ? AND error IS NULL`, err, id) == 0 {
		return nil
	}

	return s
}

//language=SQL
func (storage *SQLStorage) InsertPostRef(pk *dvach.PostKey, ref *dvach.MessageRef) {
	storage.mustExec(`INSERT OR IGNORE INTO dvach_post_ref (chat_id, board_id, thread_id, num, username, message_id)
VALUES (?, ?, ?, ?, ?)`, pk.ChatID, pk.BoardID, pk.ThreadID, pk.Num, ref.Username, ref.MessageID)
}

//language=SQL
func (storage *SQLStorage) GetPostRef(pk *dvach.PostKey) (*dvach.MessageRef, bool) {
	rows := storage.mustQuery(`SELECT username, message_id
FROM dvach_post_ref 
WHERE chat_id = ? AND board_id = ? AND thread_id = ? AND num = ?`,
		pk.ChatID, pk.BoardID, pk.ThreadID, pk.Num)

	if !rows.Next() {
		_ = rows.Close()
		return nil, false
	}

	ref := new(dvach.MessageRef)
	lego.Check(rows.Scan(&ref.Username, &ref.MessageID))
	_ = rows.Close()

	return ref, true
}
