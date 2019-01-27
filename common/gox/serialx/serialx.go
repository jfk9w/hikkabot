package serialx

import "time"

type T struct {
	queue chan *Item
	retry int
}

func New(delay time.Duration, retry int, bufSize int) *T {
	var queue chan *Item
	if bufSize > 0 {
		queue = make(chan *Item, bufSize)
	} else {
		queue = make(chan *Item)
	}

	go func() {
		for item := range queue {
			for !item.Resolve(nil) {
				time.Sleep(delay)
			}

			time.Sleep(delay)
		}
	}()

	return &T{queue, retry}
}

func (sx *T) Submit(action func(interface{}) Out) Out {
	item := NewItem(action, sx.retry)
	sx.queue <- item
	return item.Out()
}

func (sx *T) Close() error {
	close(sx.queue)
	return nil
}
