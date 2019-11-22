package storage

var sqlDialects = map[string]sqlDialect{
	"pg":      pg{},
	"sqlite3": sqlite3{},
}

type sqlDialect interface {
	jsonType() string
	timeType() string
	now() string
}

type pg struct{}

func (pg) jsonType() string {
	return "JSONB"
}

func (pg) timeType() string {
	return "TIMESTAMP WITH TIME ZONE"
}

func (pg) now() string {
	return "now()"
}

type sqlite3 struct{}

func (sqlite3) jsonType() string {
	return "TEXT"
}

func (sqlite3) timeType() string {
	return "TEXT"
}

func (sqlite3) now() string {
	return `datetime("now")`
}
