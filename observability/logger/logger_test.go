package logger_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/samsonnaze5/aeternixth-go-lib/observability/logger"
)

// parseLines decodes one JSON object per non-empty line from buf. Used
// by the level-filter and timestamp-drop tests.
func parseLines(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	out := []map[string]any{}
	for _, line := range strings.Split(buf.String(), "\n") {
		if line == "" {
			continue
		}
		var got map[string]any
		if err := json.Unmarshal([]byte(line), &got); err != nil {
			t.Fatalf("decode %q: %v", line, err)
		}
		out = append(out, got)
	}
	return out
}

func TestNew_DropsTimestamp(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New("info", &buf)

	log.Info("hello")

	lines := parseLines(t, &buf)
	if len(lines) != 1 {
		t.Fatalf("want 1 line, got %d", len(lines))
	}
	if _, has := lines[0]["time"]; has {
		t.Errorf("time field should be dropped from output (got %v)", lines[0])
	}
	if lines[0]["msg"] != "hello" {
		t.Errorf("msg: want %q, got %v", "hello", lines[0]["msg"])
	}
}

func TestNew_LevelFiltering(t *testing.T) {
	cases := []struct {
		level     string
		emit      func(*log_helper)
		wantLines int
	}{
		{"debug", func(l *log_helper) { l.Debug(); l.Info(); l.Warn(); l.Error() }, 4},
		{"info", func(l *log_helper) { l.Debug(); l.Info(); l.Warn(); l.Error() }, 3}, // debug filtered
		{"warn", func(l *log_helper) { l.Debug(); l.Info(); l.Warn(); l.Error() }, 2},
		{"WARNING", func(l *log_helper) { l.Debug(); l.Info(); l.Warn(); l.Error() }, 2}, // case + alias
		{"error", func(l *log_helper) { l.Debug(); l.Info(); l.Warn(); l.Error() }, 1},
		{"unknown-value", func(l *log_helper) { l.Debug(); l.Info(); l.Warn(); l.Error() }, 3}, // → info
		{"  ", func(l *log_helper) { l.Info() }, 1},                                            // whitespace → info
	}
	for _, tc := range cases {
		t.Run(tc.level, func(t *testing.T) {
			var buf bytes.Buffer
			log := logger.New(tc.level, &buf)
			tc.emit(&log_helper{Logger: log})
			got := len(parseLines(t, &buf))
			if got != tc.wantLines {
				t.Errorf("level=%q: want %d lines, got %d (%s)", tc.level, tc.wantLines, got, buf.String())
			}
		})
	}
}

func TestNew_NilWriter_DoesNotPanic(t *testing.T) {
	// Documents the contract: nil writer is allowed and defaults to
	// os.Stderr. Not asserting on stderr capture; just that construction
	// + a single log call don't panic.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panic with nil writer: %v", r)
		}
	}()
	log := logger.New("error", nil)
	log.Error("nil-writer ok")
}

// log_helper wraps slog.Logger with named methods so the table-driven
// emit closures stay readable.
type log_helper struct{ *slog.Logger }

func (l *log_helper) Debug() { l.Logger.Debug("d") }
func (l *log_helper) Info()  { l.Logger.Info("i") }
func (l *log_helper) Warn()  { l.Logger.Warn("w") }
func (l *log_helper) Error() { l.Logger.Error("e") }
