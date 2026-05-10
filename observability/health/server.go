package health

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewServer constructs the per-binary HTTP server hosting probes and
// (optionally) Prometheus metrics. The returned server is unstarted —
// the caller drives ListenAndServe and Shutdown.
//
// The mux exposes:
//
//   - PathLivez (/health/livez): livenessProbe target, HTTP 200
//     unconditionally while the process serves.
//   - PathReadyz (/health/readyz): readinessProbe target with parallel
//     Pinger execution under the 800 ms internal deadline.
//   - PathMetrics (/metrics): Prometheus exposition for the supplied
//     registry; mounted only when registry is non-nil.
//
// An empty addr returns nil — useful for binaries that opt out of the
// probe surface entirely (one-shot CLIs, tests). A nil *http.Server is
// safe to use in defer chains so callers do not need a conditional
// branch around ListenAndServe.
//
// Shutdown should be ordered AFTER dependency pool closes (so /readyz
// naturally reports 503 as deps tear down) and BEFORE the metrics or
// tracer provider's Shutdown (so /metrics remains scrapable until the
// very end).
func NewServer(addr string, checks map[string]Pinger, registry *prometheus.Registry) *http.Server {
	if addr == "" {
		return nil
	}

	mux := http.NewServeMux()
	mux.Handle(PathLivez, LiveHandler())
	mux.Handle(PathReadyz, ReadyHandler(checks))
	if registry != nil {
		mux.Handle(PathMetrics, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	}

	return &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
}
