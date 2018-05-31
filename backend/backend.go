package backend

import (
	"io"

	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/hikkabot/feed"
	"github.com/jfk9w-go/hikkabot/keeper"
	"github.com/jfk9w-go/hikkabot/text"
	"github.com/jfk9w-go/logrus"
	"github.com/jfk9w-go/telegram"
	"github.com/orcaman/concurrent-map"
)

type (
	Feed interface {
		io.Closer
		Subscribe(dvach.Ref, string, int) bool
		Unsubscribe(dvach.Ref)
		Running() feed.State
		CollectErrors() (bool, []error)
	}

	FeedFactory interface {
		CreateFeed(telegram.ChatRef) Feed
	}

	Bot interface {
		DeleteRoute(telegram.ChatRef)
		SendPost(telegram.ChatRef, text.Post) error
		GetAdmins(telegram.ChatRef) ([]telegram.ChatRef, error)
		NotifyAll([]telegram.ChatRef, string, ...interface{})
	}

	Dvach interface {
		Thread(dvach.Ref, int) ([]*dvach.Post, error)
		Post(dvach.Ref) (*dvach.Post, error)
		Path(*dvach.File) (string, error)
	}
)

var log = logrus.GetLogger("backend")

func Run(bot Bot, ff FeedFactory) *T {
	back := &T{
		bot:   bot,
		ff:    ff,
		state: cmap.New(),
	}

	return back
}

func NewFeedFactory(bot feed.Bot, dvch feed.Dvach, db keeper.T) FeedFactory {
	return &DefaultFeedFactory{bot, dvch, db}
}

type DefaultFeedFactory struct {
	bot  feed.Bot
	dvch feed.Dvach
	db   keeper.T
}

func (df *DefaultFeedFactory) CreateFeed(chat telegram.ChatRef) Feed {
	return feed.Run(df.bot, df.dvch, df.db, chat)
}
