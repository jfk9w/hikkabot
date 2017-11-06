package service

type stopHandle struct {
	signal chan struct{}
	done   chan struct{}
}

func newStopHandle() stopHandle {
	return stopHandle{
		signal: make(chan struct{}, 1),
		done:   make(chan struct{}, 1),
	}
}

func (s stopHandle) check() bool {
	if _, ok := <-s.signal; ok {
		return false
	}

	return true
}

func (s stopHandle) notify() {
	s.signal <- unit
}

func (s stopHandle) wait() {
	<-s.done
}
