# HTTP Contract: `/health/livez`, `/health/readyz`, `/metrics`

**Phase**: 1 (Design & Contracts)
**Audience**: operators writing Kubernetes manifests, runbook authors, dashboard owners

This file is the operator-facing surface. Anything documented here is **stable contract**. Renaming a field, status code, or path is a major version bump for the library.

## Endpoints

### `GET /health/livez`

| | |
|---|---|
| Purpose | Kubernetes `livenessProbe` target. Whether to restart the pod. |
| Status codes | `200 OK` while the process is serving. No other code is emitted. |
| Response Content-Type | `application/json` |
| Response body | `{"status":"alive"}` |
| Response body stability | The body is constant; runbooks may parse it but K8s only inspects status. |

### `GET /health/readyz`

| | |
|---|---|
| Purpose | Kubernetes `readinessProbe` target. Whether dependencies are reachable. |
| Status codes | `200 OK` when every configured pinger succeeds. `503 Service Unavailable` when any pinger fails. `500 Internal Server Error` for handler programming errors only — never expected in normal operation. |
| Response Content-Type | `application/json` |
| Response body shape | See *Readiness response schema* below. |
| Response time bound | < 1 s under K8s default `timeoutSeconds: 1` (enforced by the library's 800 ms internal deadline). |

### `GET /metrics`

| | |
|---|---|
| Purpose | Prometheus exposition for the registry passed to `health.NewServer`. |
| Mounted | Only when `health.NewServer` receives a non-nil `*prometheus.Registry`. **Not** mounted by the Fiber adapter — Fiber adopters use `fiberprometheus`. |
| Response | Standard Prometheus exposition format (`promhttp.HandlerFor`). |

## Readiness response schema

```json
{
  "status": "ready",
  "checks": {
    "<caller-key>": {"ok": true},
    "<caller-key>": {"ok": false, "error": "<message>"}
  }
}
```

| Field | Type | Required | Notes |
|---|---|---|---|
| `status` | string | yes | One of `"ready"` (when every check passes) or `"not_ready"` (when any fails). |
| `checks` | object | yes | Map keyed by caller-chosen string. Always present, possibly empty. |
| `checks.<key>.ok` | boolean | yes | `true` if the underlying pinger returned nil within the 800 ms deadline. |
| `checks.<key>.error` | string | no — `omitempty` | Underlying error message. Present only when `ok: false`. Verbatim — the library does not redact, normalize, or truncate. |

### Stable contract, parsed by

- Operator runbooks (e.g., `kubectl exec <pod> -- curl localhost:9090/health/readyz | jq -r '.checks | to_entries[] | select(.value.ok == false)'`)
- Internal dashboards that scrape readiness directly (uncommon, but supported)

### Out-of-contract for parsing

- Field ordering inside `checks` — JSON object iteration order is undefined.
- Whitespace, trailing newline, or `Content-Length` — these are HTTP details, not contract.

## Status code semantics — operator interpretation

| Code | What it means | Operator action |
|---|---|---|
| `200` (livez) | Process is up. | None. Continue running. |
| `200` (readyz) | All deps reachable. | Continue. Roll out new replicas if doing a deploy. |
| `503` (readyz) | At least one dep failing. Body identifies which. | Inspect failing check; consult dep-specific runbook. |
| `503` (readyz) flapping every probe | Likely Kafka rolling broker restart (per ADR-0002). | Verify via lag metrics and broker logs; do **not** restart pods. Probe spec should already absorb this — if it doesn't, raise `failureThreshold`. |
| `500` (any) | Library handler bug or panic. | File a lib issue; pod likely also misbehaving. |

## Deadline behavior contract

- Every individual pinger gets ≤ 800 ms before the deadline elapses.
- Pinger failures **do not** cancel sibling pingers in the same call. Every configured check is reported in the response, even when one fails immediately.
- When the 800 ms deadline elapses with one or more pingers still running, those pingers are reported `ok: false` with a context-deadline-exceeded error message; sibling pingers' actual results from the same call are still reported.

## Sample interactions

### Healthy worker, three pingers

```http
GET /health/readyz HTTP/1.1
Host: pod-internal:9090
```

```http
HTTP/1.1 200 OK
Content-Type: application/json

{"status":"ready","checks":{"state_pool":{"ok":true},"source_pool":{"ok":true},"kafka":{"ok":true}}}
```

### Kafka topic deleted

```http
HTTP/1.1 503 Service Unavailable
Content-Type: application/json

{"status":"not_ready","checks":{"state_pool":{"ok":true},"source_pool":{"ok":true},"kafka":{"ok":false,"error":"healthkafka: topic missing: feeder.deals.v1"}}}
```

### Liveness only

```http
GET /health/livez HTTP/1.1
```

```http
HTTP/1.1 200 OK
Content-Type: application/json

{"status":"alive"}
```
