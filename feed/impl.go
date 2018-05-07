package feed

import (
	"sync"

	"time"

	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/hikkabot/html"
	"github.com/jfk9w-go/telegram"
	"github.com/jfk9w-go/unit"
	"github.com/pkg/errors"
)

const maxQueueSize = 100

type T struct {
	aux unit.Aux

	bot  Bot
	dvch Dvach
	conv Converter

	chat  telegram.ChatRef
	queue chan dvach.ID
	err   chan Error

	entries map[dvach.ID]Entry
	mu      *sync.RWMutex
}

func (feed *T) run() {
	log.Infof("Run %s", feed.chat)
	defer log.Infof("Exit %s", feed.chat)

	for {
		select {
		case <-feed.aux.C:
			return

		case thread := <-feed.queue:
			entry, ok := feed.get(thread)
			if !ok {
				log.Debugf("Removed %s from %s queue", thread.URL(), feed.chat)
				continue
			}

			if feed.exec(thread, entry) == unit.ErrInterrupted {
				return
			}
		}
	}
}

func (feed *T) get(thread dvach.ID) (Entry, bool) {
	feed.mu.RLock()
	defer feed.mu.RUnlock()
	entry, ok := feed.entries[thread]
	return entry, ok
}

func (feed *T) delete(thread dvach.ID) {
	feed.mu.Lock()
	defer feed.mu.Unlock()
	delete(feed.entries, thread)
}

func (feed *T) update(thread dvach.ID, offset int) {
	feed.mu.Lock()
	defer feed.mu.Unlock()
	if entry, ok := feed.entries[thread]; ok {
		feed.entries[thread] = entry.WithOffset(offset)
	}
}

func (feed *T) size() int {
	feed.mu.RLock()
	defer feed.mu.RUnlock()
	return len(feed.entries)
}

func (feed *T) exec(thread dvach.ID, entry Entry) error {
	offset := entry.Offset
	if offset != 0 {
		offset++
	}

	posts, err := feed.dvch.Thread(thread, offset)
	if err != nil {
		log.Warningf("Unable to get new %s posts for %s: %s", thread.URL(), feed.chat, err)
		feed.delete(thread)
		feed.err <- Error{thread, entry.Hash, err}
		return err
	}

	for _, post := range posts {
		for _, file := range post.Files {
			if file.Type == dvach.Webm {
				go feed.conv.Convert(file.URL(), nil)
			}
		}
	}

	log.Debugf("%d new posts from %s for %s", len(posts), thread.URL(), feed.chat)

	offset = entry.Offset
	for i, post := range posts {
		if feed.interrupted() {
			return unit.ErrInterrupted
		}

		if err := feed.bot.SendPost(feed.chat, html.Post{post, thread.Board, entry.Hash}); err != nil {
			log.Debugf("Failed to send post from %s to %s: %s", thread.URL(), feed.chat, err)
			feed.delete(thread)
			return err
		}

		if i%5 == 0 {
			if _, ok := feed.get(thread); !ok {
				log.Debugf("Interrupting %s for %s", thread.URL(), feed.chat)
				return nil
			}
		}

		offset = post.NumInt()
	}

	if len(posts) > 0 {
		feed.update(thread, offset)
		log.Debugf("Updated offset %d for %s in %s", offset, thread.URL(), feed.chat)
	}

	feed.queue <- thread
	time.Sleep(2 * time.Minute)

	return nil
}

func (feed *T) interrupted() bool {
	select {
	case <-feed.aux.C:
		return true

	default:
		return false
	}
}

func (feed *T) Subscribe(thread dvach.ID, hash string, offset int) error {
	feed.mu.Lock()
	defer feed.mu.Unlock()

	if _, ok := feed.entries[thread]; ok {
		return errors.New("exists")
	}

	if len(feed.entries) >= maxQueueSize {
		return errors.New("too many subscriptions")
	}

	feed.entries[thread] = Entry{offset, hash}
	feed.queue <- thread

	log.Infof("Subscribed %s to %s with offset %d", feed.chat, thread.URL(), offset)
	return nil
}

func (feed *T) Unsubscribe(thread dvach.ID) error {
	if _, ok := feed.get(thread); !ok {
		return errors.New("not subscribed")
	}

	feed.mu.Lock()
	defer feed.mu.Unlock()

	delete(feed.entries, thread)

	log.Infof("Unsubscribed %s from %s", feed.chat, thread.URL())
	return nil
}

func (feed *T) IsEmpty() bool {
	return feed.size() == 0
}

func (feed *T) Errors() []Error {
	errs := make([]Error, 0)
	for {
		select {
		case err := <-feed.err:
			errs = append(errs, err)
		default:
			break
		}
	}

	return errs
}

func (feed *T) Close() error {
	return feed.aux.Close()
}

func (feed *T) Dump() map[dvach.ID]Entry {
	feed.mu.RLock()
	defer feed.mu.RUnlock()

	r := make(map[dvach.ID]Entry)
	for k, v := range feed.entries {
		r[k] = v
	}

	return r
}
