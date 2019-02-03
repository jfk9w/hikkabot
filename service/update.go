package service

import (
	"log"
	"os"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/html"
	"golang.org/x/exp/utf8string"
)

type UpdateText interface {
	Get(gmf GetMessageFunc) []string
}

type UpdateTextFunc func(gmf GetMessageFunc) []string

func (f UpdateTextFunc) Get(gmf GetMessageFunc) []string {
	return f(gmf)
}

type UpdateTextSlice []string

func (s UpdateTextSlice) Get(gmf GetMessageFunc) []string {
	return s
}

type OnUpdateCompleteFunc func(*telegram.Message)

type Update struct {
	Offset    int64
	Text      UpdateText
	MediaSize int
	Media     <-chan MediaResponse
	Key       MessageKey
}

const maxCaptionSize = telegram.MaxCaptionSize * 5 / 7

func (u Update) Send(bot *telegram.Bot, gmf GetMessageFunc) (*telegram.Message, error) {
	text := u.Text.Get(gmf)
	collapse :=
		u.MediaSize == 1 &&
			len(text) == 1 &&
			utf8string.NewString(text[0]).RuneCount() <= maxCaptionSize

	var firstm *telegram.Message
	if !collapse {
		for _, part := range text {
			m, err := u.send(bot, chatID, part, nil)
			if err != nil {
				return nil, err
			}

			if firstm == nil {
				firstm = m
			}
		}
	}

	if u.MediaSize > 0 {
		for resp := range u.Media {
			var media *Media
			if resp.Err != nil {
				log.Printf("%s download failed: %s", resp.Media.Href, resp.Err)
			} else {
				media = &resp.Media
			}

			if collapse {
				return u.send(bot, chatID, text[0], media)
			}

			m, err := u.send(bot, chatID, "", media)
			if err != nil {
				return nil, err
			}

			if firstm == nil {
				firstm = m
			}
		}
	}

	return firstm, nil
}

func (u Update) send(bot *telegram.Bot, chatID telegram.ID, text string, media *Media) (*telegram.Message, error) {
	if media != nil {
		text = html.Link(media.Href, "[ATTACH]") + "\n" + text
		opts := telegram.NewSendOpts().
			DisableNotification(true).
			Media().
			Caption(text).
			ParseMode(telegram.HTML)

		if media.Type == Photo {
			opts.Photo()
		} else {
			opts.Video()
		}

		m, err := bot.Send(chatID, media.Resource, opts)
		_ = os.RemoveAll(media.Resource.Path())
		if err != nil {
			log.Printf("failed to send %s: %s", media.Href, err)
		} else {
			return m, nil
		}
	}

	opts := telegram.NewSendOpts().
		DisableNotification(true).
		Message().
		ParseMode(telegram.HTML).
		DisableWebPagePreview(media == nil)

	return bot.Send(chatID, text, opts)
}

type UpdatePipe struct {
	updateCh chan Update
	stopCh   chan struct{}
	Err      error
}

func NewUpdatePipe() *UpdatePipe {
	return &UpdatePipe{
		updateCh: make(chan Update, 10),
		stopCh:   make(chan struct{}),
	}
}

func (p *UpdatePipe) stop() {
	p.stopCh <- struct{}{}
}

func (p *UpdatePipe) Submit(update Update) bool {
	select {
	case p.updateCh <- update:
		return true

	case <-p.stopCh:
		return false
	}
}

func (p *UpdatePipe) Close() {
	close(p.updateCh)
}

func (p *UpdatePipe) closeOut() {
	close(p.stopCh)
}
