package subscription

import (
	"github.com/jfk9w/hikkabot/format"
	"github.com/jfk9w/hikkabot/media"
)

type Update struct {
	Offset int64
	Text   format.Text
	Media  media.Batch
}

type UpdateSession struct {
	ch     chan Update
	err    error
	cancel chan struct{}
}

func (s *UpdateSession) Submit(update Update) bool {
	select {
	case <-s.cancel:
		return false
	case s.ch <- update:
		return true
	}
}

func (s *UpdateSession) Fail(err error) {
	if s.err == nil {
		s.err = err
	}
}

func (s *UpdateSession) close() {
	close(s.ch)
}

func (s *UpdateSession) run(ctx Context, offset int64, item Item) {
	defer close(s.ch)
	item.Update(ctx, offset, s)
}
