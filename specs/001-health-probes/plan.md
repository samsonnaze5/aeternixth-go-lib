# Implementation Plan: Health Probe Library

**Branch**: `001-health-probes` | **Date**: 2026-05-10 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-health-probes/spec.md`

## Summary

Add an `observability/health` package to `aeternixth-go-lib` that ships Kubernetes-style liveness and readiness probes plus dependency adapters (Redis, GORM, Kafka), reusable across the six Go services in the fleet. The core exposes a `Pinger` interface, parallel-execution readiness handler with a fixed 800 ms internal deadline, a Fiber router mount helper, and a convenience `*http.Server` constructor for non-Fiber callers. Adapter sub-packages (`healthredis`, `healthgorm`, `healthkafka`) each provide a `New{Type}` constructor that validates inputs and returns `(*T, error)` per the project constitution. Library implementation reproduces the merged `onetrust-feeder` PR #8 pattern with two changes: caller-keyed `map[string]Pinger` instead of feeder-named struct, and the `MetadataPinger` impl is owned by the lib (per ADR-0001 / ADR-0002 already committed in `docs/adr/`).

## Technical Context

**Language/Version**: Go 1.25.0 (matches `go.mod`)
**Primary Dependencies**:
- `github.com/gofiber/fiber/v2` — Fiber adapter only
- `github.com/jackc/pgx/v5/pgxpool` — already satisfies Pinger natively, no adapter
- `github.com/redis/go-redis/v9` — wrapped by `healthredis`
- `gorm.io/gorm` — wrapped by `healthgorm`
- `github.com/segmentio/kafka-go` — wrapped by `healthkafka.MetadataPinger`
- `github.com/prometheus/client_golang/prometheus` and `.../prometheus/promhttp` — only for `NewServer` metrics mount
- `golang.org/x/sync/errgroup` — parallel ping execution under deadline (NEW in lib's direct deps; transitive already)

**Storage**: N/A (library — no persistence)
**Testing**:
- `testing` (stdlib) for unit tests, table-driven where multiple inputs share logic
- `net/http/httptest` for handler tests
- `github.com/testcontainers/testcontainers-go/modules/kafka` for `healthkafka` integration tests (already a lib dep)
- `-race` for the parallel-execution paths in core
- `go test -bench=. -benchmem` for the readiness aggregator (constitution IV)

**Target Platform**: Go-supported platforms; linux/amd64 + linux/arm64 in production
**Project Type**: library (single module — extends existing `aeternixth-go-lib`)
**Performance Goals**:
- `/health/readyz` total response < 1 s under K8s default `timeoutSeconds: 1` (the 800 ms internal deadline gives 200 ms buffer)
- Readiness aggregator overhead < 50 µs over the slowest pinger (parallel execution; no serial hops)
- `LiveHandler` allocates the response body once at construction and reuses on every request

**Constraints**:
- Fixed paths (`/health/livez`, `/health/readyz`, `/metrics`) — not configurable
- Fixed 800 ms internal deadline — not configurable
- JSON response shape stable per `CONTEXT.md` — never reshape without major version bump
- No new direct cross-package imports inside the lib (`observability/health/*` is a new top-level subtree; sub-packages depend on `observability/health` only)

**Scale/Scope**: 6 consumer repos initially (`feeder`, `mt5-processor`, `client-portal-api`, `finance-service`, `archiver`, `mt5-proxy-api`); typical readiness map size 1-3 pingers per binary; fleet-wide ~25-30 binaries when worker subcommands are counted

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Verify the plan against the four principles in `.specify/memory/constitution.md`.

### I. Code Quality & Simplicity — **PASS**

- New packages introduced: `observability/health`, `observability/health/healthfiber`, `observability/health/healthredis`, `observability/health/healthgorm`, `observability/health/healthkafka`. Each is independently buildable.
- New cross-package edges: `healthfiber → health`, `healthredis → health`, `healthgorm → health`, `healthkafka → health`. These extend the documented dependency graph in CLAUDE.md (alongside the existing `response → errors`, etc.). The CLAUDE.md update lands in this plan's Phase 1 step "Agent context update".
- Every exported identifier in the new packages will carry godoc with at least one runnable example (covered in Phase 1 contracts and the quickstart).
- Sentinel errors planned (where applicable): `healthredis.ErrNilClient`, `healthgorm.ErrNilDB`, `healthkafka.ErrEmptyBrokers`, `healthkafka.ErrEmptyTopics`, `healthkafka.ErrTopicMissing` (wrapped with topic name). All `Err{Description}` and `errors.Is`-comparable.
- No speculative abstractions: rejected `DialPinger`, `healthpgx`, configurable paths, configurable deadline, functional options (each documented in spec Assumptions; rationale in ADRs).

### II. Testing Standards (NON-NEGOTIABLE) — **PASS**

- Every export gets a unit test for happy + at least one failure path:
  - `LiveHandler`: 200 + body shape
  - `ReadyHandler`: all-pass (200), one-fail (503 with error surfaced), all-fail, deadline-elapsed (sibling preservation)
  - `NewServer`: nil on empty addr, non-nil with mux mounted, registry=nil omits `/metrics`
  - `healthfiber.LiveHandler/ReadyHandler/Mount`: same matrix via Fiber test app
  - Adapters: nil-client (constructor error), happy ping, dependency-side error surface
- Cross-package integration: `healthkafka` against testcontainers Kafka (broker reachable, broker stopped, topic deleted)
- No mocking of internal lib packages (Pinger interface is the seam — mocks live in test files, not as exported types)
- The plan includes regression coverage for the deadline-elapsed sibling-preservation case, which is the subtle invariant from `feeder` PR #8

### III. API Consistency & Developer Experience — **PASS** (with one documented idiom for `NewServer`)

- Adapter constructors follow `New{Type}(...) (*T, error)`:
  - `healthredis.NewPinger(client *redis.Client) (*Pinger, error)`
  - `healthgorm.NewPinger(db *gorm.DB) (*Pinger, error)`
  - `healthkafka.NewMetadataPinger(brokers, topics []string, tls *tls.Config) (*MetadataPinger, error)`
- Handler constructors return `http.Handler` / `fiber.Handler` (interface return — matches stdlib idiom for `http.HandlerFunc`, `promhttp.Handler`):
  - `health.LiveHandler() http.Handler`
  - `health.ReadyHandler(checks map[string]Pinger) http.Handler`
  - `healthfiber.LiveHandler() fiber.Handler`
  - `healthfiber.ReadyHandler(checks map[string]health.Pinger) fiber.Handler`
- Convenience server: `health.NewServer(addr string, checks map[string]Pinger, registry *prometheus.Registry) *http.Server`. Returns nil for empty addr (documented opt-out, not an error). This matches stdlib `http.NewServeMux() *http.ServeMux` — constructors with no input that can fail validate-vacuously and skip the `error` return. **Documented in Complexity Tracking below.**
- No `Null{Type}` introduced (no nullable values in this surface).
- No new generics (no `interface{}` to remove).
- Directory and package names match (no `*util` divergence) — sub-packages use `health{x}` prefix to avoid collision when a caller imports both the adapter sub-package and its underlying client (e.g., `healthredis` + `go-redis/redis` in the same file).

### IV. Performance Requirements — **PASS**

- Hot-path benchmarks planned:
  - `BenchmarkReadyHandler_AllPass_5Pingers` — measures parallel-execution overhead with successful pingers
  - `BenchmarkReadyHandler_OneSlow_5Pingers` — measures deadline-bounded path
  - `BenchmarkLiveHandler` — measures the constant-body fast path
- Allocations target: ReadyHandler ≤ 1 alloc per check + 1 for the response struct (`go test -bench=. -benchmem`)
- Concurrency: ReadyHandler uses `errgroup.WithContext` with a deadline-bound context; goroutines terminate deterministically when the deadline elapses (no leaks). `-race` covered via the deadline-elapsed test case.
- Adapter Ping methods are not "hot path" (called every 15-30 s by K8s probes); no benchmarks required for them per constitution.

### Constitution Check verdict: **PASS** — proceed to Phase 0 research

## Project Structure

### Documentation (this feature)

```text
specs/001-health-probes/
├── plan.md              # this file
├── research.md          # Phase 0 — research findings (technology choices, references)
├── data-model.md        # Phase 1 — Pinger contract, response schema, ProbeDeps shape
├── quickstart.md        # Phase 1 — copy-paste wiring snippets for worker + Fiber
├── contracts/
│   ├── http-contract.md # /health/livez + /health/readyz + /metrics paths, status codes
│   └── pinger-contract.md  # Pinger interface invariants (parallel exec, deadline, no panic)
└── checklists/
    └── requirements.md  # already produced by /speckit-specify
```

### Source Code (repository root)

```text
aeternixth-go-lib/
├── observability/
│   └── health/
│       ├── doc.go                    # package overview + JSON contract reference
│       ├── health.go                 # Pinger, ProbeDeps map type alias (optional), constants
│       ├── handlers.go               # LiveHandler, ReadyHandler
│       ├── server.go                 # NewServer (net/http convenience)
│       ├── handlers_test.go          # unit tests for handlers + parallel exec invariants
│       ├── server_test.go            # NewServer paths (nil addr, with/without registry)
│       ├── healthfiber/
│       │   ├── healthfiber.go        # LiveHandler, ReadyHandler, Mount
│       │   └── healthfiber_test.go
│       ├── healthredis/
│       │   ├── healthredis.go        # Pinger struct + NewPinger constructor
│       │   └── healthredis_test.go
│       ├── healthgorm/
│       │   ├── healthgorm.go         # Pinger + NewPinger
│       │   └── healthgorm_test.go
│       └── healthkafka/
│           ├── healthkafka.go        # MetadataPinger + NewMetadataPinger
│           ├── checktopics.go        # internal helper using segmentio/kafka-go
│           ├── healthkafka_test.go   # unit tests with mock kafka client
│           └── healthkafka_integration_test.go  # testcontainers — Kafka broker scenarios
├── observability/  # (this feature only — sibling like tracer/, logger/ may follow)
└── (existing packages unchanged: aws/, decimal/, errors/, fiber/, gmail/, jwt/, ...)
```

**Structure Decision**: New top-level directory `observability/` with `health/` as the first child. Sub-packages of `health/` carry the `health` prefix in package name to avoid collision when an adopter imports the adapter alongside its underlying client (`healthredis` next to `redis`, `healthkafka` next to `kafka`, etc.) — chosen during the grilling session (Q7 = C). Future `observability/tracer/`, `observability/logger/` may follow the same shape but are out of scope for this plan.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| `health.NewServer` returns `*http.Server` (not `(*http.Server, error)`) — drift from constitution III "constructors return `(*T, error)`" | The function's only failure mode is "addr is empty," which is a documented opt-out (caller wants no probe server), not an error. Returning nil for empty addr matches `http.NewServeMux() *http.ServeMux` and similar stdlib constructors. Surfacing it as `(srv, err)` where err is always nil would be misleading. | `(srv, err) error` always-nil pattern misleads callers into checking err that can never fire. Renaming to drop "New" prefix (e.g., `BuildServer`) breaks idiom and surprises adopters. |
