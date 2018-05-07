package backend

import (
	"sync"
	"time"

	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/telegram"
	"github.com/pkg/errors"
)

type T struct {
	bot  Bot
	dvch Dvach
	ff   FeedFactory

	feeds map[telegram.ChatRef]Feed
	mu    *sync.Mutex
}

func Run(bot Bot, dvch dvach.Api, ff FeedFactory) *T {
	backend := &T{bot, dvch, ff, make(map[telegram.ChatRef]Feed), &sync.Mutex{}}
	go backend.maintenance()
	return backend
}

func (backend *T) loadOrDelete(chat telegram.ChatRef) ([]telegram.ChatRef, error) {
	admins, err := backend.bot.GetAdmins(chat)
	if err != nil {
		delete(backend.feeds, chat)
		log.Warningf("Removed %s from feeds because of error: %s", chat, err)
		return nil, err
	}

	return admins, err
}

func (backend *T) maintenance() {
	for {
		time.Sleep(30 * time.Minute)
		backend.mu.Lock()

		for chat, feed := range backend.feeds {
			var admins []telegram.ChatRef
			errs := feed.Errors()
			if len(errs) > 0 {
				var err error
				if admins, err = backend.loadOrDelete(chat); err != nil {
					continue
				}
			}

			for _, err := range errs {
				log.Debugf("Notifying error for %s: %s", chat, err)
				go backend.bot.NotifyAll(admins,
					"#info\nAn error occured.\nChat: %s\nThread: %s\nError: %s",
					chat, err.Thread, err.Err)
			}

			if feed.IsEmpty() {
				delete(backend.feeds, chat)
				log.Infof("Garbage collected user %s", chat)
				continue
			}
		}

		backend.mu.Unlock()
	}
}

func (backend *T) Subscribe(chat telegram.ChatRef, thread dvach.ID, hash string, offset int) error {
	backend.mu.Lock()
	defer backend.mu.Unlock()

	feed, ok := backend.feeds[chat]
	if ok {
		return feed.Subscribe(thread, hash, offset)
	}

	feed = backend.ff.CreateFeed(chat)
	backend.feeds[chat] = feed

	return feed.Subscribe(thread, hash, offset)
}

func (backend *T) Unsubscribe(chat telegram.ChatRef, thread dvach.ID) error {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	if feed, ok := backend.feeds[chat]; ok {
		return feed.Unsubscribe(thread)
	}

	return errors.New("not subscribed")
}

func (backend *T) UnsubscribeAll(chat telegram.ChatRef) error {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	if feed, ok := backend.feeds[chat]; ok {
		feed.Close()
		delete(backend.feeds, chat)
		log.Infof("Removed %s from feeds", chat)
		return nil
	}

	return errors.New("not subscribed")
}
