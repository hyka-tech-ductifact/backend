package logging

import (
	"log/slog"
	"os"

	"ductifact/internal/config"
)

// NewLogger creates a configured slog.Logger based on the provided config.
//
// In production (Format="json"), it outputs JSON lines for machine parsing.
// In development (default), it outputs human-readable text.
//
// The log level is controlled by cfg.Level (debug, info, warn, error).
func NewLogger(cfg config.Log) *slog.Logger {
	level := parseLevel(cfg.Level)

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if cfg.Format == "json" {
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
