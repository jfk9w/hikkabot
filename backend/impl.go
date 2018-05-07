package backend

import (
	"sync"
	"time"

	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/hikkabot/feed"
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

		for chat, f := range backend.feeds {
			var admins []telegram.ChatRef
			errs := f.Errors()
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

			if f.IsEmpty() {
				delete(backend.feeds, chat)
				log.Infof("Garbage collected feed %s", chat)
				continue
			}
		}

		backend.mu.Unlock()
	}
}

func (backend *T) Subscribe(chat telegram.ChatRef, thread dvach.ID, hash string, offset int) error {
	backend.mu.Lock()
	defer backend.mu.Unlock()

	f, ok := backend.feeds[chat]
	if ok {
		return f.Subscribe(thread, hash, offset)
	}

	f = backend.ff.CreateFeed(chat)
	backend.feeds[chat] = f

	return f.Subscribe(thread, hash, offset)
}

func (backend *T) Unsubscribe(chat telegram.ChatRef, thread dvach.ID) error {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	if f, ok := backend.feeds[chat]; ok {
		return f.Unsubscribe(thread)
	}

	return errors.New("not subscribed")
}

func (backend *T) UnsubscribeAll(chat telegram.ChatRef) error {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	if f, ok := backend.feeds[chat]; ok {
		f.Close()
		delete(backend.feeds, chat)
		log.Infof("Removed %s from feeds", chat)
		return nil
	}

	return errors.New("not subscribed")
}

func (backend *T) Dump(chat telegram.ChatRef) map[dvach.ID]feed.Entry {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	if f, ok := backend.feeds[chat]; ok {
		return f.Dump()
	}

	return map[dvach.ID]feed.Entry{}
}
