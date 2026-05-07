package itestkit

import (
	"log"
	"testing"
)

// Logger receives diagnostic output from itestkit. The interface intentionally
// matches `testing.TB` so callers can pass `t` directly. Custom loggers (e.g.
// zerolog adapters) only need to implement Printf.
type Logger interface {
	Printf(format string, args ...any)
}

// nopLogger discards everything written to it. Used when no Logger and no
// testing.TB are provided.
type nopLogger struct{}

func (nopLogger) Printf(string, ...any) {}

// tbLogger adapts a testing.TB to the Logger interface by forwarding to t.Logf.
type tbLogger struct{ tb testing.TB }

func (l tbLogger) Printf(format string, args ...any) { l.tb.Logf(format, args...) }

// stdLogger forwards to the standard library logger. Used in non-test contexts.
type stdLogger struct{}

func (stdLogger) Printf(format string, args ...any) { log.Printf(format, args...) }

// resolveLogger returns the best available logger given the user-supplied
// logger and testing.TB. Precedence: explicit Logger > testing.TB > nop.
func resolveLogger(user Logger, tb testing.TB) Logger {
	if user != nil {
		return user
	}
	if tb != nil {
		return tbLogger{tb: tb}
	}
	return nopLogger{}
}
