package logf

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/fatih/color"
	"github.com/jfk9w-go/flu/syncf"
	"github.com/pkg/errors"
)

// Level is the logging level.
type Level int8

const (
	Trace Level = iota + 1
	Debug
	Info
	Warn
	Error
	Panic
	Silent
)

var level2string = map[Level]string{
	Trace:  "trace",
	Debug:  "debug",
	Info:   "info",
	Warn:   "warn",
	Error:  "error",
	Panic:  "panic",
	Silent: "silent",
}

var string2level = make(map[string]Level)

// Skip returns true if msgLevel does not match this level.
func (l Level) Skip(msgLevel Level) bool {
	return msgLevel == Silent || l == Silent || msgLevel != Panic && l > msgLevel
}

func (l *Level) UnmarshalJSON(data []byte) (err error) {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	*l, err = ParseLevel(value)
	return
}

func (l Level) MarshalJSON() ([]byte, error) {
	return json.Marshal(level2string[l])
}

func (l *Level) UnmarshalYAML(node *yaml.Node) (err error) {
	*l, err = ParseLevel(node.Value)
	return
}

func (l Level) MarshalYAML() (any, error) {
	return level2string[l], nil
}

// ParseLevel parses Level from string.
func ParseLevel(value string) (Level, error) {
	if level, ok := string2level[value]; ok {
		return level, nil
	} else {
		return Silent, errors.Errorf("unknown log level: %s", value)
	}
}

// Colored should be set if colored logging output is desired
// (colors are used for levels and context IDs).
var Colored = true

type colorFunc func(string, ...any) string

var level2color = map[Level]colorFunc{
	Trace: fmt.Sprintf,
	Debug: fmt.Sprintf,
	Info:  color.CyanString,
	Warn:  color.YellowString,
	Error: color.RedString,
	Panic: color.HiRedString,
}

var colors = []colorFunc{
	color.HiBlackString,
	color.CyanString,
	color.GreenString,
	color.MagentaString,
	color.WhiteString,
	color.HiWhiteString,
}

func randomColor(value string) colorFunc {
	hash := 0
	for _, c := range value {
		hash += int(c) % len(colors)
	}

	idx := hash % len(colors)
	return colors[idx]
}

// Bare is a bare logger interface.
type Bare interface {
	Logf(ctx context.Context, level Level, pattern string, values ...any)
}

// Interface is the main logger interface.
// Note that Interface instances should not be kept, but instead always got from Get.
type Interface interface {
	// SetLevel sets level for this logger.
	SetLevel(level Level)
	// Level returns current logging level.
	Level() Level
	// Logf logs the message with specified level.
	Logf(ctx context.Context, level Level, pattern string, values ...any)
	// Resultf uses `ok` level if the last argument is nil or `bad` level if the last argument is non-nil error.
	Resultf(ctx context.Context, ok, bad Level, pattern string, values ...any)
	// Printf prints log message with arguments.
	Printf(ctx context.Context, pattern string, values ...any)
	// Tracef prints log message with Trace level.
	Tracef(ctx context.Context, pattern string, values ...any)
	// Debugf prints log message with Debug level.
	Debugf(ctx context.Context, pattern string, values ...any)
	// Infof prints log message with Info level.
	Infof(ctx context.Context, pattern string, values ...any)
	// Warnf prints log message with Warn level.
	Warnf(ctx context.Context, pattern string, values ...any)
	// Errorf prints log message with Error level.
	Errorf(ctx context.Context, pattern string, values ...any)
	// Panicf prints log message with Panic level and panics.
	// This behavior is inherent in Resultf if `bad` == Panic.
	Panicf(ctx context.Context, pattern string, values ...any)
	// Panic prints error and panics.
	Panic(ctx context.Context, value any)
}

// StdLogger is a wrapper for stdlib log.Logger with additional capabilities.
type StdLogger struct {
	logger *log.Logger
	level  Level
	name   string
	mu     syncf.RWMutex
}

// NewStdLogger creates a new StdLogger instance.
func NewStdLogger(name string, flags int, writer io.Writer) Bare {
	return &StdLogger{
		logger: log.New(writer, "", flags|log.Ldate|log.Ltime),
		name:   name,
	}
}

func (l *StdLogger) SetOutput(w io.Writer) {
	l.logger.SetOutput(w)
}

func (l *StdLogger) Logf(ctx context.Context, level Level, pattern string, values ...any) {
	message := strings.ToUpper(level2string[level])
	if len(message) > 5 {
		message = message[:5]
	} else {
		// padding
		for len(message) < 5 {
			message += " "
		}
	}

	if Colored {
		message = level2color[level](message)
	}

	var addSeparator bool
	if l.name != "" {
		message += " " + l.name
		addSeparator = true
	}

	if goroutineID, ok := syncf.GoroutineID(ctx); ok {
		if Colored {
			goroutineID = randomColor(goroutineID)(goroutineID)
		}

		message += " (" + goroutineID + ")"
		addSeparator = true
	}

	if addSeparator {
		if Colored {
			message += " •"
		} else {
			message += " $"
		}
	}

	message += " " + pattern
	l.logger.Printf(message, values...)
}

// BareAdapter is adapter for Bare to match Interface.
type BareAdapter struct {
	Bare
	level Level
	mu    syncf.RWMutex
}

func (a *BareAdapter) SetLevel(level Level) {
	_, cancel := a.mu.Lock(nil)
	defer cancel()
	a.level = level
}

func (a *BareAdapter) Level() Level {
	_, cancel := a.mu.RLock(nil)
	defer cancel()
	return a.level
}

func (a *BareAdapter) Logf(ctx context.Context, level Level, pattern string, values ...any) {
	if a.Level().Skip(level) {
		return
	}

	if level == Panic {
		err, _ := getLastError(values)
		if err == nil {
			err = errors.Errorf(pattern, values...)
		}

		if _, ok := err.(recovered); !ok {
			err = recovered{err}
			a.Bare.Logf(ctx, Panic, pattern, values...)
		}

		panic(err)
	}

	a.Bare.Logf(ctx, level, pattern, values...)
}

func (a *BareAdapter) Resultf(ctx context.Context, ok, bad Level, pattern string, values ...any) {
	if err, _ := getLastError(values); err != nil {
		a.Logf(ctx, bad, pattern, values...)
	} else {
		values[len(values)-1] = "ok"
		a.Logf(ctx, ok, pattern, values...)
	}
}

func (a *BareAdapter) Printf(ctx context.Context, pattern string, values ...any) {
	a.Resultf(ctx, Debug, Warn, pattern, values...)
}

func (a *BareAdapter) Tracef(ctx context.Context, pattern string, values ...any) {
	a.Logf(ctx, Trace, pattern, values...)
}

func (a *BareAdapter) Debugf(ctx context.Context, pattern string, values ...any) {
	a.Logf(ctx, Debug, pattern, values...)
}

func (a *BareAdapter) Infof(ctx context.Context, pattern string, values ...any) {
	a.Logf(ctx, Info, pattern, values...)
}

func (a *BareAdapter) Warnf(ctx context.Context, pattern string, values ...any) {
	a.Logf(ctx, Warn, pattern, values...)
}

func (a *BareAdapter) Errorf(ctx context.Context, pattern string, values ...any) {
	a.Logf(ctx, Error, pattern, values...)
}

func (a *BareAdapter) Panicf(ctx context.Context, pattern string, values ...any) {
	a.Logf(ctx, Panic, pattern, values...)
}

func (a *BareAdapter) Panic(ctx context.Context, value any) {
	a.Panicf(ctx, "%+v", value)
}

func getLastError(values []any) (error, bool) {
	if len(values) > 0 {
		err, ok := values[len(values)-1].(error)
		return err, ok
	} else {
		return nil, false
	}
}

type recovered struct {
	err error
}

func (e recovered) Error() string {
	return e.err.Error()
}

func (e recovered) Unwrap() error {
	return e.err
}
