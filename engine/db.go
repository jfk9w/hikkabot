package engine

import (
	"database/sql"
	"time"

	"github.com/jfk9w-go/hikkabot/feed"
	"github.com/jfk9w-go/telegram"
	"github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
)

var _ = sqlite3.Version

var SuspendedByUser = errors.Errorf("interrupted by user")

const driver = "sqlite3"

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
id TEXT NOT NULL,
type TEXT NOT NULL,
meta TEXT NOT NULL,
offset TEXT NOT NULL,
updated INTEGER NOT NULL DEFAULT 0,
error TEXT NOT NULL DEFAULT '')`)
	db.exec(`CREATE UNIQUE INDEX IF NOT EXISTS i__feed__id ON feed (chat, id, type)`)
	return db
}

//language=SQL
func (db *DB) NextState(chat telegram.ChatID) *feed.State {
	var rs = db.query(`SELECT id, type, meta, offset
FROM feed
WHERE chat = ? AND error = ''
ORDER BY updated ASC
LIMIT 1`,
		chat)

	if !rs.Next() {
		return nil
	}

	var state = new(feed.State)
	checkpanic(rs.Scan(&state.ID, &state.Type, &state.Meta, &state.Offset))
	checkpanic(rs.Close())

	state.Chat = chat
	return state
}

//language=SQL
func (db *DB) PersistState(chat telegram.ChatID, state *feed.State) bool {
	return db.update(`UPDATE feed
SET offset = ?, error = ?, updated = ?
WHERE chat = ? AND id = ? AND type = ? AND error = ''`,
		state.Offset, state.Err(), now(),
		chat, state.ID, state.Type) > 0
}

//language=SQL
func (db *DB) AppendState(chat telegram.ChatID, state *feed.State) bool {
	return db.update(`INSERT OR IGNORE INTO feed (chat, id, type, meta, offset, updated, error)
VALUES (?, ?, ?, ?, ?, 0, '')`,
		chat, state.ID, state.Type, state.Meta, state.Offset) > 0
}

//language=SQL
func (db *DB) Suspend(chat telegram.ChatID) bool {
	return db.update(`UPDATE feed
SET error = ?, updated = ?
WHERE chat = ? AND error = ''`,
		SuspendedByUser.Error(), now(),
		chat) > 0
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
