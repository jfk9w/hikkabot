package feed

import (
	"strings"

	"github.com/jfk9w-go/dvach"
)

func toKey(id dvach.Ref) string {
	return id.Board + "+" + id.Num
}

func fromKey(val string) dvach.Ref {
	ts := strings.Split(val, "+")
	return dvach.Ref{ts[0], ts[1]}
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

	State = map[dvach.Ref]Entry
)
