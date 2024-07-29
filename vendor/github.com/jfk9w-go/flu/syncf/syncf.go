// Package syncf contains common utilities for synchronization.
package syncf

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/pkg/errors"
)

var DefaultSignals = []os.Signal{syscall.SIGINT, syscall.SIGABRT, syscall.SIGKILL, syscall.SIGTERM}

// AwaitSignal blocks until a signal is received.
// By default, listens to DefaultSignals.
func AwaitSignal(ctx context.Context, signals ...os.Signal) {
	if len(signals) == 0 {
		signals = DefaultSignals
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, signals...)
	select {
	case <-c:
		return
	case <-ctx.Done():
		return
	}
}

// IsContextRelated checks if this is a "context" package error.
func IsContextRelated(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// Ref is a value which may not be available yet.
// This is synonymous to Future in Scala.
type Ref[V any] interface {

	// Get is used to get the actual value.
	// It should block until the value is available.
	Get(ctx context.Context) (V, error)
}

// Promise may be completed.
// This is synonymous to Promise in Scala.
type Promise[V any] interface {

	// Complete is used to complete this Promise.
	// It should generally not be called more than once.
	Complete(ctx context.Context, value V, err error) error
}

// Success is used to set a Promise.
func Success[V any](ctx context.Context, p Promise[V], value V) error {
	return p.Complete(ctx, value, nil)
}

// Failure is used to fail Promise.
func Failure[V any](ctx context.Context, p Promise[V], err error) error {
	var empty V
	return p.Complete(ctx, empty, err)
}

// Resolve is a function adapter for Ref.
type Resolve[V any] func(ctx context.Context) (V, error)

func (f Resolve[V]) Get(ctx context.Context) (V, error) {
	return f(ctx)
}

// Lazy runs the calculation on demand and stores the result.
func Lazy[V any](body Resolve[V]) Ref[V] {
	var (
		v    V
		err  error
		once sync.Once
	)

	return Resolve[V](func(ctx context.Context) (V, error) {
		once.Do(func() { v, err = body(ctx) })
		return v, err
	})
}

type result[V any] struct {
	value V
	err   error
}

// Var is a Ref and Promise implementation.
type Var[V any] struct {
	// Direct makes the Complete pass the result directly to Get without buffering.
	Direct   bool
	c        chan result[V]
	init0    sync.Once
	set0     sync.Once
	complete error
}

func (v *Var[V]) init() {
	v.init0.Do(func() {
		size := 1
		if v.Direct {
			size = 0
		}

		v.c = make(chan result[V], size)
	})
}

func (v *Var[V]) Get(ctx context.Context) (value V, err error) {
	if ctx == nil {
		ctx = context.Background()
	}

	v.init()
	select {
	case result := <-v.c:
		if v.Direct {
			v.c <- result
		}

		return result.value, result.err
	case <-ctx.Done():
		err = ctx.Err()
		return
	}
}

func (v *Var[V]) Complete(ctx context.Context, value V, err error) error {
	if ctx == nil {
		ctx = context.Background()
	}

	v.init()
	v.set0.Do(func() {
		select {
		case v.c <- result[V]{value, err}:
		case <-ctx.Done():
			v.complete = ctx.Err()
		}
	})

	return v.complete
}

// Val is a read-only value.
// Implements Ref interface.
type Val[V any] struct {
	V V
	E error
}

func (v Val[V]) Get(ctx context.Context) (V, error) {
	return v.V, v.E
}

// Async starts an asynchronous calculation and returns a Ref to the result.
func Async[V any](ctx context.Context, body Resolve[V]) Ref[V] {
	return AsyncWith[V](ctx, context.WithCancel, body)
}

// AsyncWith starts an asynchronous calculation and returns a Ref to the result.
// ContextFunc is used to spawn a new context for calculation.
func AsyncWith[V any](ctx context.Context, contextFunc ContextFunc, body Resolve[V]) Ref[V] {
	var result Var[V]
	if _, err := GoWith(ctx, contextFunc, func(ctx context.Context) {
		value, err := body(ctx)
		_ = result.Complete(ctx, value, err)
	}); err != nil {
		_ = Failure[V](ctx, &result, err)
	}

	return &result
}
