package subscription

import (
	"github.com/jfk9w/hikkabot/media"
)

type Offset = int64

type Update struct {
	Offset Offset
	Text   []string
	Media  []media.Media
}

type UpdateCollection struct {
	C      chan Update
	Error  error
	cancel chan struct{}
}

func NewUpdateCollection(size int) *UpdateCollection {
	if size < 0 {
		panic("size must be non-negative")
	}

	return &UpdateCollection{make(chan Update, size), nil, make(chan struct{}, 1)}
}

func (uc *UpdateCollection) Cancel() <-chan struct{} {
	return uc.cancel
}
