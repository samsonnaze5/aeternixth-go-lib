# Feature Specification: Health Probe Library

**Feature Branch**: `001-health-probes`
**Created**: 2026-05-10
**Status**: Draft
**Input**: User description: "Add observability/health package with probe handlers and dependency adapters"

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Worker service exposes Kubernetes probes (Priority: P1)

A developer building a Kafka-consumer worker binary (no inbound HTTP traffic) needs to expose Kubernetes liveness and readiness endpoints alongside Prometheus metrics on a sidecar HTTP port. Readiness must report whether the worker can still reach its dependencies (state DB pool, source DB pool, Kafka brokers and configured topics).

**Why this priority**: This is the dominant pattern in the fleet — `onetrust-feeder` (proven via PR #8) and `onetrust-mt5-processor` ship as worker fleets, plus the worker subset of `onetrust-finance-service` and `onetrust-archiver`. Without this, every repo writes the same ~50 lines of HTTP server + handler + JSON-shape boilerplate, with subtle drift over time.

**Independent Test**: Compose a worker binary that imports the library, registers any pinger under a chosen key, and starts the convenience server. `/health/livez` returns 200 unconditionally; `/health/readyz` returns 200 with the pinger reported under its key when the pinger succeeds, and 503 with the failure surfaced when it fails.

**Acceptance Scenarios**:

1. **Given** a worker with all configured pingers healthy, **When** the operator's `readinessProbe` calls `/health/readyz`, **Then** the response is HTTP 200 with `status: "ready"` and every check reported as `{"ok": true}`.
2. **Given** a worker with one pinger returning an error, **When** `/health/readyz` is called, **Then** the response is HTTP 503 with `status: "not_ready"` and the failing check carries an `error` field with the underlying message; sibling checks still report their actual results in the same response.
3. **Given** a worker whose readyz handler is mid-evaluation when the 800 ms internal deadline elapses, **When** Kubernetes' default `timeoutSeconds: 1` deadline approaches, **Then** the handler returns within 1 s total, with any not-yet-resolved pinger marked failed.
4. **Given** a worker started with an empty `addr` configuration, **When** the convenience server constructor is called, **Then** it returns nil so the caller can opt out without branching around `ListenAndServe`.

---

### User Story 2 — Fiber API service mounts probes alongside business routes (Priority: P2)

A developer maintaining a Fiber-based HTTP API (`onetrust-mt5-proxy-api`, `onetrust-client-portal-api`, plus the Fiber portion of `finance-service` and `archiver`) needs to add liveness and readiness probes to the existing Fiber app, on the same port as business routes, without disturbing an already-mounted `fiberprometheus` `/metrics` endpoint.

**Why this priority**: Fiber services already own a listener; forcing a sidecar port would fight their deployment model. The mount-style integration unblocks four of six target repos. P2 because worker repos (P1) carry more services per repo.

**Independent Test**: Compose a Fiber app with `fiberprometheus` mounted, call the library's mount helper, and verify `/health/livez`, `/health/readyz`, and `/metrics` all respond on the Fiber app's port — including the readiness JSON contract identical to the worker case.

**Acceptance Scenarios**:

1. **Given** a Fiber app with `fiberprometheus` already mounted at `/metrics`, **When** the developer calls the library's Fiber mount helper, **Then** `/health/livez` and `/health/readyz` respond on the same port and `/metrics` continues to respond from `fiberprometheus`.
2. **Given** a Fiber service that reports the same dependency map as a worker, **When** both binaries are queried at `/health/readyz`, **Then** the JSON response shape is byte-for-byte equivalent (same field names, same status conventions).

---

### User Story 3 — Reusable adapters for common dependencies (Priority: P2)

A developer wiring any service in the fleet needs ready-made pingers for the dependencies the fleet actually uses (Redis, GORM, Kafka) so they don't repeatedly write nil-safety, error-conversion, or Kafka Metadata-RPC plumbing.

**Why this priority**: GORM appears in four of six target services; Kafka in five of six. Without shared adapters, each repo re-derives the same wrapper, and per-repo divergence undermines the consistency benefit of having a shared library.

**Independent Test**: Given a mock or testcontainer-backed dependency, each adapter returns nil on success and a descriptive error on failure; constructing an adapter with a nil client and invoking ping must return an error without panic.

**Acceptance Scenarios**:

1. **Given** a healthy GORM connection, **When** the GORM adapter's pinger is invoked, **Then** it returns nil within the probe deadline.
2. **Given** a Kafka cluster where one configured topic has been deleted, **When** the Kafka adapter's pinger is invoked, **Then** it returns an error naming the missing topic so the operator's runbook step can identify the root cause without further inspection.
3. **Given** any adapter constructed with a nil client, **When** ping is invoked, **Then** a descriptive error is returned and the process does not panic.
4. **Given** a pgx connection pool (which already satisfies the pinger contract natively), **When** it is registered in the readiness map directly, **Then** it works without any library adapter.

---

### Edge Cases

- A pinger blocks past the 800 ms deadline: the handler returns immediately, marks that pinger failed (deadline-exceeded error), and reports sibling pingers' actual results — partial responses are not produced.
- `addr == ""` passed to the worker server constructor: the constructor returns nil; the caller's normal lifecycle code must skip `Listen` and `Shutdown` when nil.
- A pinger panics: the handler must surface a 5xx response and not crash the process. The library does not silently absorb panics — adapters are expected not to panic.
- Two services on the same host with the same probe paths: standard Kubernetes pod port isolation handles this; not a library concern.
- A caller-chosen readiness key collides with another caller's key in shared dashboards: keys are caller-namespaced inside the JSON body; collision is the operator's coordination problem, not the library's.
- Kafka rolling broker restart causes simultaneous readiness flap across all pods using the Kafka adapter: documented operational risk in ADR-0002; the library does not work around it.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The library MUST expose a `/health/livez` endpoint that returns HTTP 200 with body `{"status":"alive"}` unconditionally while the process is serving.
- **FR-002**: The library MUST expose a `/health/readyz` endpoint that returns HTTP 200 when every configured dependency check succeeds, and HTTP 503 when any check fails.
- **FR-003**: The readiness response body MUST follow the JSON contract documented in `CONTEXT.md`: `{"status": "ready"|"not_ready", "checks": {"<caller-key>": {"ok": bool, "error"?: string}}}`. The `error` field is omitted when `ok` is true.
- **FR-004**: All dependency checks for a single readiness call MUST run in parallel under a fixed 800 ms deadline, with individual check failures NOT cancelling sibling checks within the same call.
- **FR-005**: The library MUST provide a convenience HTTP server constructor (for non-Fiber callers) that mounts livez, readyz, and (optionally) Prometheus metrics on a single address. An empty address MUST cause the constructor to return nil rather than starting an unbound listener.
- **FR-006**: The library MUST provide a Fiber router mount helper that registers livez and readyz on an existing Fiber app. The Fiber adapter MUST NOT manage `/metrics` — Fiber adopters integrate Prometheus via their existing `fiberprometheus` setup.
- **FR-007**: The library MUST ship adapter sub-packages for Redis client, GORM connection, and Kafka (broker reachability with topic-existence verification via Metadata RPC).
- **FR-008**: Every adapter MUST reject a nil client with a descriptive non-panic error and surface dependency-side errors verbatim in the readiness response.
- **FR-009**: The Kafka adapter MUST require a caller-supplied broker list, topic list, and TLS configuration. The Metadata RPC MUST verify every supplied topic exists; missing topics MUST produce an error that names the topic.
- **FR-010**: Caller-provided check names ("keys") MUST appear unchanged in the JSON response; the library MUST NOT rewrite, normalize, or drop them, because runbooks parse on these strings.
- **FR-011**: The path constants for livez, readyz, and metrics MUST be exported from the core package so adopters can reference them without string-literal duplication.

### Key Entities

- **Pinger**: An abstract dependency that can answer "are you reachable?" via a single context-aware call. Identified in a probe by a caller-chosen string key.
- **Probe**: A single HTTP endpoint (`/health/livez` or `/health/readyz`) backed by zero or more pingers. Outputs the readiness JSON contract.
- **Adapter**: A small sub-package wrapping a specific third-party client (Redis, GORM, Kafka) so it conforms to the Pinger contract without each adopter writing the wrapper themselves.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All six target services (`feeder`, `mt5-processor`, `client-portal-api`, `finance-service`, `archiver`, `mt5-proxy-api`) can adopt the library by removing their own probe boilerplate and substituting library imports — verified by counting net lines deleted minus added per repo, summed across the fleet, returning a net reduction.
- **SC-002**: A worker service developer can wire livez + readyz + metrics in 10 lines or fewer of binding code, including imports.
- **SC-003**: A Fiber service developer can wire livez + readyz on an existing Fiber app in 5 lines or fewer of binding code, including imports.
- **SC-004**: A `/health/readyz` request never exceeds 1 second total response time even when one or more dependencies are unresponsive — measured against `failureThreshold` events in the library's own integration tests.
- **SC-005**: When a Kafka topic is deleted out from under a service that uses the Kafka adapter, the next `/health/readyz` call returns HTTP 503 with the missing topic name surfaced in the response body within one probe interval.
- **SC-006**: The library's own unit and integration tests cover every public symbol with at least one happy-path and one failure-path test case before the first repo adopts it.

## Assumptions

- Kubernetes default `timeoutSeconds: 1` for `livenessProbe` and `readinessProbe`. The library's 800 ms internal deadline is sized 200 ms below this. Clusters with custom timeoutSeconds must adjust deployment manifests independently.
- The Prometheus metrics endpoint is the responsibility of the worker convenience server (when given a non-nil registry) and explicitly out of scope for the Fiber adapter — Fiber callers already use `fiberprometheus`.
- pgx connection pools and ClickHouse connections already satisfy the Pinger contract natively. No library adapter ships for either; callers register them directly.
- The Kafka adapter ships only the Metadata RPC variant. Dial-only is rejected per ADR-0001; callers with a contrary need write a 3-line wrapper in their own repo.
- Path strings (`/health/livez`, `/health/readyz`, `/metrics`) are fixed in the library. They are referenced by Kubernetes manifests, runbooks, and dashboards — making them caller-configurable would defeat the consistency goal.
- The 800 ms internal probe deadline is fixed in the library. A future need to tune it signals a dependency that should be fixed at the dependency, not by widening the deadline.
- The library's `onetrust-feeder` consumer retains its existing `internal/kafka.CheckTopicsExist` for startup gating. Drift between feeder's startup check and the library's runtime probe is accepted; if it becomes painful, the function will be promoted into the library in a follow-up.
- Runbooks and dashboards in the operator org parse the readiness JSON shape as stable contract. Schema changes to the response body require coordinated runbook updates and constitute a major version bump for any future v1.0 of the library.
