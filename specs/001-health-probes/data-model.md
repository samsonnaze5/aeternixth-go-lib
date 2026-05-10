# Data Model: Health Probe Library

**Phase**: 1 (Design & Contracts)
**Inputs**: [spec.md](./spec.md), [plan.md](./plan.md), [research.md](./research.md), [CONTEXT.md](../../CONTEXT.md)

The library is stateless — it persists nothing. "Data model" here is the set of in-memory shapes that flow across the library's public boundary: the `Pinger` contract, the readiness response wire shape, and the parameter shapes for the public constructors.

## Core types

### `health.Pinger` (interface)

```go
type Pinger interface {
    Ping(ctx context.Context) error
}
```

- **Invariants**: must respect `ctx` cancellation; must not panic on internal errors; must return a non-nil error on failure (never `(error)(nil)` for an unhealthy state).
- **Naming**: every concrete implementer in this lib is named `Pinger` (in `healthredis`, `healthgorm`) or `MetadataPinger` (in `healthkafka`). `*pgxpool.Pool` and `clickhouse.Conn` already satisfy this contract via their existing `Ping(ctx) error` methods — no wrapper needed.

### Readiness map

```go
type Checks = map[string]Pinger  // type alias, not a new named type
```

- **Keys** are caller-chosen strings. They appear unchanged in the JSON response (FR-010).
- **Values** are any `Pinger`. The library does not validate keys (no length limit, no charset restriction) — operator runbooks own the convention.
- **Reserved keys**: none. Adopters may use any string; collisions across services are an operator coordination concern.
- The library will likely export `Checks` as a type alias for documentation clarity, but functions accept the underlying map type to match Go idiom.

### Probe response (wire shape)

The exact JSON shape is contract-stable per [CONTEXT.md](../../CONTEXT.md). Internal Go types:

```go
type checkResult struct {
    OK    bool   `json:"ok"`
    Error string `json:"error,omitempty"`
}

type readyResponse struct {
    Status string                 `json:"status"`  // "ready" | "not_ready"
    Checks map[string]checkResult `json:"checks"`
}

type liveResponse struct {
    Status string `json:"status"`  // always "alive"
}
```

- **Visibility**: both response types are unexported. Callers do not parse responses programmatically — humans read them via `kubectl exec curl`, K8s reads only the HTTP status.
- **Response shape stability**: any rename of `Status`, `Checks`, `OK`, or `Error` JSON tags is a major version bump for the lib (Constitution governance + spec Assumptions).

## Public function signatures

Documented here for one-place review. Final godoc lives next to the source.

### Core (`observability/health`)

```go
const (
    PathLivez   = "/health/livez"
    PathReadyz  = "/health/readyz"
    PathMetrics = "/metrics"
)

func LiveHandler() http.Handler

func ReadyHandler(checks map[string]Pinger) http.Handler

// addr=="" returns nil (opt-out for callers without a probe port).
// registry==nil omits /metrics from the mux.
func NewServer(
    addr string,
    checks map[string]Pinger,
    registry *prometheus.Registry,
) *http.Server
```

### Fiber (`observability/health/healthfiber`)

```go
func LiveHandler() fiber.Handler
func ReadyHandler(checks map[string]health.Pinger) fiber.Handler

// Registers GET PathLivez and GET PathReadyz on the supplied router.
func Mount(app fiber.Router, checks map[string]health.Pinger)
```

### Redis adapter (`observability/health/healthredis`)

```go
var ErrNilClient = errors.New("healthredis: nil *redis.Client")

type Pinger struct{ /* unexported field */ }

func NewPinger(client *redis.Client) (*Pinger, error)
func (p *Pinger) Ping(ctx context.Context) error
```

### GORM adapter (`observability/health/healthgorm`)

```go
var ErrNilDB = errors.New("healthgorm: nil *gorm.DB")

type Pinger struct{ /* unexported field */ }

func NewPinger(db *gorm.DB) (*Pinger, error)
func (p *Pinger) Ping(ctx context.Context) error
```

### Kafka adapter (`observability/health/healthkafka`)

```go
var (
    ErrEmptyBrokers = errors.New("healthkafka: brokers list is empty")
    ErrEmptyTopics  = errors.New("healthkafka: topics list is empty")
    ErrTopicMissing = errors.New("healthkafka: topic missing")  // wrapped with topic name via fmt.Errorf("%w: %s", ErrTopicMissing, topic)
)

type MetadataPinger struct{ /* unexported fields */ }

// tls may be nil to disable TLS.
func NewMetadataPinger(brokers, topics []string, tls *tls.Config) (*MetadataPinger, error)
func (p *MetadataPinger) Ping(ctx context.Context) error
```

## Cardinality and lifecycle

- A binary owns **at most one** probe HTTP server (workers via `health.NewServer`, Fiber apps via `healthfiber.Mount`).
- A probe server hosts **exactly two** probe endpoints (`/health/livez`, `/health/readyz`) and **zero or one** metrics endpoint (`/metrics` only when a non-nil `*prometheus.Registry` is supplied to `NewServer`).
- A `Pinger` instance is constructed once at binary startup and reused across every `/health/readyz` invocation. It is **stateless** — concrete adapters wrap a long-lived client; per-invocation state lives in the goroutine spawned by `errgroup`.

## State transitions

The library has no state machine in the application sense. The only transitions are:

- **Probe response status**: `"ready"` (HTTP 200) when every check returns nil; `"not_ready"` (HTTP 503) otherwise. Determined per-call; no memory across calls.
- **Server lifecycle**: standard `*http.Server` — caller drives `ListenAndServe()` then `Shutdown(ctx)`. Per ADR-0001 (referenced in feeder PR), shutdown order should be _between_ pool closes and Provider.Shutdown so `/readyz` reports failure as deps tear down.
