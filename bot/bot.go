package bot

import (
	"github.com/jfk9w-go/httpx"
	"github.com/jfk9w-go/logrus"
	"github.com/jfk9w-go/telegram"
)

type (
	Config struct {
		MaxWait    int           `json:"max_wait"`
		HttpConfig *httpx.Config `json:"http"`
	}

	Bot interface {
		telegram.Api
		telegram.Updater
		telegram.MessageRouter
	}

	Converter interface {
		Get(string) (string, error)
	}

	Storage interface {
		Download(string) error
		Path(string) (string, error)
		Remove(string) error
	}
)

var log = logrus.GetLogger("bot")
