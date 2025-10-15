package util

import (
	"fmt"
	"log/slog"
	"os"
)

// Fail logs a formatted fatal error if err is non-nil.
func Warning(err error, format string, args ...any) {
	if err != nil {
		slog.Warn("warning", "message", fmt.Sprintf(format, args...), "error", err)

	}
}

// Fail logs a formatted fatal error and terminates the program if err is non-nil.
func Fail(err error, format string, args ...any) {
	if err != nil {
		slog.Error("fatal", "message", fmt.Sprintf(format, args...), "error", err)
		os.Exit(1)
	}
}

// Annotate wraps an existing error with a message while preserving its cause.
// If msg is empty, the original error is returned unchanged.
func Annotate(err error, msg string) error {
	if err == nil {
		return nil
	}
	if msg == "" {
		return err
	}
	return fmt.Errorf("%s: %w", msg, err)
}
