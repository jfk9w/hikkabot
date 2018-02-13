package storage

import (
	"strings"
	"time"

	"github.com/jfk9w/hikkabot/telegram"
)

type (
	AccountID = string
	ThreadID  = string

	Config struct {
		SubscriptionTTL time.Duration `json:"subscription_ttl"`
	}

	State = map[AccountID]map[ThreadID]int

	T interface {
		DumpState() (State, error)
		Resume(AccountID, ThreadID) error
		Suspend(AccountID, ThreadID) error
		SuspendAll(AccountID) error
		IsActive(AccountID, ThreadID) (bool, error)
		Update(AccountID, ThreadID, int) error
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
