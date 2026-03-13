package log

import (
	"log/slog"
	"os"
)

func NewLogger(verbose bool) *slog.Logger {
	var level slog.Level

	if verbose {
		level = slog.LevelDebug
	} else {
		level = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})
	return slog.New(handler)
}
