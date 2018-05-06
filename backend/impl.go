package backend

import (
	"regexp"

	"strings"

	"strconv"

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

var postHashtagRegex = regexp.MustCompile(`#?([A-Za-z]+)(\d+)`)

func (b *backend) ParseID(value string) (*dvach.ID, int, error) {
	thread, offset, err := dvach.ParseThread(value)
	if err != nil {
		groups := postHashtagRegex.FindSubmatch([]byte(value))
		if len(groups) == 3 {
			thread := &dvach.ID{
				Board: strings.ToLower(string(groups[1])),
				Num:   string(groups[2]),
			}

			post, err := b.Post(*thread)
			if err != nil {
				log.Warningf("Unable to load post %s: %s")
				return nil, 0, err
			}

			if post.Parent != "0" {
				offset, _ = strconv.Atoi(thread.Num)
				thread.Num = post.Parent
			}

			return thread, offset, nil
		}

		return nil, 0, err
	}

	return thread, offset, nil
}

func (b *backend) Subscribe(
	chat telegram.ChatRef, admins []telegram.ChatRef,
	thread dvach.ID, offset int) {

	b.gc()

	feed, ok := b.users[chat]
	if !ok {
		feed = NewFeed(b, b, b, chat)
		b.users[chat] = feed
	}

	if feed.Submit(thread, offset, admins) {
		go b.NotifyAll(admins,
			"#info\nSubscription OK.\nChat: %s\nThread: %s\nOffset: %d",
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
