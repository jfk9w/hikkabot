package api

import (
	"regexp"
	"strconv"

	"github.com/pkg/errors"
)

type MessageID = int

type Ref interface {
	Value() string
}

type ChatID int64

func ParseChatID(value string) (ChatID, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	return ChatID(id), err
}

func (id ChatID) Value() string {
	return strconv.FormatInt(int64(id), 10)
}

type Username string

var usernameRegexp = regexp.MustCompile("^@[A-Za-z0-9_]+$")

func ParseUsername(value string) (Username, error) {
	if usernameRegexp.MatchString(value) {
		return Username(value), nil
	}

	return Username(""), errors.Errorf("invalid username: %s", value)
}

func (id Username) Value() string {
	return string(id)
}

func (id Username) IsDefined() bool {
	return string(id) != ""
}
