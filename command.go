package main

import (
	"github.com/boltdb/bolt"
	"github.com/jfk9w/tele2ch/dvach"
	"github.com/jfk9w/tele2ch/telegram"
)

type Command struct {
	Caller  telegram.ChatID
	Handler CommandHandler
}

type CommandHandler interface {
	Execute(ctx Context) error
}

type SubscribeHandler struct {
	ThreadLink string
}

func (h SubscribeHandler) Execute(ctx Context) error {
	key, feed, err := ctx.Dvach().GetThreadFeed(h.ThreadLink, 0)
	if err != nil {
		if err.Error() == dvach.ThreadFeedAlreadyRegistered {
			feed.Stop()
		}
	}
}
