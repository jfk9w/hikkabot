// Package logf contains logging registry & extensions based on stdlib "log".
package logf

import (
	"context"
	"io"
	"log"
	"os"
	"strings"

	"github.com/jfk9w-go/flu/internal"
	"github.com/jfk9w-go/flu/syncf"
)

var (
	loggers map[string]Interface
	mu      syncf.RWMutex

	defaultLogger Interface = &BareAdapter{
		Bare:  NewStdLogger("", log.Flags()|log.Ldate|log.Ltime, os.Stderr),
		level: Info,
	}

	defaultFactory Factory = func(name string, defaultLogger Interface) Interface {
		return &BareAdapter{
			Bare:  NewStdLogger(name, log.Flags()|log.Ldate|log.Ltime, os.Stderr),
			level: defaultLogger.Level(),
		}
	}
)

func init() {
	for level, str := range level2string {
		string2level[str] = level
	}
}

func getLogger(key string) (Interface, bool) {
	_, cancel := mu.RLock(nil)
	defer cancel()
	logger, ok := loggers[key]
	return logger, ok
}

// Get returns a logger with name corresponding to path elements.
// Each path element is converted to string via flu.Readable.
// Level, io.Writer or Factory can also be passed and will be applied
// only when creating a new logger for the path.
func Get(path ...any) Interface {
	return get(path)
}

// OutputSetter is a logger which supports resetting output.
type OutputSetter interface {

	// SetOutput sets output for this logger.
	SetOutput(w io.Writer)
}

func get(path []any) Interface {
	var (
		key     strings.Builder
		level   Level
		writer  io.Writer
		factory Factory
	)

	for i, value := range path {
		switch v := value.(type) {
		case Level:
			level = v
		case io.Writer:
			writer = v
		case Factory:
			factory = v
		default:
			if i > 0 {
				key.WriteRune('.')
			}

			key.WriteString(internal.Readable(value))
		}
	}

	if key.String() == "" {
		return defaultLogger
	}

	if logger, ok := getLogger(key.String()); ok {
		return logger
	}

	_, cancel := mu.Lock(nil)
	defer cancel()
	if logger, ok := loggers[key.String()]; ok {
		return logger
	}

	if factory == nil {
		factory = defaultFactory
	}

	logger := factory(key.String(), defaultLogger)
	if setter, ok := logger.(OutputSetter); ok && writer != nil {
		setter.SetOutput(writer)
	}

	if level > 0 {
		logger.SetLevel(level)
	}

	if loggers == nil {
		loggers = make(map[string]Interface)
	}

	loggers[key.String()] = logger
	return logger
}

// Factory creates a logger for a provided name.
type Factory func(name string, defaultLogger Interface) Interface

// Default returns default (no name) logger.
func Default() Interface {
	return defaultLogger
}

// ForEach applies f for each registered logger.
func ForEach(f func(name string, logger Interface) bool) {
	_, cancel := mu.Lock(nil)
	defer cancel()
	for name, logger := range loggers {
		if !f(name, logger) {
			break
		}
	}
}

// ResetLevel sets level for all registered loggers.
func ResetLevel(level Level) {
	Default().SetLevel(level)
	ForEach(func(_ string, logger Interface) bool {
		logger.SetLevel(level)
		return true
	})
}

// ResetFactory sets logger factory and removes all registered loggers.
func ResetFactory(factory Factory) {
	_, cancel := mu.Lock(nil)
	defer cancel()
	defaultLogger = factory("", nil)
	defaultFactory = factory
	loggers = nil
}

// DefaultFactory returns default logger factory.
func DefaultFactory() Factory {
	_, cancel := mu.RLock(nil)
	defer cancel()
	return defaultFactory
}

// Resultf calls Resultf on default logger.
func Resultf(ctx context.Context, ok, bad Level, pattern string, values ...any) {
	Default().Resultf(ctx, ok, bad, pattern, values...)
}

// Printf calls Printf on default logger.
func Printf(ctx context.Context, pattern string, values ...any) {
	Default().Printf(ctx, pattern, values...)
}

// Tracef calls Tracef on default logger.
func Tracef(ctx context.Context, pattern string, values ...any) {
	Default().Tracef(ctx, pattern, values...)
}

// Debugf calls Debugf on default logger.
func Debugf(ctx context.Context, pattern string, values ...any) {
	Default().Debugf(ctx, pattern, values...)
}

// Infof calls Infof on default logger.
func Infof(ctx context.Context, pattern string, values ...any) {
	Default().Infof(ctx, pattern, values...)
}

// Warnf calls Warnf on default logger.
func Warnf(ctx context.Context, pattern string, values ...any) {
	Default().Warnf(ctx, pattern, values...)
}

// Errorf calls Errorf on default logger.
func Errorf(ctx context.Context, pattern string, values ...any) {
	Default().Errorf(ctx, pattern, values...)
}

// Panicf calls Panicf on default logger.
func Panicf(ctx context.Context, pattern string, values ...any) {
	Default().Panicf(ctx, pattern, values...)
}
