package health

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/samsonnaze5/aeternixth-go-lib/observability/health/internal/core"
)

// fakePinger is a controllable Pinger for unit tests. delay sleeps before
// returning err (or nil); ctx cancellation is honored — when the deadline
// elapses during a delay, the pinger returns ctx.Err().
type fakePinger struct {
	err   error
	delay time.Duration
}

func (f *fakePinger) Ping(ctx context.Context) error {
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return f.err
}

func TestLiveHandler(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, PathLivez, nil)

	LiveHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type: want application/json, got %q", got)
	}

	if !bytes.Equal(rec.Body.Bytes(), core.LiveBody) {
		t.Errorf("body: want %q, got %q", core.LiveBody, rec.Body.Bytes())
	}
}

func TestReadyHandler_AllPass(t *testing.T) {
	checks := map[string]Pinger{
		"db":    &fakePinger{},
		"cache": &fakePinger{},
	}

	rec := httptest.NewRecorder()
	ReadyHandler(checks).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, PathReadyz, nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d (body=%s)", rec.Code, rec.Body.String())
	}

	var got core.ReadyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Status != "ready" {
		t.Errorf("status: want %q, got %q", "ready", got.Status)
	}
	if len(got.Checks) != 2 {
		t.Fatalf("checks: want 2 entries, got %d", len(got.Checks))
	}
	for name, c := range got.Checks {
		if !c.OK {
			t.Errorf("check %q: want OK, got false", name)
		}
		if c.Error != "" {
			t.Errorf("check %q: want no error, got %q", name, c.Error)
		}
	}
}

func TestReadyHandler_OneFail(t *testing.T) {
	checks := map[string]Pinger{
		"db":    &fakePinger{},
		"kafka": &fakePinger{err: errors.New("broker unreachable")},
	}

	rec := httptest.NewRecorder()
	ReadyHandler(checks).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, PathReadyz, nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status: want 503, got %d (body=%s)", rec.Code, rec.Body.String())
	}

	var got core.ReadyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Status != "not_ready" {
		t.Errorf("status: want %q, got %q", "not_ready", got.Status)
	}

	// Sibling check ("db") must still report its actual result.
	if !got.Checks["db"].OK {
		t.Errorf("sibling check db: want OK, got false (sibling-isolation invariant violated)")
	}
	if got.Checks["db"].Error != "" {
		t.Errorf("sibling check db: want no error, got %q", got.Checks["db"].Error)
	}

	// Failing check must surface error verbatim.
	kafka := got.Checks["kafka"]
	if kafka.OK {
		t.Errorf("kafka check: want failed, got OK")
	}
	if kafka.Error != "broker unreachable" {
		t.Errorf("kafka error: want %q, got %q", "broker unreachable", kafka.Error)
	}
}

func TestReadyHandler_AllFail(t *testing.T) {
	checks := map[string]Pinger{
		"db":    &fakePinger{err: errors.New("db down")},
		"kafka": &fakePinger{err: errors.New("kafka down")},
	}

	rec := httptest.NewRecorder()
	ReadyHandler(checks).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, PathReadyz, nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status: want 503, got %d", rec.Code)
	}

	var got core.ReadyResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	for name, c := range got.Checks {
		if c.OK {
			t.Errorf("check %q: want failed, got OK", name)
		}
		if c.Error == "" {
			t.Errorf("check %q: want error message, got empty", name)
		}
	}
}

func TestReadyHandler_DeadlineElapsed(t *testing.T) {
	// The blocker exceeds the 800 ms internal deadline; the fast pinger
	// returns immediately. The aggregator must report both within the
	// 1 s overall bound (G-3: deadline-elapsed reporting; G-2: sibling
	// isolation).
	checks := map[string]Pinger{
		"fast":    &fakePinger{},
		"blocker": &fakePinger{delay: 5 * time.Second},
	}

	start := time.Now()
	rec := httptest.NewRecorder()
	ReadyHandler(checks).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, PathReadyz, nil))
	elapsed := time.Since(start)

	if elapsed > 1*time.Second {
		t.Errorf("elapsed: want <= 1s, got %v (deadline did not bound the handler)", elapsed)
	}
	if elapsed < core.ProbeDeadline {
		t.Errorf("elapsed: want >= %v, got %v (deadline did not actually wait)", core.ProbeDeadline, elapsed)
	}
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status: want 503 (blocker timed out), got %d", rec.Code)
	}

	var got core.ReadyResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if !got.Checks["fast"].OK {
		t.Errorf("fast check: want OK (sibling preserved), got false")
	}
	if got.Checks["blocker"].OK {
		t.Errorf("blocker check: want failed (deadline exceeded), got OK")
	}
	if got.Checks["blocker"].Error == "" {
		t.Errorf("blocker check: want non-empty error from deadline-exceeded ctx, got empty")
	}
}

func TestReadyHandler_EmptyChecks(t *testing.T) {
	// No configured checks → nothing can fail → ready. Edge case for
	// CLI-style binaries that wire only liveness.
	rec := httptest.NewRecorder()
	ReadyHandler(map[string]Pinger{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, PathReadyz, nil))

	if rec.Code != http.StatusOK {
		t.Errorf("status: want 200 (no checks → ready), got %d", rec.Code)
	}
}
