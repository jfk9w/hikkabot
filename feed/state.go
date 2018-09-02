package feed

import (
	"encoding/json"
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

type Type string

const (
	DvachType  Type = "dvach"
	RedditType Type = "reddit"
)

type Offset = string

type State struct {
	ID     string
	Type   Type
	Meta   json.RawMessage
	Offset Offset
	Error  error
}

func (s State) ParseMeta(v interface{}) error {
	return json.Unmarshal(s.Meta, v)
}

func (s State) WithOffset(offset Offset) State {
	s.Offset = offset
	return s
}

func (s State) WithError(err error) State {
	s.Error = err
	return s
}

func (s State) Err() string {
	if s.Error == nil {
		return ""
	}

	return s.Error.Error()
}
