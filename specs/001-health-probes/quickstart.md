# Quickstart: Health Probe Library

**Phase**: 1 (Design & Contracts)
**Audience**: developers wiring a service in the fleet to use `observability/health`

Three flavours covering every adopting repo. Pick the one that matches your service shape.

## Worker binary (no inbound HTTP) — `feeder`, `mt5-processor`, worker subcommands of `finance-service` / `archiver`

```go
package main

import (
    "context"
    "errors"
    "log/slog"
    "net/http"
    "os"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/prometheus/client_golang/prometheus"

    "github.com/samsonnaze5/aeternixth-go-lib/observability/health"
    "github.com/samsonnaze5/aeternixth-go-lib/observability/health/healthkafka"
)

func main() {
    ctx := context.Background()

    statePool, _ := pgxpool.New(ctx, "...")
    defer statePool.Close()

    sourcePool, _ := pgxpool.New(ctx, "...")
    defer sourcePool.Close()

    kafkaPinger, err := healthkafka.NewMetadataPinger(
        []string{"broker-1:9092", "broker-2:9092"},
        []string{"feeder.deals.v1", "feeder.deals.dlq.v1"},
        nil, // tls
    )
    if err != nil {
        slog.Error("healthkafka", "err", err.Error())
        os.Exit(1)
    }

    registry := prometheus.NewRegistry()
    // ... register your metrics on registry ...

    srv := health.NewServer(":9090", map[string]health.Pinger{
        "state_pool":  statePool,
        "source_pool": sourcePool,
        "kafka":       kafkaPinger,
    }, registry)

    if srv != nil {
        go func() {
            if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
                slog.Warn("probe server exited", "err", err.Error())
            }
        }()
        defer srv.Shutdown(context.Background())
    }

    // ... rest of binary lifecycle ...
}
```

**Notes**
- `*pgxpool.Pool` already satisfies `health.Pinger` — no adapter needed.
- Pass `addr=""` to skip the probe server entirely (e.g., for one-shot CLI binaries that don't run long enough to be probed).
- Pass `registry=nil` to skip `/metrics` (e.g., for binaries that expose metrics elsewhere).

## Fiber API (single-port app) — `mt5-proxy-api`, `client-portal-api`, Fiber subcommands of `finance-service` / `archiver`

```go
package main

import (
    "github.com/ansrivas/fiberprometheus/v2"
    "github.com/gofiber/fiber/v2"
    "gorm.io/gorm"

    "github.com/samsonnaze5/aeternixth-go-lib/observability/health"
    "github.com/samsonnaze5/aeternixth-go-lib/observability/health/healthfiber"
    "github.com/samsonnaze5/aeternixth-go-lib/observability/health/healthgorm"
)

func main() {
    app := fiber.New()

    // existing fiberprometheus mount stays untouched
    fp := fiberprometheus.New("api")
    fp.RegisterAt(app, "/metrics")
    app.Use(fp.Middleware)

    db := openGorm() // your existing helper
    pg, err := healthgorm.NewPinger(db)
    if err != nil {
        panic(err) // bail on misconfig
    }

    healthfiber.Mount(app, map[string]health.Pinger{
        "postgres": pg,
    })

    // ... business routes ...

    app.Listen(":8080")
}
```

**Notes**
- `healthfiber.Mount` registers `GET /health/livez` and `GET /health/readyz` on the supplied router. It does **not** touch `/metrics` — `fiberprometheus` keeps owning that path.
- Same library, same JSON shape as the worker case. K8s manifests (`livenessProbe`, `readinessProbe`) point at the Fiber app's port (e.g., `8080`), not a sidecar.

## Mixed repo — `finance-service`, `archiver`

These repos have a Fiber app (`cmd/server`) AND worker subcommands (`cmd/consumer-*`). Each binary picks the matching pattern above. The same `healthgorm.NewPinger(db)` constructor wraps the same `*gorm.DB` — pingers are reusable across binary shapes within a repo.

## Adapter cookbook

```go
// Redis
import "github.com/samsonnaze5/aeternixth-go-lib/observability/health/healthredis"
rd, err := healthredis.NewPinger(redisClient)

// GORM
import "github.com/samsonnaze5/aeternixth-go-lib/observability/health/healthgorm"
pg, err := healthgorm.NewPinger(gormDB)

// Kafka (Metadata RPC — verifies broker reachability AND topic existence)
import "github.com/samsonnaze5/aeternixth-go-lib/observability/health/healthkafka"
kf, err := healthkafka.NewMetadataPinger(brokers, topics, tlsConfig)

// pgx (no adapter needed — pool implements Pinger natively)
import "github.com/jackc/pgx/v5/pgxpool"
pool, _ := pgxpool.New(ctx, dsn)
// pass `pool` directly into the map[string]health.Pinger

// ClickHouse (no adapter needed — *clickhouse.Conn implements Pinger natively)
import "github.com/ClickHouse/clickhouse-go/v2"
conn, _ := clickhouse.Open(...)
// pass `conn` directly
```

## Kubernetes probe spec — recommended starting values

Per [ADR-0002](../../docs/adr/0002-kubernetes-probe-spec-guidance.md). Tune per cluster.

```yaml
livenessProbe:
  httpGet: { path: /health/livez, port: 9090 }
  periodSeconds: 30
  timeoutSeconds: 2
  failureThreshold: 3       # 90 s tolerance

readinessProbe:
  httpGet: { path: /health/readyz, port: 9090 }
  periodSeconds: 15
  timeoutSeconds: 2         # > the lib's 800 ms internal deadline
  failureThreshold: 20      # 5 min tolerance — covers a 3-broker rolling restart
```

For Fiber repos (single port), the port is the Fiber app's port (commonly `8080`). For workers, the port is whatever you passed to `health.NewServer` (commonly `9090`).

## Checklist before shipping

- [ ] Constructor errors from `healthkafka`/`healthgorm`/`healthredis` are handled at startup (binary refuses to start if misconfigured)
- [ ] `srv.Shutdown(ctx)` deferred ahead of the dependency closes (so `/readyz` naturally reports failure as deps tear down — see ADR-0001 in `onetrust-feeder` for ordering rationale)
- [ ] K8s deployment manifest `failureThreshold` and `periodSeconds` sized per ADR-0002 (especially for services using Kafka)
- [ ] Caller-chosen check keys (`state_pool`, `kafka`, etc.) are coordinated with the operator's runbook
