package feed

import (
	"encoding/json"

	"github.com/jfk9w-go/telegram"
)

type DvachMode = string

const (
	FullDvachMode  DvachMode = "full"
	MediaDvachMode DvachMode = "media"
)

type DvachMeta struct {
	Title string    `json:"title"`
	Mode  DvachMode `json:"mode"`
}

type DvachWatchMeta struct {
	Query []string `json:"query"`
}

type RedMeta struct {
	Mode string `json:"mode"`
	Ups  int    `json:"ups"`
}

type Type string

const (
	DvachType      Type = "dvach"
	DvachWatchType Type = "dvach_watch"
	RedType        Type = "red"
)

type Offset = int

type State struct {
	Chat    telegram.ChatID
	ID      string
	Type    Type
	Meta    json.RawMessage
	Offset  Offset
	Error   error
	Updated int64
}

func (s *State) ParseMeta(v interface{}) error {
	return json.Unmarshal(s.Meta, v)
}

func (s *State) WithOffset(offset Offset) *State {
	s.Offset = offset
	return s
}

func (s *State) WithError(err error) *State {
	s.Error = err
	return s
}

func (s *State) Err() string {
	if s.Error == nil {
		return ""
	}

	return s.Error.Error()
}
