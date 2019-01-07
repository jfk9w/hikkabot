package service

import (
	"fmt"
	"os"

	"github.com/jfk9w-go/flu"
	. "github.com/jfk9w-go/telegram-bot-api"
)

type FeedUpdate struct {
	text     string
	resource interface{}
	sendOpts SendOpts
	Offset   int64
}

func (update FeedUpdate) Send(b *Bot, chatId ID) error {
	var err error
	if update.resource != nil {
		_, err = b.Send(chatId, update.resource, update.sendOpts)
		if resource, ok := update.resource.(flu.FileSystemResource); ok {
			_ = os.RemoveAll(resource.Path())
		}

		if err != nil {
			_, err = b.Send(chatId, update.text, NewSendOpts().
				ParseMode(HTML).
				DisableNotification(true).
				Message())
		}
	} else {
		_, err = b.Send(chatId, update.text, update.sendOpts)
	}

	return err
}

type Feed struct {
	prefix string
	C      chan FeedUpdate
	I      chan struct{}
	E      chan error
}

func NewFeed(name string) *Feed {
	return &Feed{
		prefix: fmt.Sprintf("#%s\n", name),
		C:      make(chan FeedUpdate, 10),
		I:      make(chan struct{}),
		E:      make(chan error),
	}
}

func (feed *Feed) Error(err error) {
	feed.E <- err
}

func (feed *Feed) Interrupt() {
	feed.I <- struct{}{}
}

func (feed *Feed) format(content string) string {
	return feed.prefix + content
}

func (feed *Feed) SubmitText(text string, disableWebPagePreview bool, offset Offset) bool {
	return feed.submit(FeedUpdate{
		text: feed.format(text),
		sendOpts: NewSendOpts().
			DisableNotification(true).
			ParseMode(HTML).
			Message().
			DisableWebPagePreview(disableWebPagePreview),
		Offset: offset,
	})
}

func (feed *Feed) SubmitPhoto(photo interface{}, caption string, offset Offset) bool {
	formatted := feed.format(caption)
	if resource, ok := photo.(flu.FileSystemResource); ok {
		if stat, err := os.Stat(resource.Path()); err == nil && stat.Size() <= MaxPhotoSize {
			return feed.submit(FeedUpdate{
				text:     formatted,
				resource: photo,
				sendOpts: NewSendOpts().
					DisableNotification(true).
					ParseMode(HTML).
					Media().
					Caption(formatted).
					Photo(),
				Offset: offset,
			})
		}
	}

	return feed.SubmitText(caption, false, offset)
}

func (feed *Feed) SubmitVideo(video interface{}, caption string, offset Offset) bool {
	formatted := feed.format(caption)
	if resource, ok := video.(flu.FileSystemResource); ok {
		if stat, err := os.Stat(resource.Path()); err == nil && stat.Size() <= MaxVideoSize {
			return feed.submit(FeedUpdate{
				text:     formatted,
				resource: video,
				sendOpts: NewSendOpts().
					DisableNotification(true).
					ParseMode(HTML).
					Media().
					Caption(formatted).
					Video(),
				Offset: offset,
			})
		}
	}

	return feed.SubmitText(caption, false, offset)
}

func (feed *Feed) submit(update FeedUpdate) bool {
	select {
	case feed.C <- update:
		return true

	case <-feed.I:
		return false
	}
}

func (feed *Feed) CloseIn() {
	close(feed.C)
	close(feed.E)
}

func (feed *Feed) CloseOut() {
	close(feed.I)
}
