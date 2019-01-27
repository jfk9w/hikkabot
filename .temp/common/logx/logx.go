// Package logx contains a wrapper for github.com/sirupsen/logrus library.
// Provides SLF4J-esque experience in Go.
//
// Configuration
//
// Configuration file is passed via the environment variable LOGX.
// If LOGX is not set all logging output will be printed to stdout
// with debug level.
package logx

import (
	"sync"

	"github.com/sirupsen/logrus"
)

type (
	// Alias *logrus.Logger
	Ptr = *logrus.Logger

	// Alias logrus.Fields
	V = logrus.Fields
)

var obj = internal{
	config:  config(),
	loggers: new(sync.Map),
}

// Get a logger with the specified name.
func Get(name string) Ptr {
	return obj.get(name)
}
