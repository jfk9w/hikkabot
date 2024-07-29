package syncf

import (
	"context"
	"sync"
)

// WaitGroup is a wrapper for sync.WaitGroup.
type WaitGroup struct {
	sync.WaitGroup
}

// Spawn creates increments wait group counter by 1 and returns a new context with a CancelFunc.
// CancelFunc decrements wait group counter by 1.
func (wg *WaitGroup) Spawn(ctx context.Context) (context.Context, context.CancelFunc) {
	wg.Add(1)
	ctx, cancel := context.WithCancel(ctx)
	once := new(sync.Once)
	return ctx, func() {
		once.Do(func() {
			cancel()
			wg.Done()
		})
	}
}
