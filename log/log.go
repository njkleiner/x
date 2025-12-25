package log

import (
	"context"
	"log/slog"
	"runtime"
	"time"
)

// timeNow allows overriding the "current" timestamp
// used for creating new log records during testing.
var timeNow func() time.Time = time.Now

type contextKey struct{}

func Logger(ctx context.Context) *slog.Logger {
	if log, ok := ctx.Value(contextKey{}).(*slog.Logger); ok {
		return log // return the context-associated logger
	}

	return slog.Default() // otherwise return the default logger
}

func Context(ctx context.Context, log *slog.Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, log)
}

func With(ctx context.Context, args ...any) context.Context {
	return Context(ctx, Logger(ctx).With(args...))
}

func Err(err error) slog.Attr {
	return slog.Any("error", err)
}

func handle(ctx context.Context, lvl slog.Level, msg string, args ...any) {
	log := Logger(ctx)

	if !log.Enabled(ctx, lvl) {
		return // drop message
	}

	var pc [1]uintptr

	// NOTE: skip [runtime.Callers, this function, this function's caller]
	runtime.Callers(3, pc[:])

	r := slog.NewRecord(timeNow().UTC(), lvl, msg, pc[0])
	r.Add(args...) // add the given arguments to the record

	_ = log.Handler().Handle(ctx, r) // NOTE: we ignore the error (if any)
}

func Debug(ctx context.Context, msg string, args ...any) {
	handle(ctx, slog.LevelDebug, msg, args...)
}

func Info(ctx context.Context, msg string, args ...any) {
	handle(ctx, slog.LevelInfo, msg, args...)
}

func Warn(ctx context.Context, msg string, args ...any) {
	handle(ctx, slog.LevelWarn, msg, args...)
}

func Error(ctx context.Context, msg string, args ...any) {
	handle(ctx, slog.LevelError, msg, args...)
}
