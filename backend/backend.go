package backend

import (
	"io"

	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/hikkabot/feed"
	"github.com/jfk9w-go/hikkabot/html"
	"github.com/jfk9w-go/logrus"
	"github.com/jfk9w-go/telegram"
)

type (
	Feed interface {
		io.Closer
		Subscribe(dvach.ID, string, int) error
		Unsubscribe(dvach.ID) error
		IsEmpty() bool
		Errors() []feed.Error
	}

	FeedFactory interface {
		CreateFeed(telegram.ChatRef) Feed
	}

	Bot interface {
		DeleteRoute(telegram.ChatRef)
		SendPost(telegram.ChatRef, html.Post) error
		GetAdmins(telegram.ChatRef) ([]telegram.ChatRef, error)
		NotifyAll([]telegram.ChatRef, string, ...interface{})
	}

	Dvach interface {
		Thread(dvach.ID, int) ([]dvach.Post, error)
		Post(dvach.ID) (*dvach.Post, error)
	}
)

var log = logrus.GetLogger("backend")

func NewFeedFactory(bot feed.Bot, dvch feed.Dvach, conv feed.Converter) FeedFactory {
	return &DefaultFeedFactory{bot, dvch, conv}
}

type DefaultFeedFactory struct {
	bot  feed.Bot
	dvch feed.Dvach
	conv feed.Converter
}

func (df *DefaultFeedFactory) CreateFeed(chat telegram.ChatRef) Feed {
	return feed.Run(df.bot, df.dvch, df.conv, chat)
}
