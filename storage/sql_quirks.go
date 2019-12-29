package storage

import (
	"log"
	"math"
	"time"
)

var KnownSQLQuirks = map[string]SQLQuirks{
	"pg":      pg{},
	"sqlite3": sqlite3{},
}

type SQLQuirks interface {
	JSONType() string
	TimeType() string
	Now() string
	RetryQueryOrExec(error, int) bool
}

type pg struct{}

func (pg) JSONType() string {
	return "JSONB"
}

func (pg) TimeType() string {
	return "TIMESTAMP WITH TIME ZONE"
}

func (pg) Now() string {
	return "now()"
}

func (pg) RetryQueryOrExec(error, int) bool {
	return false
}

type sqlite3 struct{}

func (sqlite3) JSONType() string {
	return "TEXT"
}

func (sqlite3) TimeType() string {
	return "TEXT"
}

func (sqlite3) Now() string {
	return `datetime("now")`
}

func (sqlite3) RetryQueryOrExec(err error, try int) bool {
	if err != nil && err.Error() == "database is locked" {
		timeout := time.Duration(math.Pow(float64(try), 2)) * 100 * time.Millisecond
		log.Printf("Database is locked, sleeping for %v", timeout)
		time.Sleep(timeout)
		return true
	} else {
		return false
	}
}
