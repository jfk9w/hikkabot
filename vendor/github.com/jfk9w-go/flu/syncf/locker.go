package syncf

import (
	"context"
	"math"
	"sync"
	"time"
)

// Locker may be locked (interruptibly).
type Locker interface {
	// Lock locks something.
	// It returns a context which should be for errors (via context.Context.Err()) and used for further
	// execution under the lock. CancelFunc must be called to release the lock (usually inside defer).
	Lock(ctx context.Context) (context.Context, context.CancelFunc)
}

// RWLocker supports read-write locks.
type RWLocker interface {
	Locker

	// RLock acts the same way as Lock does, but it allows for multiple readers to hold the lock (or a single writer).
	RLock(ctx context.Context) (context.Context, context.CancelFunc)
}

type semaphore struct {
	clock    Clock
	interval time.Duration
	c        chan time.Time
}

// Semaphore allows at most `size` events at given time interval.
// This becomes a classic semaphore when interval = 0, and a simple mutex when size = 1.
// This is a reentrant Locker.
func Semaphore(clock Clock, size int, interval time.Duration) Locker {
	if size <= 0 {
		return unlock{}
	} else {
		if clock == nil {
			clock = ClockFunc(func() (r time.Time) { return })
		}

		c := make(chan time.Time, size)
		for i := 0; i < size; i++ {
			c <- clock.Now().Add(-interval)
		}

		return &semaphore{
			clock:    clock,
			interval: interval,
			c:        c,
		}
	}
}

func (s *semaphore) Lock(ctx context.Context) (context.Context, context.CancelFunc) {
	return reentrant(ctx, s, 1, s.lock)
}

func (s *semaphore) lock(ctx context.Context) (context.Context, context.CancelFunc) {
	select {
	case occ := <-s.c:
		if s.interval > 0 {
			wait := occ.Add(s.interval).Sub(s.clock.Now())
			if wait > 0 {
				select {
				case <-time.After(wait):
				case <-ctx.Done():
					s.c <- occ
					return ctx, noop
				}
			}
		}

		once := new(sync.Once)
		return ctx, func() { once.Do(func() { s.c <- s.clock.Now() }) }
	case <-ctx.Done():
		return ctx, noop
	}
}

// RWMutex is an interruptible reentrant sync.RWMutex implementation.
type RWMutex struct {
	w    chan bool
	r    chan int
	once sync.Once
}

func (m *RWMutex) init() {
	m.w = make(chan bool, 1)
	m.r = make(chan int, 1)
}

func (m *RWMutex) Lock(ctx context.Context) (context.Context, context.CancelFunc) {
	return reentrant(ctx, m, 1, m.lock)
}

func (m *RWMutex) RLock(ctx context.Context) (context.Context, context.CancelFunc) {
	return reentrant(ctx, m, 2, m.rlock)
}

func (m *RWMutex) lock(ctx context.Context) (context.Context, context.CancelFunc) {
	m.once.Do(m.init)
	select {
	case m.w <- true:
		once := new(sync.Once)
		return ctx, func() { once.Do(func() { <-m.w }) }
	case <-ctx.Done():
		return ctx, noop
	}
}

func (m *RWMutex) rlock(ctx context.Context) (context.Context, context.CancelFunc) {
	m.once.Do(m.init)
	var rs int
	select {
	case m.w <- true:
	case rs = <-m.r:
	case <-ctx.Done():
		return ctx, noop
	}

	rs++
	m.r <- rs
	once := new(sync.Once)
	return ctx, func() {
		once.Do(func() {
			rs := <-m.r
			rs--
			if rs == 0 {
				<-m.w
			} else {
				m.r <- rs
			}
		})
	}
}

// Unlock does nothing.
var Unlock Locker = unlock{}

type unlock struct{}

func (unlock) Lock(ctx context.Context) (context.Context, context.CancelFunc) {
	return ctx, noop
}

func reentrant(ctx context.Context, key any, level int, lock ContextFunc) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}

	contextLevel := math.MaxInt
	if level, ok := ctx.Value(key).(int); ok {
		contextLevel = level
	}

	if contextLevel <= level {
		return ctx, noop
	} else {
		ctx, cancel := lock(ctx)
		if ctx.Err() != nil {
			return ctx, noop
		}

		return context.WithValue(ctx, key, level), cancel
	}
}

func noop() {}

// Lockers represents a Locker which locks all Locker instances in the slice when Lock is called
// and unlocks them in reverse order when CancelFunc is called.
type Lockers []Locker

func (ls Lockers) Lock(ctx context.Context) (context.Context, context.CancelFunc) {
	cancels := make([]context.CancelFunc, 0, len(ls))
	cancelAll := func() {
		for i := len(cancels) - 1; i >= 0; i-- {
			cancels[i]()
		}
	}

	for _, locker := range ls {
		var cancel context.CancelFunc
		ctx, cancel = locker.Lock(ctx)
		if ctx.Err() != nil {
			cancelAll()
			return ctx, noop
		}

		cancels = append(cancels, cancel)
	}

	return ctx, cancelAll
}
