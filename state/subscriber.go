package state

import (
	"strings"

	"github.com/jfk9w/hikkabot/telegram"
)

type SubscriberKey string

func getSubscriberKey(chat telegram.ChatRef) SubscriberKey {
	if len(chat.Username) > 0 {
		return SubscriberKey(chat.Username)
	} else {
		return SubscriberKey(telegram.FormatChatID(chat.ID))
	}
}

func parseSubscriberKey(key SubscriberKey) telegram.ChatRef {
	str := string(key)
	if strings.HasPrefix(str, "@") {
		return telegram.ChatRef{
			Username: str,
		}
	} else {
		return telegram.ChatRef{
			ID: telegram.ParseChatID(str),
		}
	}
}

type Subscriber struct {
	Active   map[ThreadKey]ActiveThread   `json:"active"`
	Inactive map[ThreadKey]InactiveThread `json:"inactive"`
}

func newSubscriber() Subscriber {
	return Subscriber{
		Active:   make(map[ThreadKey]ActiveThread),
		Inactive: make(map[ThreadKey]InactiveThread),
	}
}
