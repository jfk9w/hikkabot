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
		Subscribe(telegram.ChatRef, []telegram.ChatRef, dvach.Thread, int)
		UnsubscribeAll(telegram.ChatRef, []telegram.ChatRef) error
	}
)

var log = logrus.GetLogger("backend")
