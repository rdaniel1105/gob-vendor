package logger

import (
	"context"
	"io"
	"log/slog"
)

type logCtxKeyType int

var (
	logCtxKey logCtxKeyType = 1
)

// Set sets the logger in the context
func Set(ctx context.Context, log *slog.Logger) context.Context {
	return context.WithValue(ctx, logCtxKey, log)
}

// Get gets the logger from the context or creates a new one with an unknown service name
func Get(ctx context.Context) *slog.Logger {
	log, ok := ctx.Value(logCtxKey).(*slog.Logger)
	if !ok {
		log = slog.New(slog.NewJSONHandler(io.Discard, nil)) // initialize a default logger
	}

	return log
}
