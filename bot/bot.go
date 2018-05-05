package bot

import (
	"io"

	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/logrus"
	"github.com/jfk9w-go/telegram"
)

type (
	Poster interface {
		SendPost(telegram.ChatRef, dvach.Post) error
	}

	Notifier interface {
		NotifyAll([]telegram.ChatRef, string, ...interface{})
	}

	Frontend interface {
		SendText(telegram.ChatRef, string, ...interface{})
		GetAdmins(telegram.ChatRef, telegram.ChatRef) ([]telegram.ChatRef, error)
	}

	Bot interface {
		telegram.Api
		telegram.Updater
		telegram.MessageRouter
	}

	AugmentedBot interface {
		io.Closer
		Bot
		Poster
		Frontend
		Notifier
	}
)

var log = logrus.GetLogger("bot")

const chunkSize = telegram.MaxMessageSize * 9 / 10
