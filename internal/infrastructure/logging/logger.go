package logging

import (
	"log/slog"
	"os"
)

// NewLogger creates a configured slog.Logger based on the environment.
//
// In production (LOG_FORMAT=json), it outputs JSON lines for machine parsing.
// In development (default), it outputs human-readable text.
//
// The log level is controlled by LOG_LEVEL env var (debug, info, warn, error).
// Defaults to "info" if not set.
func NewLogger() *slog.Logger {
	level := parseLevel(os.Getenv("LOG_LEVEL"))

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if os.Getenv("LOG_FORMAT") == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

// parseLevel converts a string level to slog.Level.
// Defaults to INFO if the string is not recognized.
func parseLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
