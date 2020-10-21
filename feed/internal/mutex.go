package internal

import (
	"context"

	"github.com/jfk9w-go/flu"
)

type operation struct {
	enter, exit flu.WaitGroup
	write       bool
}

func (op *operation) Unlock() {
	op.exit.Done()
}

type OrderedMutex struct {
	queue  chan *operation
	cancel func()
	work   flu.WaitGroup
}

func NewRWMutex() *OrderedMutex {
	mu := &OrderedMutex{queue: make(chan *operation)}
	mu.cancel = mu.work.Go(context.Background(), flu.RateUnlimiter, mu.daemon)
	return mu
}

func (mu *OrderedMutex) daemon(_ context.Context) {
	var active []*operation
	for op := range mu.queue {
		if !op.write {
			op.enter.Done()
			active = append(active, op)
			continue
		}

		if len(active) > 0 {
			for _, op := range active {
				op.exit.Wait()
			}

			active = []*operation{}
		}

		op.enter.Done()
		op.exit.Wait()
	}
}

func (mu *OrderedMutex) RLock() flu.Unlocker {
	return mu.doLock(false)
}

func (mu *OrderedMutex) Lock() flu.Unlocker {
	return mu.doLock(true)
}

func (mu *OrderedMutex) doLock(write bool) flu.Unlocker {
	op := &operation{write: write}
	op.enter.Add(1)
	mu.queue <- op
	op.enter.Wait()
	op.exit.Add(1)
	return op
}

func (mu *OrderedMutex) Close() {
	close(mu.queue)
	mu.cancel()
	mu.work.Wait()
}
