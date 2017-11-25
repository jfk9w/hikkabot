package service

import (
	"strings"

	"github.com/jfk9w/hikkabot/telegram"
)

// ThreadKey is a string "<board>/<thread_id>"
type ThreadKey string

// ParseThreadKey to the board name and thread ID
func ParseThreadKey(id ThreadKey) (string, string) {
	tokens := strings.Split(string(id), "/")
	return tokens[0], tokens[1]
}

// FormatThreadKey concatenates board name and thread ID to form a ThreadKey
func FormatThreadKey(board string, threadID string) ThreadKey {
	return ThreadKey(board + "/" + threadID)
}

// SubscriberKey is either a Telegram chat ID or a channel name starting with `@`
type SubscriberKey string

// ParseSubscriberKey transforms a SubscriberKey to a ChatRef
func ParseSubscriberKey(key SubscriberKey) telegram.ChatRef {
	key0 := string(key)
	if strings.HasPrefix(key0, `@`) {
		return telegram.ChatRef{
			Username: key0,
		}
	}

	return telegram.ChatRef{
		ID: telegram.ParseChatID(key0),
	}
}

// FormatSubscriberKey creates a SubscriberKey from a ChatRef
func FormatSubscriberKey(chat telegram.ChatRef) SubscriberKey {
	return SubscriberKey(chat.Key())
}
