package syncf

import (
	"time"
)

// Clock is an interface for clocks returning current time.
type Clock interface {

	// Now returns the current time according to this Clock.
	Now() time.Time
}

// ClockFunc is a functional adapter.
type ClockFunc func() time.Time

func (fun ClockFunc) Now() time.Time {
	return fun()
}

// DefaultClock uses time.Now to provide current time.
var DefaultClock Clock = ClockFunc(time.Now)
