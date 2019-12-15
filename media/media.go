package media

import (
	"os"
	"sync"

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
	} else {
		return MaxVideoSize
	}
}

func (t Type) TelegramType() telegram.MediaType {
	if t.IsPhoto() {
		return telegram.Photo
	} else {
		return telegram.Video
	}
}

type Loader interface {
	LoadMedia(flu.ResourceWriter) (Type, error)
}

type Media struct {
	Href   string
	Loader Loader
	file   flu.File
	type_  Type
	err    error
	work   sync.WaitGroup
}

type Batch = []*Media

func NewBatch(loaders ...Loader) Batch {
	batch := make([]*Media, len(loaders))
	for i, loader := range loaders {
		batch[i] = &Media{Loader: loader}
		batch[i].work.Add(1)
	}
	return batch
}

func (m *Media) WaitForResult() (flu.File, Type, error) {
	m.work.Wait()
	if m.err != nil {
		os.RemoveAll(m.file.Path())
		return "", 0, m.err
	} else {
		return m.file, m.type_, nil
	}
}
