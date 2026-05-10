# ADR-0002: Kubernetes probe spec guidance for services using healthkafka

**Status:** accepted
**Date:** 2026-05-10

## Context

Per ADR-0001, `healthkafka.MetadataPinger` is the only Kafka readiness check this library ships. Kubernetes probe defaults — `failureThreshold: 3`, `periodSeconds: 10` (≈ 30 s tolerance) — are too aggressive for services using it: a Kafka rolling broker restart (5–10 min for a 3-broker cluster) will flap every pod's readiness simultaneously, blocking deployments and triggering churn on PodDisruptionBudgets and HPAs.

## Decision

The library does not configure Kubernetes probe specs — that is a deployment concern, not a library concern. This ADR records the operational guidance: deployment manifests for any service that wires `healthkafka.MetadataPinger` into its `/health/readyz` MUST configure `failureThreshold` and `periodSeconds` sized to the longest expected Kafka maintenance window, before the deployment ships.

## Recommended starting values

Tune per cluster. The values below assume a 3-broker cluster with rolling-restart maintenance windows of ~5 min; longer windows require a larger `failureThreshold`.

```yaml
livenessProbe:
  httpGet:
    path: /health/livez
    port: 9090
  periodSeconds: 30
  timeoutSeconds: 2
  failureThreshold: 3       # 90 s tolerance — only restarts on a truly stuck process

readinessProbe:
  httpGet:
    path: /health/readyz
    port: 9090
  periodSeconds: 15
  timeoutSeconds: 2         # > the handler's 800 ms internal deadline
  failureThreshold: 20      # 5 min tolerance — covers a 3-broker rolling restart
```

Services that do not use `healthkafka` (no Kafka **Pinger** wired in) can use Kubernetes defaults — the flap risk above does not apply.

## Why this is operational, not library-level

A library that ships hard-coded probe values bakes in cluster-specific assumptions (broker count, network conditions, deployment cadence). Operators own those constraints. The library's job is to make the trade-off legible and the recommended starting point reproducible — not to enforce one cluster's configuration on every adopter.

## Consequences

- New repos adopting `healthkafka.MetadataPinger` need operator review of probe spec before the first production deployment. CI in adopting repos should not gate on this — it is an organizational checkpoint, not a code-level one.
- Lag metrics remain the primary alerting signal for Kafka health (see ADR-0001). Readiness blocks deployments; metrics page on-call. The two are not redundant — they protect different concerns.
