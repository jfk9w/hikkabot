package apfel

import (
	"context"
	"io"
	"os"
	"reflect"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/logf"
	"github.com/jfk9w-go/flu/syncf"
	"github.com/pkg/errors"
)

// Core is the default MixinApp implementation.
type Core[C any] struct {
	syncf.Clock
	id      string
	version string
	config  C
	mixins  map[string]Mixin[C]
	close   []io.Closer
	mu      syncf.RWMutex
}

func (app *Core[C]) String() string {
	return app.id
}

func (app *Core[C]) Version() string {
	return app.version
}

func (app *Core[C]) Config() C {
	return app.config
}

func (app *Core[C]) getMixin(ctx context.Context, mixin Mixin[C]) bool {
	ctx, cancel := app.mu.RLock(ctx)
	if ctx.Err() != nil {
		return false
	} else {
		defer cancel()
	}

	if present, ok := app.mixins[mixin.String()]; ok {
		reflect.Indirect(reflect.ValueOf(mixin)).Set(reflect.Indirect(reflect.ValueOf(present)))
		return true
	}

	return false
}

func (app *Core[C]) addMixin(ctx context.Context, mixin Mixin[C]) error {
	ctx, cancel := app.mu.Lock(ctx)
	if ctx.Err() != nil {
		return ctx.Err()
	} else {
		defer cancel()
	}

	if app.mixins == nil {
		app.mixins = make(map[string]Mixin[C])
	}

	for _, present := range app.mixins {
		if listener, ok := mixin.(BeforeInclude[C]); ok {
			err := listener.BeforeInclude(ctx, app, present)
			logf.Get(app).Resultf(ctx, logf.Trace, logf.Error, "[%s] before include [%s]: %v", mixin, present, err)
			if err != nil {
				return errors.Wrapf(err, "%s before include %s", mixin, present)
			}
		}

		if listener, ok := present.(BeforeInclude[C]); ok {
			err := listener.BeforeInclude(ctx, app, mixin)
			logf.Get(app).Resultf(ctx, logf.Trace, logf.Error, "[%s] before include [%s]: %v", present, mixin, err)
			if err != nil {
				return errors.Wrapf(err, "%s before include %s", present, mixin)
			}
		}
	}

	id := mixin.String()
	if present, ok := app.mixins[id]; ok {
		reflect.Indirect(reflect.ValueOf(mixin)).Set(reflect.Indirect(reflect.ValueOf(present)))
		return nil
	}

	if err := mixin.Include(ctx, app); err != nil {
		return errors.Wrapf(err, "include %s", mixin)
	}

	for _, present := range app.mixins {
		if listener, ok := mixin.(AfterInclude[C]); ok {
			err := listener.AfterInclude(ctx, app, present)
			logf.Get(app).Resultf(ctx, logf.Trace, logf.Error, "[%s] after include [%s]: %v", mixin, present, err)
			if err != nil {
				return errors.Wrapf(err, "%s after include %s", mixin, present)
			}
		}

		if listener, ok := present.(AfterInclude[C]); ok {
			err := listener.AfterInclude(ctx, app, mixin)
			logf.Get(app).Resultf(ctx, logf.Trace, logf.Error, "[%s] after include [%s]: %v", present, mixin, err)
			if err != nil {
				return errors.Wrapf(err, "%s after include %s", present, mixin)
			}
		}
	}

	app.mixins[id] = mixin
	return nil
}

func (app *Core[C]) Use(ctx context.Context, mixin Mixin[C], mustExist bool) error {
	if app.getMixin(ctx, mixin) {
		return nil
	} else if mustExist {
		return errors.Errorf("mixin not found: %s", mixin)
	}

	return app.addMixin(ctx, mixin)
}

// ErrDisabled should be returned from a Mixin.Include when it is disabled for some reason.
// A warning message will be logged for such mixins in Core.Uses.
var ErrDisabled = errors.New("disabled")

// Uses calls Use for multiple mixins, skipping the ones which returned ErrDisabled on Mixin.Include.
// It panics on other include errors.
func (app *Core[C]) Uses(ctx context.Context, mixins ...Mixin[C]) {
	for _, mixin := range mixins {
		err := app.Use(ctx, mixin, false)
		switch {
		case errors.Is(err, ErrDisabled):
			logf.Get(app).Debugf(ctx, "%s is disabled", mixin)
		case syncf.IsContextRelated(err):
			break
		case err != nil:
			logf.Get(app).Panicf(ctx, "could not include %s: %+v", mixin, err)
		}
	}
}

func (app *Core[C]) ForEach(ctx context.Context, forEach ForEachMixin[C]) error {
	ctx, unlock := app.mu.RLock(ctx)
	if ctx.Err() != nil {
		return ctx.Err()
	} else {
		defer unlock()
	}

	for _, mixin := range app.mixins {
		err := forEach(ctx, mixin)
		switch {
		case errors.Is(err, ErrStopIteration):
			return nil
		case err == nil:
			continue
		default:
			return err
		}
	}

	return nil
}

// Manage adds the Service to lifecycle management.
func (app *Core[C]) Manage(ctx context.Context, service any) error {
	if closer, ok := service.(io.Closer); ok {
		switch closer {
		case os.Stdin, os.Stdout, os.Stderr:
			// do not lock for nothing
			return nil
		}

		if ctx, cancel := app.mu.Lock(ctx); ctx.Err() == nil {
			defer cancel()
			app.close = append(app.close, closer)
			logf.Get(app).Tracef(ctx, "managing [%s]", flu.Readable(service))
		} else {
			flu.CloseQuietly(closer)
			return ctx.Err()
		}
	}

	return nil
}

// Close closes the context and shuts down the application.
func (app *Core[C]) Close() error {
	_, cancel := app.mu.Lock(nil)
	defer cancel()
	for i := len(app.close); i > 0; i-- {
		closer := app.close[i-1]
		err := closer.Close()
		logf.Get(app).Resultf(nil, logf.Debug, logf.Warn, "close [%s]: %v", flu.Readable(closer), err)
	}

	return nil
}
