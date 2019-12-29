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

func (s *UpdateQueue) Offer(update Update) bool {
	select {
	case <-s.cancel:
		return false
	case s.updates <- update:
		return true
	}
}

func (s *UpdateQueue) Fail(err error) {
	if s.err == nil {
		s.err = err
	}
}

func (s *UpdateQueue) run(ctx ApplicationContext, offset int64, item Item) {
	defer close(s.updates)
	item.Update(ctx, offset, s)
}
