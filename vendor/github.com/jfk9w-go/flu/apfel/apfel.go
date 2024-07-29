// Package apfel provides an application context implementation with the support
// of "mixins" – services which may be used in the context of "dependency injection".
// "Dependency injection" means the ability not to pass all services dependency directly
// for initialization, but instead implementing a Mixin interface and
// calling MixinApp.Use in Mixin.Include implementation in order to get (or create)
// a mixin dependency from application context.
package apfel

import (
	"context"

	"github.com/jfk9w-go/flu/syncf"
	"github.com/pkg/errors"
)

const rootLoggerName = "apfel"

// ErrStopIteration should be returned from ForEachMixin when iteration must be stopped.
var ErrStopIteration = errors.New("stop iteration")

// ForEachMixin is an iteration function.
type ForEachMixin[C any] func(ctx context.Context, mixin Mixin[C]) error

// MixinApp is the base application interface with Mixin support.
// C is the application configuration type (should be struct).
type MixinApp[C any] interface {
	syncf.Clock

	// Version returns the application version.
	Version() string

	// Config returns the application configuration.
	Config() C

	// Manage schedules the service for closing on application shutdown if service implements io.Closer.
	Manage(ctx context.Context, service any) error

	// Use creates or gets a mixin from application context.
	// If mustExist is set, an error is returned if the mixin is not initialized yet.
	// A mixin should generally be an empty struct pointer which will be initialized
	// and put into application context, or an already initialized mixin value from
	// application context will be copied into this pointer.
	//
	// This is the main "dependency injection" entrypoint.
	Use(ctx context.Context, mixin Mixin[C], mustExist bool) error

	// ForEach iterates all initialized mixins with a ForEachMixin function.
	// Iteration stops when ErrStopIteration is returned from the function.
	ForEach(ctx context.Context, forEach ForEachMixin[C]) error
}

// Mixin is a service which may be used in application context.
// A Mixin will be initialized at most once (when MixinApp.Use is called).
// This interface serves as a base "dependency injection" unit.
//
// C is the application configuration type (should be struct).
// Its type bound should be a configuration interface which must be implemented by
// enclosing application configuration type.
type Mixin[C any] interface {

	// String is used to identify mixins in application context.
	// This should be generally be a constant string which does not depend on implementing struct values.
	String() string

	// Include is called when mixin is initialized.
	// Note that no automatic cleanup is performed on mixins themselves,
	// implementations are required to call MixinApp.Manage for used resources
	// in order to schedule their closing on application shutdown.
	Include(ctx context.Context, app MixinApp[C]) error
}

// BeforeInclude is an interface which may be implemented by a Mixin
// in order to be called before some other Mixin is included (initialized).
type BeforeInclude[C any] interface {
	BeforeInclude(ctx context.Context, app MixinApp[C], mixin Mixin[C]) error
}

// AfterInclude is an interface which may be implemented by a Mixin
// in order to be called after some other Mixin is included (initialized).
type AfterInclude[C any] interface {
	AfterInclude(ctx context.Context, app MixinApp[C], mixin Mixin[C]) error
}
