package health

import (
	"context"

	"github.com/samsonnaze5/aeternixth-go-lib/observability/health/internal/core"
)

// Probe HTTP path constants — stable contract referenced by Kubernetes
// probe specs, runbooks, and dashboards. Changes here require coordinated
// updates to deployment manifests across every adopting service and
// constitute a major-version event for the package.
const (
	// PathLivez is the path Kubernetes livenessProbe targets. It returns
	// HTTP 200 with body {"status":"alive"} unconditionally while the
	// process serves.
	PathLivez = "/health/livez"

	// PathReadyz is the path Kubernetes readinessProbe targets. It
	// returns HTTP 200 when every configured Pinger succeeds, 503
	// otherwise. Per-check details are surfaced in the JSON body.
	PathReadyz = "/health/readyz"

	// PathMetrics is the Prometheus exposition path mounted by NewServer
	// when a non-nil *prometheus.Registry is supplied.
	PathMetrics = "/metrics"
)

// Pinger is the dependency contract every readiness check satisfies.
// Implementations answer "are you reachable?" via a single context-aware
// call; success is signalled by a nil error, failure by a non-nil error
// whose message will appear verbatim in the readiness JSON body.
//
// *pgxpool.Pool and clickhouse.Conn already satisfy this interface
// natively. The healthredis, healthgorm, and healthkafka sub-packages
// provide adapters for clients that do not. For one-off custom checks
// (HTTP API, gRPC health, custom lag thresholds), wrap a function with
// PingFunc.
//
// Implementations MUST respect ctx cancellation, MUST NOT panic on
// internal errors, and MUST be safe for concurrent invocation. See
// specs/001-health-probes/contracts/pinger-contract.md for the full
// invariant list.
type Pinger = core.Pinger

// PingFunc adapts an arbitrary function to the Pinger interface. It is
// the canonical primitive for one-off readiness checks that do not
// justify a typed adapter:
//
//	checks["upstream-api"] = health.PingFunc(func(ctx context.Context) error {
//	    req, _ := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
//	    resp, err := http.DefaultClient.Do(req)
//	    if err != nil {
//	        return err
//	    }
//	    defer resp.Body.Close()
//	    if resp.StatusCode != http.StatusOK {
//	        return fmt.Errorf("upstream-api: status %d", resp.StatusCode)
//	    }
//	    return nil
//	})
//
// Mirrors the http.HandlerFunc pattern from the standard library.
type PingFunc func(ctx context.Context) error

// Ping invokes the underlying function and returns its error verbatim.
// Implements the Pinger contract.
func (f PingFunc) Ping(ctx context.Context) error { return f(ctx) }
