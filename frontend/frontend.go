package frontend

import (
	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/hikkabot/feed"
	"github.com/jfk9w-go/hikkabot/text"
	"github.com/jfk9w-go/logrus"
	"github.com/jfk9w-go/telegram"
)

type (
	Backend interface {
		Subscribe(telegram.ChatRef, dvach.Ref, string, int) error
		Unsubscribe(telegram.ChatRef, dvach.Ref) error
		UnsubscribeAll(telegram.ChatRef) error
		Dump(telegram.ChatRef) (feed.State, error)
	}

	Bot interface {
		GetMe() (*telegram.User, error)
		UpdateChannel() <-chan telegram.Update
		SendText(telegram.ChatRef, string, ...interface{})
		GetAdmins(telegram.ChatRef) ([]telegram.ChatRef, error)
		NotifyAll([]telegram.ChatRef, string, ...interface{})
		SendPost(telegram.ChatRef, text.Post) error
		SendPopular(telegram.ChatRef, []*dvach.Thread, []string) error
	}

	Dvach interface {
		Catalog(dvach.Board) (*dvach.Catalog, error)
		Post(dvach.Ref) (*dvach.Post, error)
		Thread(dvach.Ref) (*dvach.Thread, error)
	}
)

func New(bot Bot, dvch Dvach, back Backend) *T {
	return &T{bot, dvch, back}
}

var log = logrus.GetLogger("frontend")
