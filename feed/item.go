package feed

import (
	"github.com/jfk9w-go/httpx"
	"github.com/jfk9w-go/telegram"
)

type Event interface {
	Interrupted()
}

type Item interface {
	Event
	Send(*telegram.T, telegram.ChatID) error
	Retry(*telegram.T, telegram.ChatID) error
}

var SendOpts = &telegram.SendOpts{
	ParseMode:           telegram.HTML,
	DisableNotification: true,
}

type TextItem struct {
	Text                  string
	DisableWebPagePreview bool
}

func (_ *TextItem) Interrupted() {

}

func (item *TextItem) Send(api *telegram.T, chat telegram.ChatID) error {
	var opts = &telegram.MessageOpts{
		SendOpts:              SendOpts,
		DisableWebPagePreview: item.DisableWebPagePreview,
	}

	var _, err = api.SendMessage(chat, item.Text, opts)
	return err
}

func (item *TextItem) Retry(api *telegram.T, chat telegram.ChatID) error {
	return item.Send(api, chat)
}

type ImageItem struct {
	File    *httpx.File
	Caption string
}

func (item *ImageItem) Interrupted() {
	item.File.Delete()
}

func (item *ImageItem) Send(api *telegram.T, chat telegram.ChatID) error {
	var opts = &telegram.MediaOpts{
		SendOpts: SendOpts,
		Caption:  item.Caption,
	}

	var _, err = api.SendPhoto(chat, item.File, opts)
	item.File.Delete()

	return err
}

func (item *ImageItem) Retry(api *telegram.T, chat telegram.ChatID) error {
	return (&TextItem{Text: item.Caption}).Send(api, chat)
}

type VideoItem struct {
	File    *httpx.File
	Caption string
}

func (item *VideoItem) Interrupted() {
	item.File.Delete()
}

func (item *VideoItem) Send(api *telegram.T, chat telegram.ChatID) error {
	var opts = &telegram.VideoOpts{
		MediaOpts: &telegram.MediaOpts{
			SendOpts: SendOpts,
			Caption:  item.Caption,
		},
	}

	var _, err = api.SendVideo(chat, item.File, opts)
	item.File.Delete()
	return err
}

func (item *VideoItem) Retry(api *telegram.T, chat telegram.ChatID) error {
	return (&TextItem{Text: item.Caption}).Send(api, chat)
}

type End struct {
	Offset Offset
}

func (_ *End) Interrupted() {

}
