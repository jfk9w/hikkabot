package subscription

import (
	"github.com/jfk9w/hikkabot/app/media"
)

type Offset = int64

type Update struct {
	Offset Offset
	Text   []string
	Media  []media.Media
}

type UpdateCollection struct {
	C   chan Update
	ic  chan struct{}
	Err error
}

func NewUpdateCollection(size int) *UpdateCollection {
	if size < 0 {
		panic("size must be non-negative")
	}

	return &UpdateCollection{make(chan Update, size), make(chan struct{}, 1), nil}
}

func (uc *UpdateCollection) interrupt() {
	uc.ic <- struct{}{}
	close(uc.ic)
}

func (uc *UpdateCollection) Interrupt() <-chan struct{} {
	return uc.ic
}
