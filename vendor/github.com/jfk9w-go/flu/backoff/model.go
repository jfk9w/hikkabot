package backoff

import "time"

// Interface is backoff strategy interface.
type Interface interface {

	// Timeout returns sleep duration.
	Timeout(retry int) time.Duration
}

// Temporary checks if the provided error is temporary.
type Temporary interface {
	IsTemporary(err error) bool
}
