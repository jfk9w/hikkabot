package state

import (
	"fmt"
	"strings"
	"time"
)

type ThreadKey string

func getThreadKey(board string, threadId string) ThreadKey {
	return ThreadKey(fmt.Sprintf("%s/%s", board, threadId))
}

func parseThreadKey(key ThreadKey) (string, string) {
	tokens := strings.Split(string(key), "/")
	return tokens[0], tokens[1]
}

type InactiveThread struct {
	Offset    int       `json:"offset"`
	StoppedAt time.Time `json:"stopped_at"`
}

func newInactiveThread(offset int) InactiveThread {
	return InactiveThread{
		Offset:    offset,
		StoppedAt: time.Now(),
	}
}
