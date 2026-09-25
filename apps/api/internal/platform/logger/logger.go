// Package logger builds the application's *slog.Logger and carries a
// request-scoped copy through context, so code can log without threading a
// logger argument through every function signature.
package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// Config is the subset of app config the logger needs, kept separate from
// internal/config.Config to avoid an import cycle between the two packages.
type Config struct {
	Level  string // debug|info|warn|error
	Format string // text|json; empty means "json in production, text otherwise"
	Env    string
}

// New builds the base logger used before any request (or request_id) exists,
// e.g. for startup and shutdown lines.
func New(cfg Config) *slog.Logger {
	handler := newHandler(cfg, os.Stdout)
	return slog.New(handler)
}

func newHandler(cfg Config, w *os.File) slog.Handler {
	opts := &slog.HandlerOptions{Level: parseLevel(cfg.Level)}

	format := cfg.Format
	if format == "" {
		if cfg.Env == "production" {
			format = "json"
		} else {
			format = "text"
		}
	}

	if format == "json" {
		return slog.NewJSONHandler(w, opts)
	}
	return slog.NewTextHandler(w, opts)
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

type contextKey int

const loggerKey contextKey = 0

// WithContext returns a copy of ctx carrying l as the request-scoped logger.
func WithContext(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}

// FromContext returns the request-scoped logger, falling back to
// slog.Default so call sites never need a nil check.
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}
