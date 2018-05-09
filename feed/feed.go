package feed

import (
	"github.com/jfk9w-go/aconvert"
	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/hikkabot/html"
	"github.com/jfk9w-go/logrus"
	"github.com/jfk9w-go/telegram"
	"github.com/jfk9w-go/unit"
	"github.com/orcaman/concurrent-map"
)

type (
	Bot interface {
		SendPost(telegram.ChatRef, html.Post) error
	}

	Dvach interface {
		Thread(dvach.ID, int) ([]dvach.Post, error)
	}

	Converter interface {
		Convert(string, chan aconvert.VideoResponse)
	}
)

var log = logrus.GetLogger("feed")

func Run(bot Bot, dvch Dvach, conv Converter, chat telegram.ChatRef) *T {
	feed := &T{
		aux:   unit.NewAux(),
		bot:   bot,
		dvch:  dvch,
		conv:  conv,
		chat:  chat,
		state: cmap.New(),
	}

	go feed.run()
	return feed
}
