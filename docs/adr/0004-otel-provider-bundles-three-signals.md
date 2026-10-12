# ADR-0004: `observability/otel` bundles traces, metric bridge, and log pipeline

**Status:** accepted
**Date:** 2026-10-12

## Context

Three services carry a near-identical `internal/observability/otel.go`: `onetrust-feeder`, `onetrust-mt5-processor`, and `onetrust-finance-service` (PR #94). All three read the same config shape — `Enabled, Exporter, Endpoint, Insecure, SampleRate, SpanQueueSize, PrometheusBridge, MetricsInterval, LogsBridge` — and build one `Provider` owning a TracerProvider, a Prometheus → OTLP MeterProvider, and an OTLP LoggerProvider behind a single `Shutdown`.

`observability/tracer` (v0.6.0) covers traces only, has no adopters, and `observability/README.md` stated "tracer/ is OpenTelemetry traces, full stop".

A review of finance PR #94 against an OTel adoption guide also found defects that every copy shares:

- Exporter construction errors after the metric reader started leaked the reader goroutine.
- `otelgorm` starts a **root** span for every query issued without a parent span — background pollers (outbox, reconcilers, schedulers) flood the trace backend with single-span traces.
- No HTTP server span, so HTTP-originated work never forms a trace.

## Decision

1. New package `observability/otel` with one `Provider` for the three signals, matching the adopters' config shape. `observability/tracer` is deprecated in its favour (no adopters, so no migration cost).
2. Instrumentation adapters as self-contained sub-packages that do not import `observability/otel` (they use the OTel globals it installs):
   - `otel/otelkafka` — W3C carrier + `Inject`/`Extract` for segmentio/kafka-go headers.
   - `otel/otelgorm` — wraps uptrace `otelgorm` with two fixed policies: no bind-parameter values, and **no span without a parent span**.
   - `otel/otelfiber` — server root span per request, named by route template, status from the app's ErrorHandler.
3. Propagator is W3C TraceContext + Baggage.
4. Metric and log bridges only take effect with the OTLP/HTTP exporter (previously the metric exporter was built against `Endpoint` even for `stdout`/`noop`).

## Why

- The "one concern per sub-package" rule protects consumers from importing what they don't use. Here every adopter uses all three signals together and shuts them down together; splitting them would move the same ~40 lines of glue into each of the 15+ `main.go` files instead of removing it.
- The child-only policy in `otelgorm` is a fleet-wide cost/noise decision, not a per-service knob — consistent with how `health` fixes its probe paths and deadline.

## Considered and rejected

- **Extend `tracer/`.** No new package, but the name would lie about its contents.
- **Three packages (`tracer`, `otelmetric`, `otellog`).** Matches the old rule; every binary then constructs and shuts down three providers and each repo re-writes the glue.
- **`gofiber/contrib/otelfiber`.** Adds a dependency for ~60 lines; this middleware must also clone every string out of fasthttp's reusable buffers before it reaches an asynchronously exported span (aliasing has crashed a fleet service before), which we want to own and test.

## Consequences

- New module dependencies: `go.opentelemetry.io/contrib/bridges/prometheus`, OTLP metric/log exporters, `otel/sdk/log`, `otel/log`, `github.com/uptrace/opentelemetry-go-extra/otelgorm` (+ `otelsql`). OTel core bumped v1.43 → v1.44.
- Queries on `context.Background()` are not traced by design; callers must pass the request/message context with `db.WithContext(ctx)`.
- A request matching no Fiber handler is named after the last middleware route it passed (e.g. `GET /`) — Fiber does not expose whether a route is middleware.
- The log bridge stays logger-agnostic: adopters tee their own zap/slog logger with `otelzap` / `otelslog` using `Provider.LoggerProvider()`.
