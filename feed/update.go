package feed

import (
	"github.com/jfk9w/hikkabot/format"
	"github.com/jfk9w/hikkabot/media"
)

type Update struct {
	Offset int64
	Text   format.Text
	Media  []*media.Media
}

type UpdateQueue struct {
	updates chan Update
	err     error
	cancel  chan struct{}
}

func newUpdateQueue() *UpdateQueue {
	return &UpdateQueue{
		updates: make(chan Update, 10),
		cancel:  make(chan struct{}),
	}
}

func (s *UpdateQueue) Offer(update Update) bool {
	select {
	case <-s.cancel:
		return false
	case s.updates <- update:
		return true
	}
}

func (s *UpdateQueue) pull(ctx Context, offset int64, item Item) {
	defer close(s.updates)
	err := item.Update(ctx, offset, s)
	if err != nil {
		s.err = err
	}
	return
}
