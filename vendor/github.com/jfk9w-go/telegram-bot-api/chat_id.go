package telegram

import (
	"fmt"
	"strconv"
)

// ID is an item identifier (chat, message, user, etc.)
type ID int64

// ParseID tries to parse a value as ID.
func ParseID(value string) (ID, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	return ID(id), err
}

// MustParseID does what ParseID does, except on error it panics.
func MustParseID(value string) ID {
	id, err := ParseID(value)
	if err != nil {
		panic(err)
	}
	return id
}

func (id ID) queryParam() string {
	if int64(id) == 0 {
		return ""
	}
	return id.String()
}

// Increment increments the new ID value equal to the old one incremented by one.
func (id ID) Increment() ID {
	return ID(int64(id) + 1)
}

func (id ID) String() string {
	return strconv.FormatInt(int64(id), 10)
}

// Username represents a Telegram username.
type Username string

func (username Username) queryParam() string {
	return "@" + string(username)
}

func (username Username) String() string {
	return username.queryParam()
}

// ChatID is either an ID or channel Username in various API calls.
type ChatID interface {
	fmt.Stringer
	queryParam() string
}
