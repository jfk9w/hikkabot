package service

import (
	"os"

	"github.com/jfk9w-go/flu"
	telegram "github.com/jfk9w-go/telegram-bot-api"
)

type Update interface {
	Send(*telegram.Bot, telegram.ID) (*telegram.Message, error)
}

type UpdateFunc func(*telegram.Bot, telegram.ID) (*telegram.Message, error)

func (f UpdateFunc) Send(bot *telegram.Bot, chatID telegram.ID) (*telegram.Message, error) {
	return f(bot, chatID)
}

type UpdateBatch interface {
	Get(chan<- Update)
}

type UpdateBatchFunc func(chan<- Update)

func (f UpdateBatchFunc) Get(updateCh chan<- Update) {
	f(updateCh)
	close(updateCh)
}

type offsetUpdateBatch struct {
	offset int64
	UpdateBatch
}

type UpdateType uint8

const (
	TextUpdate UpdateType = iota
	PhotoUpdate
	VideoUpdate
	TextPreviewUpdate
)

func (t UpdateType) params(u *GenericUpdate) (interface{}, telegram.SendOpts) {
	switch t {
	case TextUpdate:
		return u.Text, telegram.NewSendOpts().
			DisableNotification(true).
			ParseMode(telegram.HTML).
			Message().
			DisableWebPagePreview(true)

	case PhotoUpdate:
		return u.Entity, telegram.NewSendOpts().
			DisableNotification(true).
			ParseMode(telegram.HTML).
			Media().
			Caption(u.Text).
			Photo()

	case VideoUpdate:
		return u.Entity, telegram.NewSendOpts().
			DisableNotification(true).
			ParseMode(telegram.HTML).
			Media().
			Caption(u.Text).
			Video()

	case TextPreviewUpdate:
		return u.Text, telegram.NewSendOpts().
			DisableNotification(true).
			ParseMode(telegram.HTML).
			Message()

	default:
		panic("invalid update type")
	}
}

type GenericUpdate struct {
	Text   string
	Entity interface{}
	Type   UpdateType
}

func (u *GenericUpdate) Send(bot *telegram.Bot, chatID telegram.ID) (*telegram.Message, error) {
	entity, sendOpts := u.Type.params(u)
	m, err := bot.Send(chatID, entity, sendOpts)
	if u.Type != TextUpdate {
		if fsr, ok := entity.(flu.FileSystemResource); ok {
			_ = os.RemoveAll(fsr.Path())
		}

		if err != nil {
			entity, sendOpts = TextPreviewUpdate.params(u)
			m, err = bot.Send(chatID, entity, sendOpts)
		}
	}

	return m, err
}

func (u *GenericUpdate) Get(updateCh chan<- Update) {
	updateCh <- u
	close(updateCh)
}

type UpdatePipe struct {
	updateCh chan offsetUpdateBatch
	stopCh   chan struct{}
	errCh    chan error
}

func NewUpdatePipe() *UpdatePipe {
	return &UpdatePipe{
		updateCh: make(chan offsetUpdateBatch, 10),
		stopCh:   make(chan struct{}),
		errCh:    make(chan error),
	}
}

func (feed *UpdatePipe) Error(err error) {
	feed.errCh <- err
}

func (feed *UpdatePipe) stop() {
	feed.stopCh <- struct{}{}
}

func (feed *UpdatePipe) Submit(updateBatch UpdateBatch, offset int64) bool {
	select {
	case feed.updateCh <- offsetUpdateBatch{offset, updateBatch}:
		return true

	case <-feed.stopCh:
		return false
	}
}

func (feed *UpdatePipe) Close() {
	close(feed.updateCh)
	close(feed.errCh)
}

func (feed *UpdatePipe) closeOut() {
	close(feed.stopCh)
}
