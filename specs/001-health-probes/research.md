# Research: Health Probe Library

**Phase**: 0 (Outline & Research)
**Status**: complete — no `NEEDS CLARIFICATION` markers remain
**Inputs**: [spec.md](./spec.md), [plan.md](./plan.md), [docs/adr/0001](../../docs/adr/0001-kafka-pinger-uses-metadata-rpc.md), [docs/adr/0002](../../docs/adr/0002-kubernetes-probe-spec-guidance.md)

## Posture

The grilling session that produced the spec resolved every meaningful unknown ahead of writing the plan. The remaining job for Phase 0 is to record the technology choices behind each requirement so future readers see the alternatives that were considered, not just the conclusion.

## Decisions

### D1 — Parallel pinger execution under deadline

- **Decision**: `golang.org/x/sync/errgroup.WithContext` with `context.WithTimeout(ctx, 800*time.Millisecond)`. Each pinger goroutine returns `nil` to `g.Go` so individual failures do **not** cancel siblings; only the deadline cancels the aggregate.
- **Rationale**: Matches the merged `onetrust-feeder` PR #8 implementation, which has been running in production. errgroup is already a transitive dep; promoting to direct in `go.mod` is a one-line change and standard for any concurrency-bounded fan-out.
- **Alternatives considered**:
  - `sync.WaitGroup` + manual deadline plumbing — works but reinvents errgroup's cancellation discipline.
  - `select` over channels — readable for 1-2 checks, awkward for an arbitrary `map[string]Pinger`.
- **Reference**: spec FR-004; CONTEXT.md "Probe response contract" (sibling-isolation invariant).

### D2 — Kafka readiness check uses Metadata RPC

- **Decision**: `healthkafka.MetadataPinger` calls `kafka.Dial` to bootstrap then `Client.Metadata` to enumerate topics, asserting every supplied topic exists.
- **Rationale**: Documented in [ADR-0001](../../docs/adr/0001-kafka-pinger-uses-metadata-rpc.md). Catches the "topic accidentally deleted" failure mode that `kafka.Dial` alone would miss; matches semantics of the startup `CheckTopicsExist` call patterns adopters typically have.
- **Alternatives considered**: Dial-only; both flavors. Both rejected per ADR-0001.
- **Operational consequence**: ADR-0002 captures the K8s probe-spec guidance (high `failureThreshold`, generous `periodSeconds`).

### D3 — Adapter constructors return `(*T, error)`

- **Decision**: All three adapter packages (`healthredis`, `healthgorm`, `healthkafka`) export `New{Type}(...)` constructors that validate inputs (reject nil clients / empty broker or topic lists) and return `(*T, error)` with sentinel errors (`ErrNilClient`, `ErrNilDB`, `ErrEmptyBrokers`, `ErrEmptyTopics`).
- **Rationale**: Matches Constitution Principle III. Construction-time validation is fail-fast: the binary refuses to start if its readiness deps are misconfigured, instead of starting and reporting 503 forever.
- **Alternatives considered**: Direct struct literal (`&healthredis.Pinger{Client: c}`) with nil-check moved to `Ping`. Rejected because (a) constitution mandates validate-at-construction, (b) the failure surfaces in operator-noisy `/readyz` body instead of in startup logs where it belongs.

### D4 — Path constants and 800 ms deadline are fixed in lib

- **Decision**: `PathLivez = "/health/livez"`, `PathReadyz = "/health/readyz"`, `PathMetrics = "/metrics"`, internal deadline `800 * time.Millisecond` — all unexported as configurable knobs.
- **Rationale**: Spec Assumptions section. Paths are stable contract for runbooks, dashboards, and K8s manifests. The 800 ms deadline is sized below K8s default `timeoutSeconds: 1`; any team with a different `timeoutSeconds` adjusts the deployment manifest, not the library.
- **Alternatives considered**: Functional options (`WithLivezPath`, `WithDeadline`). Rejected during grilling — surface area without value, since 100 % of consumers use the defaults.

### D5 — `*pgxpool.Pool` and `clickhouse.Conn` registered directly (no adapter)

- **Decision**: No `healthpgx` or `healthclickhouse` sub-package. Adopters pass the pool/conn directly into the `map[string]Pinger`.
- **Rationale**: Both types satisfy `Ping(ctx context.Context) error` natively. Adding empty-wrapper packages is noise and forces consumers to import `aeternixth-go-lib/observability/health/healthpgx` for nothing.
- **Reference**: spec User Story 3 acceptance scenario 4.

### D6 — Fiber adapter does not own `/metrics`

- **Decision**: `healthfiber.Mount` registers `/health/livez` and `/health/readyz` only. No `/metrics` integration.
- **Rationale**: All Fiber adopters (`onetrust-mt5-proxy-api`, `onetrust-client-portal-api`) already mount `/metrics` via `github.com/ansrivas/fiberprometheus/v2`. Adding lib-side `/metrics` would cause double-mount on the same path or force an awkward "either/or" config.
- **Alternatives considered**: Optional `WithMetrics` on `Mount`. Rejected — adopters shouldn't have to read lib docs to discover that their existing fiberprometheus setup conflicts.

## Test infrastructure

- `testcontainers-go/modules/kafka` is already a direct dep (used in the existing `itestkit/` package). `healthkafka` integration tests reuse this.
- No new test infra needed for `healthredis` and `healthgorm`: redis tests use `redis/go-redis` against an in-memory `miniredis`-equivalent OR a stub `*redis.Client` whose `Ping` is intercepted; gorm tests use `gorm.io/driver/sqlite` against an in-memory database (already idiomatic in the project).

## Output

`research.md` (this file) — every spec requirement traceable to either a Decision (`D1..D6`) or to existing ADRs. Phase 1 may proceed.
