package feed

import (
	"time"

	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/hikkabot/text"
	"github.com/jfk9w-go/telegram"
	"github.com/jfk9w-go/unit"
	"github.com/orcaman/concurrent-map"
	"github.com/pkg/errors"
)

type T struct {
	aux   unit.Aux
	bot   Bot
	dvch  Dvach
	conv  Converter
	chat  telegram.ChatRef
	state cmap.ConcurrentMap
}

func (feed *T) run() {
	for {
		if feed.intr() {
			return
		}

		keys := feed.state.Keys()
		for _, key := range keys {
			value, ok := feed.state.Get(key)
			if !ok {
				continue
			}

			entry, ok := value.(*Entry)
			if entry.Error != nil {
				continue
			}

			err := feed.execute(key, entry)
			switch err {
			case nil:
				continue

			case unit.ErrInterrupted:
				return

			default:
				if entry, ok := feed.state.Get(key); ok {
					entry.(*Entry).Error = err
				}
			}
		}

		if !feed.sleep() {
			return
		}
	}
}

func (feed *T) intr() bool {
	select {
	case <-feed.aux.C:
		return true

	default:
		return false
	}
}

func (feed *T) sleep() bool {
	timer := time.NewTimer(2 * time.Minute)
	select {
	case <-feed.aux.C:
		return false

	case <-timer.C:
		return true
	}
}

func (feed *T) update(key string, offset int) bool {
	if entry, ok := feed.state.Get(key); ok {
		entry.(*Entry).Offset = offset
		return true
	}

	return false
}

func (feed *T) preload(posts []*dvach.Post) {
	for _, post := range posts {
		for _, file := range post.Files {
			if file.Type == dvach.Webm {
				var url string
				if file.IsProxied() {
					url = file.ProxiedURL
				} else {
					url = file.URL()
				}

				go feed.conv.Convert(url, nil)
			}
		}
	}
}

func (feed *T) execute(key string, entry *Entry) error {
	if feed.intr() {
		return unit.ErrInterrupted
	}

	ref := fromKey(key)
	offset := entry.Offset
	if offset > 0 {
		offset++
	}

	posts, err := feed.dvch.Posts(ref, offset)
	if err != nil {
		log.Warningf("Unable to load posts from %s for %s: %s", ref, feed.chat, err)
		return errors.Errorf("unable to load posts from %s: %s", text.FormatRef(ref), err)
	}

	if len(posts) == 0 {
		return nil
	}

	log.Debugf("%d new posts for %s from %s", len(posts), feed.chat, ref)

	if feed.intr() {
		return unit.ErrInterrupted
	}

	feed.preload(posts)

	for _, post := range posts {
		if feed.intr() {
			return unit.ErrInterrupted
		}

		if err := feed.bot.SendPost(feed.chat, text.Post{post, entry.Hashtag}); err != nil {
			log.Warningf("Unable to send post %s/%s to %s: %s", ref, post.Ref, feed.chat, err)
			return errors.Errorf("unable to send post %s from %s: %s",
				text.FormatRef(post.Ref), text.FormatRef(ref), err)
		}

		if !feed.update(key, post.Num) {
			return nil
		}
	}

	if !feed.sleep() {
		return unit.ErrInterrupted
	}

	return nil
}

func (feed *T) Subscribe(thread dvach.Ref, hash string, offset int) bool {
	return feed.state.SetIfAbsent(toKey(thread), &Entry{hash, offset, nil})
}

func (feed *T) Unsubscribe(thread dvach.Ref) {
	feed.state.Remove(toKey(thread))
}

func (feed *T) Close() error {
	return feed.aux.Close()
}

func (feed *T) CollectErrors() (bool, []error) {
	empty := true
	errs := make([]error, 0)
	for k, v := range feed.state.Items() {
		entry := *v.(*Entry)
		if entry.Error == nil {
			empty = false
		} else {
			errs = append(errs, entry.Error)
			feed.state.Remove(k)
		}
	}

	return empty, errs
}

func (feed *T) Running() State {
	r := State{}
	for k, v := range feed.state.Items() {
		entry := *v.(*Entry)
		if entry.Error == nil {
			r[fromKey(k)] = *v.(*Entry)
		}
	}

	return r
}
