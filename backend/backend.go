package backend

import (
	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/hikkabot/bot"
	"github.com/jfk9w-go/logrus"
	"github.com/jfk9w-go/telegram"
)

type (
	Bot interface {
		bot.Poster
		bot.Notifier
	}

	Backend interface {
		ParseID(string) (*dvach.ID, int, error)
		Subscribe(telegram.ChatRef, []telegram.ChatRef, dvach.ID, int)
		UnsubscribeAll(telegram.ChatRef, []telegram.ChatRef) error
	}
)

var log = logrus.GetLogger("backend")
