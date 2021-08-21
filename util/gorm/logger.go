package gorm

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm/logger"
)

var LogrusLogger logger.Interface = (*logrusLogger)(logrus.StandardLogger())

type logrusLogger logrus.Logger

func (l *logrusLogger) unmask() *logrus.Logger {
	return (*logrus.Logger)(l)
}

func (l *logrusLogger) LogMode(level logger.LogLevel) logger.Interface {
	return l
}

func (l *logrusLogger) Info(ctx context.Context, s string, i ...interface{}) {
	l.unmask().WithContext(ctx).Infof(s, i...)
}

func (l *logrusLogger) Warn(ctx context.Context, s string, i ...interface{}) {
	l.unmask().WithContext(ctx).Warnf(s, i...)
}

func (l *logrusLogger) Error(ctx context.Context, s string, i ...interface{}) {
	l.unmask().WithContext(ctx).Errorf(s, i...)
}

func (l *logrusLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	sql, rowsAffected := fc()
	fields := logrus.Fields{"begin": begin}
	l.unmask().WithContext(ctx).WithFields(fields).Tracef("%s (%d): %v", sql, rowsAffected, err)
}
