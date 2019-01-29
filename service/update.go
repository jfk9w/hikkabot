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
			Message().
			ParseMode(telegram.HTML).
			DisableWebPagePreview(true)

	case PhotoUpdate:
		return u.Entity, telegram.NewSendOpts().
			DisableNotification(true).
			Media().Photo().
			ParseMode(telegram.HTML).
			Caption(u.Text)

	case VideoUpdate:
		return u.Entity, telegram.NewSendOpts().
			DisableNotification(true).
			Media().Video().
			ParseMode(telegram.HTML).
			Caption(u.Text)

	case TextPreviewUpdate:
		return u.Text, telegram.NewSendOpts().
			DisableNotification(true).
			Message().
			ParseMode(telegram.HTML)

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
	Error    error
}

func NewUpdatePipe() *UpdatePipe {
	return &UpdatePipe{
		updateCh: make(chan offsetUpdateBatch, 10),
		stopCh:   make(chan struct{}),
	}
}

func (p *UpdatePipe) stop() {
	p.stopCh <- struct{}{}
}

func (p *UpdatePipe) Submit(updateBatch UpdateBatch, offset int64) bool {
	select {
	case p.updateCh <- offsetUpdateBatch{offset, updateBatch}:
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
