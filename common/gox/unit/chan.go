package unit

type Chan chan T

func NewChan() Chan {
	return Chan(make(chan T))
}

func NewBufferedChan(size int) Chan {
	return Chan(make(chan T, size))
}

func (mon Chan) Exec(body func()) bool {
	exec := NewChan()
	go func() {
		body()
		exec.Sync()
	}()

	select {
	case <-mon.Out():
		return false

	case <-exec.Out():
		return true
	}
}

func (mon Chan) Out() <-chan T {
	return mon
}

func (mon Chan) Ack() {
	<-mon.Out()
}

func (mon Chan) Sync() {
	(chan T)(mon) <- V
}

func (mon Chan) Intr() bool {
	_, ok := <-mon.Out()
	return ok
}

func (mon Chan) Close() {
	mon.Sync()
}
