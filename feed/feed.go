package feed

import (
	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/hikkabot/keeper"
	"github.com/jfk9w-go/hikkabot/text"
	"github.com/jfk9w-go/logrus"
	"github.com/jfk9w-go/telegram"
	"github.com/jfk9w-go/unit"
	"github.com/orcaman/concurrent-map"
)

type (
	Bot interface {
		SendPost(telegram.ChatRef, text.Post) error
	}

	Dvach interface {
		Posts(dvach.Ref, int) ([]*dvach.Post, error)
	}
)

var log = logrus.GetLogger("feed")

func Run(bot Bot, dvch Dvach, db keeper.T, chat telegram.ChatRef) *T {
	feed := &T{
		aux:   unit.NewAux(),
		bot:   bot,
		dvch:  dvch,
		db:    db,
		chat:  chat,
		state: cmap.New(),
	}

	go feed.run()
	return feed
}
