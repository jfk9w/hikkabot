package service

import (
	"time"
)

// Used for service management
type Handle interface {

	// Check if service is active
	IsActive() bool

	// Send stop signal
	Stop()

	// Wait while service stops
	Wait()

	// Prepare and start
	start() <-chan time.Time

	// Stop listener
	stopped()
}
