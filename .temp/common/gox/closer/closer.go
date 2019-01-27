package closer

import (
	"sync"
)

type I interface {
	Close()
}

type seq []I

func (s seq) Close() {
	for _, closer := range s {
		closer.Close()
	}
}

type bc []I

func (b bc) Close() {
	work := sync.WaitGroup{}
	work.Add(len(b))
	for _, closer := range b {
		go func(closer I) {
			defer work.Done()
			closer.Close()
		}(closer)
	}

	work.Wait()
}

func Sequential(closers ...I) I {
	return seq(closers)
}

func Broadcast(closers ...I) I {
	return bc(closers)
}
