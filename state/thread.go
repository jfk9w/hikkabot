package state

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	. "github.com/jfk9w/hikkabot/util"
)

type ThreadKey string

func getThreadKey(board string, threadId string) ThreadKey {
	return ThreadKey(fmt.Sprintf("%s/%s", board, threadId))
}

func parseThreadKey(key ThreadKey) (string, string) {
	tokens := strings.Split(string(key), "/")
	return tokens[0], tokens[1]
}

type ActiveThread struct {
	offset int
	halt   Signal
	done   Signal
}

func newThread() ActiveThread {
	return ActiveThread{
		halt: NewSignal(),
		done: NewSignal(),
	}
}

func (t ActiveThread) suspend() InactiveThread {
	return InactiveThread{
		Offset:    t.offset,
		StoppedAt: time.Now(),
	}
}

func (t *ActiveThread) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.offset)
}

func (t *ActiveThread) UnmarshalJSON(raw []byte) error {
	err := json.Unmarshal(raw, &t.offset)
	if err != nil {
		return err
	}

	t.halt = NewSignal()
	t.done = NewSignal()

	return nil
}

type InactiveThread struct {
	Offset    int       `json:"offset"`
	StoppedAt time.Time `json:"stopped_at"`
}

func (t InactiveThread) resume() ActiveThread {
	return ActiveThread{
		offset: t.Offset,
		halt:   NewSignal(),
		done:   NewSignal(),
	}
}
