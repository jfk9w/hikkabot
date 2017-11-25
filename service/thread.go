package service

import "time"

// InactiveThread is a suspended thread (i.e. thread stopped due to an error)
type InactiveThread struct {

	// Offset in thread posts
	Offset int `json:"offset"`

	// StoppedAt denotes a moment in time when the thread was stopped (for garbage collection)
	StoppedAt time.Time `json:"stopped_at"`
}

func newInactiveThread(offset int) InactiveThread {
	return InactiveThread{
		Offset:    offset,
		StoppedAt: time.Now(),
	}
}
