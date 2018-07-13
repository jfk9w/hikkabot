package service

import (
	"database/sql"
	"time"

	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/telegram"
	"github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
)

var _ = sqlite3.Version

var SuspendedByUser = errors.Errorf("interrupted by user")

const driver = "sqlite3"

const (
	All   = "all"
	Text  = "text"
	Media = "media"
)

type FeedItem struct {
	Ref      dvach.Ref
	LastPost int
	Mode     string
	Header   string
	Error    error
	Exists   bool
}

type DB sql.DB

func OpenDB(filename string) *DB {
	db, err := sql.Open(driver, filename)
	checkpanic(err)
	return (*DB)(db)
}

func (db *DB) query(query string, args ...interface{}) *sql.Rows {
	rows, err := (*sql.DB)(db).Query(query, args...)
	checkpanic(err)
	return rows
}

func (db *DB) exec(query string, args ...interface{}) sql.Result {
	result, err := (*sql.DB)(db).Exec(query, args...)
	checkpanic(err)
	return result
}

func (db *DB) update(query string, args ...interface{}) int64 {
	rows, err := db.exec(query, args...).RowsAffected()
	checkpanic(err)
	return rows
}

//language=SQL
func (db *DB) InitSchema() *DB {
	db.exec(`CREATE TABLE IF NOT EXISTS feed (
chat INTEGER NOT NULL,
board TEXT NOT NULL,
thread TEXT NOT NULL,
last_post INTEGER NOT NULL DEFAULT 0,
mode TEXT NOT NULL,
header TEXT NOT NULL,
updated INTEGER NOT NULL DEFAULT 0,
error TEXT NOT NULL DEFAULT '')`)

	db.exec(`CREATE UNIQUE INDEX IF NOT EXISTS i__feed__id ON feed (chat, board, thread)`)
	return db
}

//language=SQL
func (db *DB) Feed(chat telegram.ChatID) (item FeedItem) {
	var (
		rs  *sql.Rows
		err error
	)

	rs = db.query(`SELECT board, thread, last_post, mode, header
FROM feed
WHERE chat = ? AND error = ''
ORDER BY updated ASC
LIMIT 1`, chat)

	if !rs.Next() {
		return
	}

	var board, thread string
	checkpanic(rs.Scan(&board, &thread, &item.LastPost, &item.Mode, &item.Header))
	checkpanic(rs.Close())

	item.Ref, err = dvach.ToRef(board, thread)
	checkpanic(err)

	item.Exists = true
	return
}

//language=SQL
func (db *DB) UpdateSubscription(chat telegram.ChatID, item FeedItem) bool {
	return db.update(`UPDATE feed
SET updated = ?, last_post = ?
WHERE chat = ? AND board = ? AND thread = ? AND error = ''`,
		now(), item.LastPost,
		chat, item.Ref.Board, item.Ref.NumString,
	) > 0
}

//language=SQL
func (db *DB) SuspendSubscription(chat telegram.ChatID, ref dvach.Ref, reason error) bool {
	return db.update(`UPDATE feed
SET updated = ?, error = ?
WHERE chat = ? AND board = ? AND thread = ? AND error = ''`,
		now(), reason.Error(),
		chat, ref.Board, ref.NumString,
	) > 0
}

//language=SQL
func (db *DB) SuspendAccount(chat telegram.ChatID, reason error) int64 {
	return db.update(`UPDATE feed
SET updated = ?, error = ?
WHERE chat = ? AND error = ''`,
		now(), reason.Error(),
		chat,
	)
}

//language=SQL
func (db *DB) CreateSubscription(chat telegram.ChatID, item FeedItem) bool {
	return db.update(`INSERT OR IGNORE INTO feed (chat, mode, board, thread, header, last_post)
VALUES (?, ?, ?, ?, ?, ?)`,
		chat, item.Mode, item.Ref.Board, item.Ref.NumString, item.Header, item.LastPost,
	) > 0
}

//language=SQL
func (db *DB) LoadActiveAccounts() []telegram.ChatID {
	var (
		rs    *sql.Rows
		chats = make([]telegram.ChatID, 0)
	)

	rs = db.query(`SELECT DISTINCT chat 
FROM feed
WHERE error = ''
ORDER BY chat ASC`)

	for rs.Next() {
		var chat telegram.ChatID
		checkpanic(rs.Scan(&chat))
		chats = append(chats, chat)
	}

	checkpanic(rs.Close())
	return chats
}

func (db *DB) Query(query string) ([][]string, error) {
	var rs, err = (*sql.DB)(db).Query(query)
	if err != nil {
		return nil, err
	}

	defer rs.Close()

	var rows = make([][]string, 0)
	var header []string
	header, err = rs.Columns()
	if err != nil {
		return nil, err
	}

	rows = append(rows, header)
	for rs.Next() {
		var (
			row = make([]string, len(header))
			raw = make([]interface{}, len(header))
		)

		for i := range row {
			raw[i] = &row[i]
		}

		err = rs.Scan(raw...)
		if err != nil {
			return nil, err
		}

		rows = append(rows, row)
	}

	return rows, nil
}

func (db *DB) Exec(query string) (int64, error) {
	var r, err = (*sql.DB)(db).Exec(query)
	if err != nil {
		return 0, err
	}

	return r.RowsAffected()
}

func (db *DB) Close() {
	checkpanic((*sql.DB)(db).Close())
}

func now() int64 {
	return time.Now().UnixNano() / 1e3
}

func checkpanic(err error) {
	if err != nil {
		panic(err)
	}
}
