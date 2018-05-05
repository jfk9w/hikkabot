package backend

import (
	"github.com/jfk9w-go/aconvert"
	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/telegram"
	"github.com/pkg/errors"
)

type (
	key   = telegram.ChatRef
	entry = *Feed
)

type backend struct {
	Bot
	dvach.Api
	aconvert.CacheService
	users map[key]entry
}

func New(bot Bot, dvch dvach.Api, webm aconvert.CacheService) Backend {
	return &backend{bot, dvch, webm, make(map[key]entry)}
}

func (b *backend) gc() {
	for key, value := range b.users {
		if value.IsEmpty() {
			delete(b.users, key)
			log.Infof("Garbage collected user %s", key)
		}
	}
}

func (b *backend) Subscribe(
	chat telegram.ChatRef, admins []telegram.ChatRef,
	thread dvach.Thread, offset int) {

	b.gc()

	feed, ok := b.users[chat]
	if !ok {
		feed = NewFeed(b, b, b, chat)
		b.users[chat] = feed
	}

	if feed.Subscribe(admins, thread, offset) {
		go b.NotifyAll(admins,
			"#info\nSubscription OK.\nChat: %s\nThread: %s\nOffset: 0",
			chat, thread.URL(), offset)

		return
	}

	go b.NotifyAll(admins, "Error: too many subscriptions.")
}

func (b *backend) UnsubscribeAll(chat telegram.ChatRef, admins []telegram.ChatRef) error {
	feed, ok := b.users[chat]
	if !ok {
		return errors.New("not subscribed")
	}

	feed.Close()
	delete(b.users, chat)
	go b.NotifyAll(admins, "#info\nSubscriptions cleared.\nChat: %s", chat)

	return nil
}
