package media

import (
	"github.com/jfk9w-go/flu"
	telegram "github.com/jfk9w-go/telegram-bot-api"
)

const (
	MaxPhotoSize int64 = 10 * (2 << 20)
	MaxVideoSize int64 = 50 * (2 << 20)
	MinMediaSize int64 = 10 << 10
)

type Type uint8

const (
	Photo Type = iota
	Video
	WebM
)

func (t Type) IsPhoto() bool {
	return t == Photo
}

func (t Type) IsVideo() bool {
	return t == Video || t == WebM
}

func (t Type) MaxSize() int64 {
	if t.IsPhoto() {
		return MaxPhotoSize
	}

	return MaxVideoSize
}

func (t Type) TelegramType() telegram.MediaType {
	if t.IsPhoto() {
		return telegram.Photo
	}

	return telegram.Video
}

type Factory = func(flu.FileSystemResource) (Type, error)

type Media struct {
	Href    string
	Factory Factory

	resource  flu.FileSystemResource
	mediaType Type
	err       error
	done      chan struct{}
}

func (m *Media) init() *Media {
	m.done = make(chan struct{}, 1)
	return m
}

func (m *Media) complete() {
	m.done <- struct{}{}
	close(m.done)
}

func (m *Media) Get() (flu.FileSystemResource, Type, error) {
	<-m.done
	return m.resource, m.mediaType, m.err
}
