package feed

import (
	"strings"

	"github.com/jfk9w-go/dvach"
)

func toKey(id dvach.ID) string {
	return id.Board + "+" + id.Num
}

func fromKey(val string) dvach.ID {
	ts := strings.Split(val, "+")
	return dvach.ID{ts[0], ts[1]}
}

type (
	Entry struct {
		Hash   string
		Offset int
		Error  error
	}

	Error struct {
		Entry Entry
		Err   error
	}

	State = map[dvach.ID]Entry
)
