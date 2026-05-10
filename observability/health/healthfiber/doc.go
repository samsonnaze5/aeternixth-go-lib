// Package healthfiber adapts the [github.com/samsonnaze5/aeternixth-go-lib/observability/health]
// handlers for Fiber-based services that mount probes alongside business
// routes on a single port.
//
// Use [Mount] to register both /health/livez and /health/readyz on an
// existing fiber.Router. The package does not own /metrics — Fiber
// adopters typically have fiberprometheus already mounted at /metrics on
// the same app, and this package leaves that path untouched.
package healthfiber
