// Package logger provides minimal structured JSON logging using the standard library.
// Extension point for OpenTelemetry / metrics correlation later.
package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
)

type ctxKey struct{}

// New creates a JSON slog logger for the given service name.
func New(service string) *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	return slog.New(handler).With("service", service)
}

// NewWithWriter is useful in tests.
func NewWithWriter(service string, w io.Writer, level slog.Level) *slog.Logger {
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})
	return slog.New(handler).With("service", service)
}

// WithContext stores the logger in context.
func WithContext(ctx context.Context, log *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, log)
}

// FromContext returns the logger from context, or a default if missing.
func FromContext(ctx context.Context) *slog.Logger {
	if v, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok && v != nil {
		return v
	}
	return slog.Default()
}
