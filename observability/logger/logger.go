package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// New returns a slog.Logger that emits JSON to w at the requested
// minimum level. The level string is one of "debug", "info", "warn",
// "warning", or "error" (case- and whitespace-insensitive); unknown
// values default to "info".
//
// A nil w defaults to os.Stderr — the canonical log destination for
// containerised workloads.
//
// Timestamps are intentionally suppressed. The deployment platform
// (Kubernetes log driver, fluent-bit, log aggregator) attaches the
// authoritative timestamp at log ingestion; embedding a process-local
// timestamp duplicates state and risks clock skew between the binary
// and the collector. If a deployment needs in-process timestamps, use
// slog directly with a custom HandlerOptions instead of this helper.
func New(level string, w io.Writer) *slog.Logger {
	if w == nil {
		w = os.Stderr
	}
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: parseLevel(level),
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if len(groups) == 0 && a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}))
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
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
