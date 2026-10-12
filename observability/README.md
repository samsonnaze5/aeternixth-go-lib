# observability/

Shared **observability primitives** for services that consume `aeternixth-go-lib`. This directory is the umbrella for every cross-cutting concern that touches "what's the system doing right now" — Kubernetes probes today, more siblings as repeated patterns surface across the fleet.

## Sub-packages

### Available

| Path | Purpose |
|---|---|
| [`health/`](./health/) | Kubernetes-style liveness and readiness probes — `Pinger` interface, parallel-execution `/health/readyz` aggregator, `/health/livez` handler, convenience HTTP server. |
| [`health/healthfiber/`](./health/healthfiber/) | Fiber router adapter for the same probes (mounts on existing Fiber apps without a sidecar port). |
| [`health/healthredis/`](./health/healthredis/) | Pinger adapter for `*redis.Client`. |
| [`health/healthgorm/`](./health/healthgorm/) | Pinger adapter for `*gorm.DB`. |
| [`health/healthkafka/`](./health/healthkafka/) | Metadata-RPC Pinger adapter for Kafka brokers + topics ([ADR-0001](../docs/adr/0001-kafka-pinger-uses-metadata-rpc.md)). |
| [`otel/`](./otel/) | OpenTelemetry SDK lifecycle — one `Provider` for traces, the Prometheus → OTLP metric bridge, and the OTLP log pipeline ([ADR-0004](../docs/adr/0004-otel-provider-bundles-three-signals.md)). |
| [`otel/otelkafka/`](./otel/otelkafka/) | W3C trace-context carrier for segmentio/kafka-go headers. |
| [`otel/otelgorm/`](./otel/otelgorm/) | GORM query spans — child-only (no span without a parent), no bind values. |
| [`otel/otelfiber/`](./otel/otelfiber/) | Fiber middleware opening the server root span per request. |
| [`logger/`](./logger/) | `slog` JSON logger setup. |
| [`tracer/`](./tracer/) | **Deprecated** — superseded by `otel/`. |

### Planned

| Path | Purpose | Trigger |
|---|---|---|
| `metrics/` | `prometheus.Registry` helper — collector registration conventions, default labels (`service`, `version`), build-info gauge. | When two adopters need the same registry shape. |
| `middleware/` | HTTP middleware that wires `tracer` + `logger` + `metrics` automatically (request span, structured access log, request-duration histogram). | Once `tracer`, `logger`, `metrics` exist and an adopter wants the bundled package. |

The planned set is forward-looking — each sub-package lands only when extraction passes the **deletion test**: would removing the proposal force every adopter to write the same code? If yes, extract. If no, leave it in the consuming repos.

## Conventions for new sub-packages

When adding a sibling under `observability/`:

1. **One concern per sub-package.** Exception: `otel/` bundles the three OTel signals because every adopter constructs and shuts them down together ([ADR-0004](../docs/adr/0004-otel-provider-bundles-three-signals.md)).
2. **Adapters live in nested sub-packages with a prefix.** Pattern: `health/healthredis/`, `health/healthgorm/`. The prefix avoids import-name collisions with the underlying client (`healthredis` next to `redis`, `healthkafka` next to `kafka`).
3. **Constructors `New{Type}(...) (*T, error)`** with sentinel errors named `Err{Description}`, per the project [constitution](../.specify/memory/constitution.md) Principle III.
4. **Fixed paths and budgets are values, not knobs.** `health` package fixes the probe paths and the 800 ms internal deadline — exposing them as configuration would defeat the consistency that makes a shared library worthwhile. Apply the same discipline here.
5. **Internal helpers go under `<package>/internal/`.** When two sub-packages need to share machinery (e.g., `health` and `healthfiber` share readiness aggregation), put the shared code in `health/internal/core/`. Don't widen the public API just to reach across siblings.
6. **Document the policy in `docs/adr/`** when extraction involves a real trade-off (alternative considered, alternative rejected with reason). See [ADR-0001](../docs/adr/0001-kafka-pinger-uses-metadata-rpc.md) for the canonical example.
7. **Update [CONTEXT.md](../CONTEXT.md)** with the new package's domain language under its own subsection.

Children of this directory MUST be self-contained: any sub-package can be imported in isolation without dragging the entire `observability/` tree into a downstream module's dependency graph. Cross-sub-package imports are forbidden by convention except where a shared `internal/` makes the relationship explicit.
