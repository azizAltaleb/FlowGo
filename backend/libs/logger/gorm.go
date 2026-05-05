package logger

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// GormLogger adapts our structured logger for GORM
type GormLogger struct {
	log                  *Logger
	SlowThreshold        time.Duration
	IgnoreRecordNotFound bool
	LogLevel             gormlogger.LogLevel
}

// NewGormLogger creates a new GORM logger adapter
func NewGormLogger(component string) *GormLogger {
	return &GormLogger{
		log:                  New(component),
		SlowThreshold:        200 * time.Millisecond,
		IgnoreRecordNotFound: true,
		LogLevel:             gormlogger.Warn,
	}
}

func (l *GormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

func (l *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Info {
		l.log.Info(ctx, fmt.Sprintf(msg, data...), nil)
	}
}

func (l *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Warn {
		l.log.Warn(ctx, fmt.Sprintf(msg, data...), nil)
	}
}

func (l *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Error {
		l.log.Error(ctx, fmt.Sprintf(msg, data...), nil)
	}
}

func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.LogLevel <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	fields := map[string]any{
		"duration_ms":   elapsed.Milliseconds(),
		"rows_affected": rows,
	}

	switch {
	case err != nil && l.LogLevel >= gormlogger.Error:
		// Ignore "record not found" errors if configured
		if l.IgnoreRecordNotFound && errors.Is(err, gorm.ErrRecordNotFound) {
			// Log at debug level instead of error
			fields["sql"] = sql
			l.log.Debug(ctx, "record not found", fields)
			return
		}
		fields["error"] = err.Error()
		fields["sql"] = sql
		l.log.Error(ctx, "database error", fields)

	case elapsed > l.SlowThreshold && l.SlowThreshold != 0 && l.LogLevel >= gormlogger.Warn:
		fields["sql"] = sql
		fields["slow_threshold_ms"] = l.SlowThreshold.Milliseconds()
		l.log.Warn(ctx, "slow query", fields)

	case l.LogLevel >= gormlogger.Info:
		fields["sql"] = sql
		l.log.Debug(ctx, "query executed", fields)
	}
}
