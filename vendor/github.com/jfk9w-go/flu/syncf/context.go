package syncf

import (
	"context"
	"time"
)

// ContextFunc creates a child context.Context from a given one.
type ContextFunc func(parent context.Context) (context.Context, context.CancelFunc)

// Stack stacks this ContextFunc with another ContextFunc.
// The resulting ContextFunc will first call the second one, and then the first one.
func (f ContextFunc) Stack(another ContextFunc) ContextFunc {
	return func(parent context.Context) (context.Context, context.CancelFunc) {
		ctx, cancel0 := f(parent)
		if ctx.Err() != nil {
			return ctx, func() {}
		}

		ctx, cancel1 := another(ctx)
		if ctx.Err() != nil {
			cancel0()
			return ctx, func() {}
		}

		return ctx, func() {
			cancel1()
			cancel0()
		}
	}
}

// Timeout returns ContextFunc which uses context.WithTimeout.
func Timeout(timeout time.Duration) ContextFunc {
	return func(parent context.Context) (context.Context, context.CancelFunc) {
		return context.WithTimeout(parent, timeout)
	}
}

// Deadline returns ContextFunc which uses context.WithDeadline.
func Deadline(deadline time.Time) ContextFunc {
	return func(parent context.Context) (context.Context, context.CancelFunc) {
		return context.WithDeadline(parent, deadline)
	}
}
