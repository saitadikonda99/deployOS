// Package logging provides the structured JSON logger used across every
// DeployOS binary, built on the standard library's log/slog.
package logging

import (
	"log/slog"
	"os"
	"strings"
)

// New builds a JSON slog.Logger at the given level ("debug", "info",
// "warn", or "error"; unrecognized values fall back to "info"). Every
// record includes a timestamp and level by virtue of slog's JSON handler.
func New(level string) *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLevel(level),
	})
	return slog.New(handler)
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
