package storage

import (
	"database/sql"

	"github.com/jfk9w-go/hikkabot/service"
	"github.com/jfk9w-go/hikkabot/service/dvach"
	"github.com/jfk9w-go/lego"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/segmentio/ksuid"
)

type SQLStorage sql.DB

func SQL(driverName string, dataSourceName string) *SQLStorage {
	db, err := sql.Open(driverName, dataSourceName)
	lego.Check(err)
	return (*SQLStorage)(db).init()
}

func (s *SQLStorage) db() *sql.DB {
	return (*sql.DB)(s)
}

func (s *SQLStorage) query(query string, args ...interface{}) (*sql.Rows, error) {
	return s.db().Query(query, args...)
}

func (s *SQLStorage) mustQuery(query string, args ...interface{}) *sql.Rows {
	rows, err := s.query(query, args...)
	lego.Check(err)
	return rows
}

func (s *SQLStorage) exec(query string, args ...interface{}) (sql.Result, error) {
	return s.db().Exec(query, args...)
}

func (s *SQLStorage) mustExec(query string, args ...interface{}) sql.Result {
	result, err := s.exec(query, args...)
	lego.Check(err)
	return result
}

func (s *SQLStorage) update(query string, args ...interface{}) (rows int64, err error) {
	result, err := s.exec(query, args...)
	if err != nil {
		return
	}

	rows, err = result.RowsAffected()
	return
}

func (s *SQLStorage) mustUpdate(query string, args ...interface{}) int64 {
	rows, err := s.update(query, args...)
	lego.Check(err)
	return rows
}

func (s *SQLStorage) selectFeed(query string, args ...interface{}) *service.Feed {
	rows := s.mustQuery(`SELECT id, secondary_id, chat_id, service_id, name, options, offset FROM feed `+query+` LIMIT 1`, args...)
	if !rows.Next() {
		_ = rows.Close()
		return nil
	}

	f := new(service.Feed)
	lego.Check(rows.Scan(&f.ID, &f.SecondaryID, &f.ChatID, &f.ServiceID, &f.Name, &f.OptionsBytes, &f.Offset))
	_ = rows.Close()

	return f
}

func (s *SQLStorage) init() *SQLStorage {
	s.mustExec(`CREATE TABLE IF NOT EXISTS feed (
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

	s.mustExec(`CREATE TABLE IF NOT EXISTS dvach_post_ref (
  chat_id INTEGER NOT NULL,
  board_id TEXT NOT NULL,
  thread_id INTEGER NOT NULL,
  num INTEGER NOT NULL,
  username TEXT NOT NULL,
  message_id INTEGER NOT NULL
)`)

	s.mustExec(`CREATE UNIQUE INDEX IF NOT EXISTS i__feed__id ON feed(id)`)
	s.mustExec(`CREATE UNIQUE INDEX IF NOT EXISTS i__feed__secondary_id ON feed(secondary_id, chat_id, service_id)`)
	s.mustExec(`CREATE UNIQUE INDEX IF NOT EXISTS i__dvach_post_ref__id ON dvach_post_ref(chat_id, board_id, thread_id, num)`)
	s.mustExec(`CREATE UNIQUE INDEX IF NOT EXISTS i__dvach_post_ref__message_id ON dvach_post_ref(chat_id, message_id)`)

	return s
}

func (s *SQLStorage) ActiveSubscribers() []telegram.ID {
	rows := s.mustQuery(`SELECT DISTINCT chat_id 
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

func (s *SQLStorage) InsertFeed(f *service.Feed) bool {
	f.ID = ksuid.New().String()
	return s.mustUpdate(`INSERT OR IGNORE INTO feed (id, secondary_id, chat_id, name, service_id, options) 
VALUES (?, ?, ?, ?, ?, ?)`, f.ID, f.SecondaryID, f.ChatID, f.Name, f.ServiceID, f.OptionsBytes) == 1
}

func (s *SQLStorage) NextFeed(chatID telegram.ID) *service.Feed {
	return s.selectFeed(`WHERE chat_id = ? AND error IS NULL ORDER BY updated IS NULL DESC, updated ASC`, chatID)
}

func (s *SQLStorage) UpdateFeedOffset(id string, offset int64) bool {
	return s.mustUpdate(`UPDATE feed
SET offset = ?, updated = datetime('now') 
WHERE id = ? AND error IS NULL`, offset, id) == 1
}

func (s *SQLStorage) GetFeed(id string) *service.Feed {
	return s.selectFeed(`WHERE id = ? AND error IS NULL`, id)
}

func (s *SQLStorage) SuspendFeed(id string, err error) bool {
	return s.mustUpdate(`UPDATE feed
SET error = ?
WHERE id = ? AND error IS NULL`, err.Error(), id) > 0
}

func (s *SQLStorage) InsertPostRef(pk *dvach.PostKey, ref *dvach.MessageRef) {
	s.mustExec(`INSERT OR IGNORE INTO dvach_post_ref (chat_id, board_id, thread_id, num, username, message_id)
VALUES (?, ?, ?, ?, ?, ?)`, pk.ChatID, pk.BoardID, pk.ThreadID, pk.Num, ref.Username, ref.MessageID)
}

func (s *SQLStorage) GetPostRef(pk *dvach.PostKey) (*dvach.MessageRef, bool) {
	rows := s.mustQuery(`SELECT username, message_id
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
