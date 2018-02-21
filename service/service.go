package service

import (
	"strings"
	"time"

	"github.com/jfk9w/hikkabot/telegram"
)

type (
	AccountID = string
	ThreadID  = string

	Config struct {
		ThreadTTL time.Duration `json:"thread_ttl"`
		WebmTTL   time.Duration `json:"webm_ttl"`
	}

	State = map[AccountID]map[ThreadID]int

	Storage interface {
		Load() (State, error)
		InsertThread(AccountID, ThreadID) bool
		DeleteThread(AccountID, ThreadID)
		DeleteAccount(AccountID)
		GetOffset(AccountID, ThreadID) int
		UpdateOffset(AccountID, ThreadID, int) bool
	}
)

func GetThreadID(board string, thread string) ThreadID {
	return board + path2 + thread
}

func ReadThreadID(id ThreadID) (string, string) {
	ts := strings.Split(id, path2)
	return ts[0], ts[1]
}

func GetAccountID(chat telegram.ChatRef) AccountID {
	return chat.Key()
}

func ReadAccountID(id AccountID) telegram.ChatRef {
	if strings.HasPrefix(id, "@") {
		return telegram.ChatRef{
			Username: id,
		}
	}

	id0 := telegram.ParseChatID(id)
	return telegram.ChatRef{
		ID: id0,
	}
}
