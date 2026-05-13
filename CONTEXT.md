# aeternixth-go-lib

A Go utility library for backend services. Most packages are flat technical utilities (`null`, `pagination`, `decimal`, etc.); this document captures domain language only for packages where terms have stable meanings that consumers (engineers, operators, runbooks, dashboards) depend on.

## observability/ — umbrella

`observability/` is the umbrella for cross-cutting "what's the system doing right now" primitives shared across every Go service in the fleet. `observability/health/` is the only populated sub-package today; planned siblings (`tracer/`, `logger/`, `metrics/`, `middleware/`) are documented in [observability/README.md](observability/README.md). Each lands when extraction would eliminate duplicated code in two or more adopters.

## observability/health — language

`observability/health` and its sub-packages (`healthfiber`, `healthredis`, `healthgorm`, `healthkafka`) provide Kubernetes-style liveness and readiness probes for services that consume this lib. Audience for these terms: engineers wiring up probes AND operators/SREs writing runbooks and Kubernetes probe specs against the resulting endpoints.

### Terms

**Live**:
An HTTP responder inside the process can return 200. Nothing more — no DB ping, no Kafka check, no last-tick freshness. Used by Kubernetes `livenessProbe` to decide whether to restart the pod. Restart is the only fix Live can deliver, so anything checked here must be unfixable except by restart.
_Avoid_: alive, up, running.

**Ready**:
Every external dependency the service needs to make forward progress is reachable, as evaluated by the configured set of **Pinger**s. Used by Kubernetes `readinessProbe`.
_Avoid_: healthy (overloaded with Live), available, serving.

**Pinger**:
Anything that satisfies `Ping(ctx context.Context) error` — the single-method contract every readiness check resolves to. The library provides adapter sub-packages for common dependencies (Redis, GORM, Kafka); `*pgxpool.Pool` and `clickhouse.Conn` already satisfy the interface natively. For one-off custom checks (HTTP API, gRPC health, custom lag thresholds), wrap a function with `PingFunc`.
_Avoid_: HealthCheck, Checker, Probe (Probe means the HTTP endpoint, not the dependency).

**PingFunc**:
A function adapter for **Pinger** — `health.PingFunc(myFunc)` makes any function with signature `func(context.Context) error` satisfy the contract. The canonical primitive for one-off readiness checks that don't justify a typed adapter, mirroring the `http.HandlerFunc` pattern from the standard library.
_Avoid_: PingFn (the trailing "Func" mirrors stdlib idiom).

**Probe**:
A single HTTP endpoint Kubernetes polls — `/health/livez` or `/health/readyz`. A **Probe** runs zero or more **Pinger**s in parallel under a fixed 800 ms internal deadline.
_Avoid_: HealthCheck (overloaded — could mean **Pinger** or **Probe**).

### Relationships

- **Live** ⊂ **Ready**: a binary cannot be **Ready** without being **Live**, but it can be **Live** without being **Ready** (e.g., its DB is temporarily unreachable).
- A **Probe** consumes a `map[string]Pinger`; the map keys appear unchanged in the JSON response, so they are stable contract for runbooks — do not rename keys in production code without coordinated runbook updates.
- The 800 ms internal **Probe** deadline is sized 200 ms below Kubernetes' default `timeoutSeconds: 1` and is intentionally not configurable.

### Probe response contract

`/health/livez` returns `{"status":"alive"}` with HTTP 200 unconditionally while the process serves.

`/health/readyz` returns:

```json
{
  "status": "ready",
  "checks": {
    "<caller-key>": {"ok": true},
    "<caller-key>": {"ok": false, "error": "..."}
  }
}
```

| Status | Meaning | When |
|---|---|---|
| `200 OK` | ready | every configured **Pinger** succeeded |
| `503 Service Unavailable` | not ready | at least one **Pinger** failed |

Failed checks include an `error` field with the underlying message; `omitempty` keeps happy-path bodies compact. Check-name keys are caller-provided.

### Example dialogue

> **Dev:** "Should `/readyz` fail when the upstream HTTP API our service calls is slow?"
> **Reviewer:** "Only if you can model that API as a **Pinger** and a runbook step exists for the failure case. **Ready** means 'dependencies reachable,' not 'business operation fast.' Tail-latency belongs in metrics and alerting, not in readiness."

### Flagged ambiguities

- "healthy" was used to mean both **Live** (process up) and **Ready** (dependencies reachable). Resolved: use **Live** and **Ready** explicitly; do not use "healthy" in package names, exported symbols, or runbook text.
- "HealthCheck" was used to mean both the HTTP endpoint (a **Probe**) and a single dependency check (a **Pinger**). Resolved: **Probe** is the endpoint, **Pinger** is the dependency.

## itestkit — language

`itestkit` is the integration-test infrastructure package: `StartStack` spins up real PostgreSQL/ClickHouse/Redis/Kafka/HTTP-mock/LocalStack containers via Testcontainers Go and returns connection information. The language below is shared between the lib and every consumer's `tests/integration/` package, so engineers moving between repos read the same surface.

### Terms

**Stack**:
The active set of running infrastructure dependencies created by `StartStack` and returned as `*itestkit.Stack`. One per `go test` process. Owns the Docker network, every container, and the `Cleanup` function. Every consumer holds it in a package-level `Stack` variable inside `tests/integration/bootstrap.go`.
_Avoid_: environment, fixtures, harness, containers.

**Instance**:
A named member of a **Service Map** — e.g., the `"main"` in `Postgres: map[string]PostgresOptions{"main": …}`. Container names, DSNs, env vars, testdata directories, and helper calls are all keyed by this name. Must match `^[a-z][a-z0-9_-]*$`. The first **Instance** of each service is canonically named `main` (Postgres, Kafka) / `cache` (Redis) / `events` (ClickHouse); consumers SHOULD override with a DDD-bounded-context or production-service name when one applies (e.g., `ledger`, `orders`).
_Avoid_: shard, tenant, database (a single **Instance** can host several databases).

**Service Map**:
The `map[string]XxxOptions` field per service type inside `StackOptions`. Empty map disables that service.
_Avoid_: services list, service config.

**Reset**:
The per-test isolation operation exposed as `itest.Reset(t)` in every consumer's `tests/integration/helpers.go`. TRUNCATEs every non-system table in every Postgres and ClickHouse **Instance**, `FLUSHALL`s every Redis **Instance**, and re-applies the original MockServer/WireMock expectations. Kafka and LocalStack are not reset — tests use per-test consumer-group IDs and per-test resource names instead.
_Avoid_: cleanup (cleanup is the **Stack** teardown registered with `t.Cleanup`), setup.

### Relationships

- A **Stack** has zero or more **Instances** per service type, indexed by the **Service Map** key.
- Every consumer helper is `Helper(t *testing.T, name string)` — `name` is the **Instance** name. No shorthand for the single-**Instance** case; explicit names everywhere.
- **Reset** assumes a sequential **Stack** — integration tests MUST NOT call `t.Parallel()`.

### Example dialogue

> **Dev:** "ใน test เราต้อง TRUNCATE table เองหรือเปล่า?"
> **Reviewer:** "ทุก test เริ่มด้วย `itest.Reset(t)`. มันจะ TRUNCATE ทุก table ใน **Instance** ของ Postgres/ClickHouse, FLUSHALL Redis, reset mocks ให้เอง. แต่ Kafka ไม่ถูก reset — test ต้องใช้ consumer group เฉพาะ (เช่น `t.Name()`) เพื่อไม่ให้ message ข้าม test."
