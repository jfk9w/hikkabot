package feed

import (
	"sync"

	"github.com/jfk9w-go/aconvert"
	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/hikkabot/html"
	"github.com/jfk9w-go/logrus"
	"github.com/jfk9w-go/telegram"
	"github.com/jfk9w-go/unit"
)

type (
	Bot interface {
		SendPost(telegram.ChatRef, html.Post) error
	}

	Dvach interface {
		Thread(dvach.ID, int) ([]dvach.Post, error)
	}

	Converter interface {
		Convert(string, chan aconvert.VideoResponse)
	}

	Entry struct {
		Offset int
		Hash   string
	}

	Error struct {
		Thread dvach.ID
		Hash   string
		Err    error
	}
)

func (e Entry) WithOffset(offset int) Entry {
	e.Offset = offset
	return e
}

func Run(bot Bot, dvch Dvach, conv Converter, chat telegram.ChatRef) *T {
	feed := &T{
		aux:     unit.NewAux(),
		bot:     bot,
		dvch:    dvch,
		conv:    conv,
		chat:    chat,
		queue:   make(chan dvach.ID, maxQueueSize),
		err:     make(chan Error, maxQueueSize),
		entries: make(map[dvach.ID]Entry),
		mu:      new(sync.RWMutex),
	}

	go feed.run()
	return feed
}

var log = logrus.GetLogger("feed")
