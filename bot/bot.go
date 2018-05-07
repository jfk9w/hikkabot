package bot

import (
	"github.com/jfk9w-go/logrus"
	"github.com/jfk9w-go/telegram"
)

type (
	Bot interface {
		telegram.Api
		telegram.Updater
		telegram.MessageRouter
	}

	Converter interface {
		Get(string) (string, error)
	}
)

var log = logrus.GetLogger("bot")

const chunkSize = telegram.MaxMessageSize * 9 / 10
