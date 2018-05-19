package feed

import (
	"strings"

	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/hikkabot/text"
)

func toKey(id dvach.Ref) string {
	return id.Board + "+" + id.NumString
}

func fromKey(val string) dvach.Ref {
	ts := strings.Split(val, "+")
	ref, _ := dvach.ToRef(ts[0], ts[1])
	return ref
}

type (
	Entry struct {
		Hashtag text.Hashtag
		Offset  int
		Error   error
	}

	Error struct {
		Entry Entry
		Err   error
	}

	State = map[dvach.Ref]Entry
)
