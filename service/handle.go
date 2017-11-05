package service

type Handle struct {
	stop0 chan struct{}
	done  chan struct{}
}

func NewHandle() *Handle {
	return &Handle{
		done: make(chan struct{}, 1),
	}
}

func (h *Handle) IsActive() bool {
	return h.stop0 != nil
}

func (h *Handle) Stop() {
	h.stop0 <- unit
}

func (h *Handle) Watch() <-chan struct{} {
	return h.stop0
}

func (h *Handle) start() <-chan struct{} {
	h.stop0 = make(chan struct{}, 1)
	return h.stop0
}

func (h *Handle) notify() {
	h.done <- unit
}

func (h *Handle) wait() {
	<-h.done
}
