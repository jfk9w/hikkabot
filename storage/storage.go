package storage

import (
	"strings"
	"time"

	"github.com/jfk9w/hikkabot/telegram"
)

type (
	AccountID = telegram.ChatRef
	ThreadID  = [2]string

	AccountKey = string
	ThreadKey  = string

	Config struct {
		SubscriptionTTL time.Duration `json:"subscription_ttl"`
	}

	State = map[AccountKey]map[ThreadKey]int

	T interface {
		SelectAll() (State, error)
		Resume(AccountID, ThreadID) error
		Suspend(AccountID, ThreadID) error
		SuspendAll(AccountID) error
		IsActive(AccountID, ThreadID) (bool, error)
		Update(AccountID, ThreadID, int) error
	}
)

const (
	threadKeySeparator = "/"
)

func NewThreadKey(id ThreadID) ThreadKey {
	return id[0] + threadKeySeparator + id[1]
}

func ParseThreadKey(key ThreadKey) ThreadID {
	ts := strings.Split(key, threadKeySeparator)
	return [2]string{ts[0], ts[1]}
}

func NewAccountKey(id AccountID) AccountKey {
	return id.Key()
}

func ParseAccountKey(key AccountKey) AccountID {
	if strings.HasPrefix(key, "@") {
		return telegram.ChatRef{
			Username: key,
		}
	}

	id := telegram.ParseChatID(key)
	return telegram.ChatRef{
		ID: id,
	}
}
