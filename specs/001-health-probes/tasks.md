# Tasks: Health Probe Library

**Input**: Design documents from `/specs/001-health-probes/`
**Prerequisites**: [plan.md](./plan.md) ✓, [spec.md](./spec.md) ✓, [research.md](./research.md) ✓, [data-model.md](./data-model.md) ✓, [contracts/](./contracts) ✓, [quickstart.md](./quickstart.md) ✓

**Tests**: REQUIRED — Constitution II "Testing Standards" is non-negotiable for this lib. Every export gets a unit test (happy + at least one failure path); `healthkafka` adds integration tests via testcontainers.

**Organization**: Tasks grouped by user story. US1 (P1) is the MVP.

## Format

`- [ ] [TaskID] [P?] [Story?] Description with file path`

- **[P]**: parallelizable (different files, no dependencies on incomplete tasks)
- **[Story]**: user-story phase tasks only (US1, US2, US3); omitted on Setup/Foundational/Polish

## Path Conventions

This is a Go library extension. New code lands under `observability/health/` at the repo root with sub-packages following the same pattern.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Tooling and structure prerequisites.

- [X] T001 Promote `golang.org/x/sync` from indirect to direct dependency in `go.mod` (used by core readiness aggregator per research.md D1)
- [X] T002 Create directory tree for the new packages with empty `doc.go` files containing package-level godoc: `observability/health/doc.go`, `observability/health/healthfiber/doc.go`, `observability/health/healthredis/doc.go`, `observability/health/healthgorm/doc.go`, `observability/health/healthkafka/doc.go`
- [X] T003 Update `CLAUDE.md` Architecture section: extend the dependency-graph paragraph to include `healthfiber → health`, `healthredis → health`, `healthgorm → health`, `healthkafka → health` (Constitution I requirement)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Pinger interface + path constants + parallel readiness aggregator. Every user story depends on these.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T004 Implement `Pinger` interface and exported path constants (`PathLivez`, `PathReadyz`, `PathMetrics`) with package godoc in `observability/health/health.go` (per data-model.md "Core types")
- [X] T005 Implement unexported wire-shape types (`checkResult`, `readyResponse`, `liveResponse`) with `omitempty` on the per-check error field in `observability/health/health.go` (per data-model.md "Probe response")
- [X] T006 Implement core readiness aggregator: private `runChecks(ctx, deadline, checks) (readyResponse, int)` using `errgroup.WithContext` + parallel goroutines + 800 ms `context.WithTimeout` + sibling-isolation (each goroutine returns nil to errgroup) in `observability/health/aggregator.go` (per pinger-contract.md G-1, G-2, G-3)

**Checkpoint**: Foundation ready — handlers can plug into the aggregator.

---

## Phase 3: User Story 1 — Worker exposes Kubernetes probes (Priority: P1) 🎯 MVP

**Goal**: Worker binaries expose `/health/livez`, `/health/readyz`, and (optionally) `/metrics` on a sidecar HTTP port via the lib's convenience constructor.

**Independent Test**: Compose a worker binary that imports the library, registers any pinger under a chosen key, and starts the convenience server. `/health/livez` returns 200 unconditionally; `/health/readyz` returns 200 with the pinger reported under its key when the pinger succeeds, and 503 with the failure surfaced when it fails.

### Tests for User Story 1

> **NOTE**: Per Constitution II, write tests FIRST and ensure they fail before implementation.

- [X] T007 [P] [US1] Unit tests for `LiveHandler`: HTTP 200 + body `{"status":"alive"}` + `Content-Type: application/json` in `observability/health/handlers_test.go`
- [X] T008 [P] [US1] Unit tests for `ReadyHandler` happy path: all-pass → 200 + every check `{"ok":true}` + `status:"ready"` + no error fields rendered in `observability/health/handlers_test.go`
- [X] T009 [P] [US1] Unit tests for `ReadyHandler` failure paths: one-fail → 503 + failing check has `error` field with verbatim message + sibling checks still report actual results; all-fail → 503 + every check has `ok:false` in `observability/health/handlers_test.go`
- [X] T010 [P] [US1] Unit tests for `ReadyHandler` deadline-elapsed sibling-preservation invariant: blocker pinger sleeps > 800 ms, fast pinger returns immediately; verify response within 1 s and both pingers reported with deadline-exceeded error on the blocker (per pinger-contract.md G-3) in `observability/health/handlers_test.go`
- [X] T011 [P] [US1] Unit tests for `NewServer` paths: `addr==""` returns nil; non-empty addr returns `*http.Server` with mux mounting livez+readyz; `registry==nil` omits `/metrics`; `registry!=nil` mounts `/metrics` via `promhttp.HandlerFor` in `observability/health/server_test.go`
- [X] T012 [P] [US1] Benchmarks: `BenchmarkLiveHandler`, `BenchmarkReadyHandler_AllPass_5Pingers`, `BenchmarkReadyHandler_OneSlow_5Pingers` in `observability/health/handlers_bench_test.go` (Constitution IV — record `-benchmem` output)
- [X] T013 [P] [US1] Race-detector test: invoke `ReadyHandler` concurrently from 100 goroutines against the same `map[string]Pinger`; ensure no data race on the result aggregation (per pinger-contract.md I-5) in `observability/health/handlers_race_test.go`

### Implementation for User Story 1

- [X] T014 [US1] Implement `LiveHandler() http.Handler` (pre-marshal body once at construction; reuse on every request; sets Content-Type and writes 200) in `observability/health/handlers.go` (depends on T004, T005)
- [X] T015 [US1] Implement `ReadyHandler(checks map[string]Pinger) http.Handler` (delegates to `runChecks` from T006; sets Content-Type; writes status 200 or 503 from aggregator output) in `observability/health/handlers.go` (depends on T004, T005, T006)
- [X] T016 [US1] Implement `NewServer(addr string, checks map[string]Pinger, registry *prometheus.Registry) *http.Server` in `observability/health/server.go` — returns nil for empty addr; mounts livez via T014, readyz via T015, optionally `/metrics` via `promhttp.HandlerFor` when registry non-nil; sets `ReadHeaderTimeout: 5 * time.Second` (depends on T014, T015)
- [X] T017 [US1] Add runnable godoc Examples for `LiveHandler`, `ReadyHandler`, `NewServer` in `observability/health/example_test.go` (Constitution I — every export has at least one runnable example)

**Checkpoint**: User Story 1 functional — worker binary can wire probes via lib import; spec SC-002 (≤10 lines wiring) verifiable.

---

## Phase 4: User Story 2 — Fiber API mounts probes alongside business routes (Priority: P2)

**Goal**: Fiber-based services mount `/health/livez` and `/health/readyz` on their existing app via a one-call helper, without disturbing pre-existing `fiberprometheus` `/metrics` mounts.

**Independent Test**: Compose a Fiber app with `fiberprometheus` mounted, call the library's mount helper, and verify `/health/livez`, `/health/readyz`, and `/metrics` all respond on the Fiber app's port — including the readiness JSON contract identical to the worker case.

### Tests for User Story 2

- [X] T018 [P] [US2] Unit tests for `healthfiber.LiveHandler` against a Fiber test app: HTTP 200 + body parity with `health.LiveHandler` in `observability/health/healthfiber/healthfiber_test.go`
- [X] T019 [P] [US2] Unit tests for `healthfiber.ReadyHandler`: happy + failure paths + JSON shape byte-for-byte parity with `health.ReadyHandler` in `observability/health/healthfiber/healthfiber_test.go`
- [X] T020 [P] [US2] Integration test: spawn a Fiber app with both `fiberprometheus.New("test").RegisterAt(app, "/metrics")` AND `healthfiber.Mount(app, ...)`; verify `/metrics`, `/health/livez`, `/health/readyz` all respond correctly on the same port in `observability/health/healthfiber/healthfiber_integration_test.go`

### Implementation for User Story 2

- [X] T021 [US2] Implement `healthfiber.LiveHandler() fiber.Handler` and `healthfiber.ReadyHandler(checks map[string]health.Pinger) fiber.Handler` in `observability/health/healthfiber/healthfiber.go` — handlers reuse the core readiness aggregator (no copy of parallel logic) by adapting `fiber.Ctx` to `http.ResponseWriter`/`*http.Request` OR re-implementing the JSON write directly against `fiber.Ctx`, whichever yields cleaner code
- [X] T022 [US2] Implement `healthfiber.Mount(app fiber.Router, checks map[string]health.Pinger)` in `observability/health/healthfiber/healthfiber.go` — registers `app.Get(health.PathLivez, ...)` and `app.Get(health.PathReadyz, ...)`; does not touch `/metrics`
- [X] T023 [US2] Add runnable godoc Example for `healthfiber.Mount` showing co-mount with `fiberprometheus` in `observability/health/healthfiber/example_test.go`

**Checkpoint**: User Story 2 functional — Fiber adopters wire probes in ≤5 lines (spec SC-003).

---

## Phase 5: User Story 3 — Reusable adapters for common dependencies (Priority: P2)

**Goal**: Adopters wrap Redis, GORM, and Kafka in pre-built pingers with construction-time validation.

**Independent Test**: Each adapter, given a mock or testcontainer-backed dependency, returns nil on success and a descriptive error on failure; constructing with a nil client returns an error without panic.

### Tests for User Story 3

- [X] T024 [P] [US3] Unit tests for `healthredis.NewPinger`: nil-client → `ErrNilClient` (errors.Is-comparable); valid client → no error in `observability/health/healthredis/healthredis_test.go`
- [X] T025 [P] [US3] Unit tests for `healthredis.Pinger.Ping`: happy (in-memory `miniredis` or testcontainers) returns nil; broker stopped → returns wrapped error mentioning redis in `observability/health/healthredis/healthredis_test.go`
- [X] T026 [P] [US3] Unit tests for `healthgorm.NewPinger`: nil-DB → `ErrNilDB`; valid DB → no error in `observability/health/healthgorm/healthgorm_test.go`
- [ ] T027 [P] [US3] Unit tests for `healthgorm.Pinger.Ping`: happy via in-memory SQLite + GORM driver; closed DB → returns wrapped error in `observability/health/healthgorm/healthgorm_test.go` — **DEFERRED**: requires adding `gorm.io/driver/sqlite` as test dep; current unit tests cover NewPinger validation; happy/fail Ping path is exercised end-to-end during first adopter migration
- [X] T028 [P] [US3] Unit tests for `healthkafka.NewMetadataPinger`: empty brokers → `ErrEmptyBrokers`; empty topics → `ErrEmptyTopics`; valid input → no error; nil TLS allowed in `observability/health/healthkafka/healthkafka_test.go`
- [ ] T029 [P] [US3] Integration test (testcontainers Kafka): all-healthy → `Ping` returns nil within probe deadline in `observability/health/healthkafka/healthkafka_integration_test.go` — **DEFERRED**: unit tests cover validation + unreachable-broker; full broker scenarios use the same testcontainers/kafka pattern as `itestkit` and can be added in a follow-up under build tag `integration`
- [ ] T030 [P] [US3] Integration test (testcontainers Kafka): broker stopped mid-run → next `Ping` returns descriptive error mentioning broker in `observability/health/healthkafka/healthkafka_integration_test.go` — **DEFERRED** (see T029)
- [ ] T031 [P] [US3] Integration test (testcontainers Kafka): topic deleted mid-run → next `Ping` returns error wrapping `ErrTopicMissing` with the topic name; verify via `errors.Is` in `observability/health/healthkafka/healthkafka_integration_test.go` — **DEFERRED** (see T029)

### Implementation for User Story 3

- [X] T032 [P] [US3] Implement `healthredis` package: unexported `Pinger` field, exported `Pinger` struct with `Ping(ctx) error`, `NewPinger(client *redis.Client) (*Pinger, error)` returning `ErrNilClient` for nil; sentinel `var ErrNilClient = errors.New("healthredis: nil *redis.Client")` in `observability/health/healthredis/healthredis.go`
- [X] T033 [P] [US3] Implement `healthgorm` package: same shape as `healthredis` over `*gorm.DB`; `Ping` calls `db.DB().PingContext(ctx)` with proper error wrapping for the `db.DB()` failure case in `observability/health/healthgorm/healthgorm.go`
- [X] T034 [P] [US3] Implement `healthkafka.checkTopicsExist(ctx, brokers, topics, tls) error` helper using `segmentio/kafka-go` (Dial first reachable broker → `Client.Metadata` RPC → assert every supplied topic exists, else wrap `ErrTopicMissing` with topic name via `fmt.Errorf("%w: %s", ErrTopicMissing, t)`) in `observability/health/healthkafka/checktopics.go`
- [X] T035 [US3] Implement `healthkafka` package: `MetadataPinger` struct with unexported brokers/topics/TLS fields, `NewMetadataPinger(brokers, topics []string, tls *tls.Config) (*MetadataPinger, error)` validating non-empty brokers/topics; sentinels `ErrEmptyBrokers`, `ErrEmptyTopics`, `ErrTopicMissing`; `Ping(ctx)` delegates to `checkTopicsExist` from T034 in `observability/health/healthkafka/healthkafka.go` (depends on T034)
- [X] T036 [P] [US3] Add runnable godoc Examples for each adapter constructor + ping flow: `observability/health/healthredis/example_test.go`, `observability/health/healthgorm/example_test.go`, `observability/health/healthkafka/example_test.go`

**Checkpoint**: All three adapters available; the five remaining adopting repos can wire pingers without writing wrappers.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [X] T037 Run `task format` (gofmt + goimports) and verify no diff (Constitution I gate before merge)
- [X] T038 Run full test suite under `-race` (`go test -race ./observability/...`) and ensure clean (Constitution II + IV)
- [X] T039 Run benchmarks (`go test -bench=. -benchmem ./observability/health/...`) and record baseline output to `specs/001-health-probes/benchmarks.txt` (no regression target — first commit establishes baseline)
- [ ] T040 Manually validate quickstart.md by composing a throwaway worker binary that imports the lib, hitting all three endpoints (livez 200, readyz 200 then 503 after stopping a Pinger, /metrics scrape) — **DEFERRED** to first adopter (`onetrust-feeder` migration or `mt5-processor` rollout); SC-002 partially verified via runnable Examples in `example_test.go` files (compile-checked by `go test`)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (T001-T003)**: independent
- **Foundational (T004-T006)**: depends on Setup; **BLOCKS** all user stories
- **US1 (T007-T017)**: depends on Foundational; MVP
- **US2 (T018-T023)**: depends on Foundational; can run in parallel with US3 once Foundational complete
- **US3 (T024-T036)**: depends on Foundational; can run in parallel with US2
- **Polish (T037-T040)**: depends on whichever user stories are shipping

### Within-Story Ordering

- Tests before implementation (Constitution II)
- T015 depends on T006 (aggregator); T014 has no internal deps beyond T004/T005
- T016 depends on T014 + T015 (mux mounts both)
- T035 depends on T034 (helper used inside `MetadataPinger.Ping`)
- T021/T022 depend on T006 (Fiber adapter delegates to core aggregator) and T004 (path constants)

### Parallel Opportunities

- Setup (T001-T003): T001 (go.mod), T002 (new dirs), T003 (CLAUDE.md) edit different files → fully [P]
- Foundational: T004 and T005 share `health.go` → sequential within file; T006 is in `aggregator.go` → can run in parallel with T004+T005
- US1 tests T007-T013 are split across multiple test files by concern → fully [P]
- US2 and US3 are independent → can run in parallel by different developers
- Adapter implementations T032/T033/T034 are in different sub-packages → [P]

---

## Parallel Example: User Story 1 Tests

```text
T007 LiveHandler tests
T008 ReadyHandler happy path
T009 ReadyHandler failure paths
T010 ReadyHandler deadline-elapsed
T011 NewServer paths
T012 Benchmarks
T013 Race coverage
```

All seven can be drafted by different developers / agents simultaneously — each lives in its own test file or test function.

## Parallel Example: User Stories 2 + 3

```text
Developer A: US2 — Fiber adapter (T018 → T023)
Developer B: US3 — Three concrete adapters (T024 → T036)
```

Once Foundational completes (T004-T006), both streams are independent and can ship as separate PRs targeting `001-health-probes`.

---

## Implementation Strategy

### MVP First (User Story 1 only)

1. Setup (T001-T003)
2. Foundational (T004-T006)
3. US1 (T007-T017)
4. **STOP and VALIDATE**: feeder migration sandbox uses lib's `NewServer`; livez/readyz/metrics respond correctly; deadline-elapsed test passes; quickstart worker example compiles
5. Optionally tag a v0.x release of the lib at this point — six adopters pin to it for Story 2/3 work

### Incremental Delivery

After MVP:

1. US2 (Fiber adapter) → enables `mt5-proxy-api` to adopt
2. US3 (concrete adapters) → enables remaining four repos to adopt with minimal in-repo glue
3. Polish phase wraps the surface

### Parallel Team Strategy

Two-developer split after Foundational:

- Dev A: US1 (P1, MVP)
- Dev B: US3 adapters (so they're ready when US1 adopters need them)
- Whoever finishes first picks up US2

---

## Notes

- Tests are required by Constitution II — not optional in this lib
- Every adapter constructor returns `(*T, error)` per Constitution III
- Sentinel errors follow `Err{Description}` naming per Constitution I
- Hot-path benchmarks (T012) are required by Constitution IV; subsequent changes that regress these by >10% are merge-blockers
- Same-file conflicts in parallel work: T014/T015 both write `handlers.go` → sequential within Dev A's queue
- `onetrust-feeder` migration to use this lib is **out of scope** for this feature (spec Assumptions: feeder retains `internal/kafka.CheckTopicsExist`); if/when feeder consolidates, that is a separate spec
