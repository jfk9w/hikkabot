package gormf

import (
	"context"
	"time"

	"github.com/jfk9w-go/flu/logf"

	"github.com/jfk9w-go/flu/syncf"

	"gorm.io/gorm/logger"
)

type logfLogger struct {
	clock syncf.Clock
	name  []any
}

// LogfLogger returns a logger.Interface adapter for logf.
func LogfLogger(clock syncf.Clock, name ...any) logger.Interface {
	return &logfLogger{
		clock: clock,
		name:  name,
	}
}

func (l logfLogger) LogMode(logger.LogLevel) logger.Interface {
	// level is not controlled by gorm
	return l
}

func (l logfLogger) Info(ctx context.Context, s string, i ...interface{}) {
	logf.Get(l.name...).Infof(ctx, s, i...)
}

func (l logfLogger) Warn(ctx context.Context, s string, i ...interface{}) {
	logf.Get(l.name...).Warnf(ctx, s, i...)
}

func (l logfLogger) Error(ctx context.Context, s string, i ...interface{}) {
	logf.Get(l.name...).Errorf(ctx, s, i...)
}

func (l logfLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if logf.Get(l.name...).Level() > logf.Trace {
		return
	}

	sql, rowsAffected := fc()
	logf.Get(l.name...).Tracef(ctx, "[%s] affected %d rows in %s: %v", sql, rowsAffected, l.clock.Now().Sub(begin), err)
}
