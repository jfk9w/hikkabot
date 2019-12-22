package media

import (
	"sync"

	"github.com/jfk9w-go/flu"
	telegram "github.com/jfk9w-go/telegram-bot-api"
)

type Remote interface {
	URL() string
	Download(flu.Writable) (Type, error)
}

type Download interface {
	URL() string
	Wait() (ReadOnly, telegram.MediaType, error)
}

type media struct {
	Remote
	res  ReadOnly
	typ  telegram.MediaType
	err  error
	work sync.WaitGroup
}

func New(remote Remote) *media {
	media := &media{Remote: remote}
	media.work.Add(1)
	return media
}

type Batch = []*media

func (m *media) Wait() (ReadOnly, telegram.MediaType, error) {
	m.work.Wait()
	return m.res, m.typ, m.err
}
