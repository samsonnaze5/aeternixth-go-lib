// Package core holds the shared readiness-evaluation primitives used by
// both the parent health package (net/http handlers) and the healthfiber
// sub-package (Fiber handlers). It is internal — the Go compiler forbids
// imports outside observability/health/...
//
// Splitting this into an internal package keeps the public API of
// observability/health small (just the Pinger contract, path constants,
// handler/server constructors, and PingFunc adapter) while the parallel
// execution machinery and shared response bytes are co-located here for
// re-use.
package core

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// Pinger is the dependency contract every readiness check satisfies.
// The parent observability/health package exposes this as health.Pinger
// via a type alias, so users see the canonical name in their imports
// while the internal machinery references the same interface.
type Pinger interface {
	Ping(ctx context.Context) error
}

// ProbeDeadline bounds each /health/readyz handler execution. Set 200 ms
// below Kubernetes' default timeoutSeconds: 1 so a single slow Pinger
// cannot trigger the probe-level connection timeout. Intentionally
// fixed — see the spec's Assumptions section for rationale.
const ProbeDeadline = 800 * time.Millisecond

// LiveBody is the constant body returned by the /health/livez handler.
// Shared between net/http and Fiber implementations so both runtimes
// emit byte-for-byte identical responses. Callers MUST NOT mutate the
// slice.
var LiveBody = []byte(`{"status":"alive"}`)

// CheckResult is one entry in the per-check map of a readyResponse. The
// Error field uses omitempty so happy-path bodies stay compact.
type CheckResult struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// ReadyResponse is the JSON shape /health/readyz emits. The schema is
// stable contract for runbook commands and dashboards (see CONTEXT.md
// "Probe response contract"). Exported so that tests in this internal
// package can introspect it; not surfaced to end users.
type ReadyResponse struct {
	Status string                 `json:"status"`
	Checks map[string]CheckResult `json:"checks"`
}

// Evaluate runs every Pinger in checks in parallel under ProbeDeadline
// and returns the marshalled JSON readiness body plus the corresponding
// HTTP status code (200 on all-pass, 503 when any check fails).
//
// Sibling-isolation invariant: an individual Pinger's failure does NOT
// cancel sibling Pingers — only the deadline cancels the aggregate.
// Every configured check appears in the response body regardless of
// the others' outcomes.
func Evaluate(ctx context.Context, checks map[string]Pinger) ([]byte, int) {
	resp, status := runChecks(ctx, checks)
	body, _ := json.Marshal(resp)
	return body, status
}

// runChecks is the parallel readiness aggregator. Given a map of named
// Pingers, it runs every check in parallel via errgroup.WithContext
// under ProbeDeadline, captures per-check errors, and returns a
// structured ReadyResponse plus the corresponding HTTP status code.
func runChecks(ctx context.Context, checks map[string]Pinger) (ReadyResponse, int) {
	cctx, cancel := context.WithTimeout(ctx, ProbeDeadline)
	defer cancel()

	g, gctx := errgroup.WithContext(cctx)

	var mu sync.Mutex
	resp := ReadyResponse{
		Status: "ready",
		Checks: make(map[string]CheckResult, len(checks)),
	}

	for name, p := range checks {
		g.Go(func() error {
			err := p.Ping(gctx)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				resp.Checks[name] = CheckResult{OK: false, Error: err.Error()}
			} else {
				resp.Checks[name] = CheckResult{OK: true}
			}
			return nil
		})
	}
	_ = g.Wait()

	statusCode := http.StatusOK
	for _, c := range resp.Checks {
		if !c.OK {
			resp.Status = "not_ready"
			statusCode = http.StatusServiceUnavailable
			break
		}
	}
	return resp, statusCode
}
