// Package health provides Kubernetes-style liveness and readiness probes
// for backend services.
//
// The package exposes:
//
//   - The [Pinger] interface — the single-method contract that every
//     readiness check satisfies. *pgxpool.Pool and clickhouse.Conn already
//     satisfy it natively; the healthredis, healthgorm, and healthkafka
//     sub-packages provide adapters for the rest.
//
//   - [LiveHandler] and [ReadyHandler] — net/http handlers that callers
//     mount on their own mux (workers) or via the healthfiber sub-package
//     (Fiber apps).
//
//   - [NewServer] — convenience constructor returning an *http.Server with
//     all three endpoints (livez, readyz, optional metrics) mounted on a
//     single address. Returns nil for empty addr so callers can opt out
//     without conditional branches.
//
// The /health/readyz endpoint runs every configured Pinger in parallel
// under a fixed 800 ms internal deadline (sized below the Kubernetes
// default timeoutSeconds of 1). Per-check failures do not cancel siblings
// — every configured check is reported in the response body, with the
// failing one carrying its underlying error message.
//
// The JSON response shape is stable contract for runbooks and dashboards.
// See CONTEXT.md at the repository root for terminology, response schema,
// and recommended Kubernetes probe spec.
package health
