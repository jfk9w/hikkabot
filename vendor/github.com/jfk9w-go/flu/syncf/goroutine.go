package syncf

import (
	"context"
	"sync"

	"github.com/jfk9w-go/flu/internal"
)

type goroutineIDKey struct{}

// GoroutineID extracts goroutine ID (which is set by GoWith) from context.Context.
func GoroutineID(ctx context.Context) (string, bool) {
	if ctx != nil {
		if id, ok := ctx.Value(goroutineIDKey{}).(string); ok {
			return id, true
		}
	}

	return "main", false
}

// Go starts a goroutine.
func Go(ctx context.Context, fun func(ctx context.Context)) (context.CancelFunc, error) {
	return GoWith(ctx, context.WithCancel, fun)
}

// GoWith starts a goroutine.
// It uses ContextFunc to spawn a new context.Context and then adds a goroutine ID values to that context.
// CancelFunc should be used for goroutine interruption via context.
// Error may be returned in case context spawning failed.
func GoWith(ctx context.Context, contextFunc ContextFunc, fun func(ctx context.Context)) (context.CancelFunc, error) {
	ctx, cancel := contextFunc(ctx)
	if ctx.Err() != nil {
		return func() {}, ctx.Err()
	}

	go func(ctx context.Context) {
		defer cancel()
		fun(context.WithValue(ctx, goroutineIDKey{}, internal.ID(ctx)[:10]))
	}(ctx)

	return cancel, nil
}

// GoSync acts like Go, but CancelFunc call will wait for goroutine to complete.
func GoSync(ctx context.Context, fun func(ctx context.Context)) context.CancelFunc {
	var work WaitGroup
	cancel, _ := GoWith(ctx, work.Spawn, fun)
	once := new(sync.Once)
	return func() {
		once.Do(func() {
			cancel()
			work.Wait()
		})
	}
}
