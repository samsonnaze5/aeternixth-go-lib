# ADR-0001: Kafka Pinger uses Metadata RPC, not Dial-only

**Status:** accepted
**Date:** 2026-05-10

## Context

`observability/health/healthkafka` provides a `Pinger` for services that publish to or consume from Kafka. Two implementations were viable:

- **Dial-only.** `kafka.DialContext` against the first reachable broker — cheap, one RTT, no topic awareness.
- **Metadata RPC.** Full Metadata call against the cluster, asserting that every topic in a caller-supplied list is present — heavier, multiple RTTs, requires a topic list.

Five of six Go services in the fleet (`feeder`, `mt5-processor`, `client-portal-api`, `finance-service`, `archiver`) speak Kafka, so the choice cascades widely.

## Decision

`healthkafka.MetadataPinger` uses the Metadata RPC variant. No `DialPinger` ships in this library.

## Why

Dial-only silently keeps a service marked **Ready** when a topic has been accidentally deleted or mis-typed in configuration — the broker is reachable but the service cannot publish or consume. Metadata RPC catches this failure mode with the same call shape consumers typically run at startup, so runtime readiness mirrors startup gating.

## Considered and rejected

- **Dial-only.** Cheaper and avoids the broker-restart flap risk recorded in ADR-0002, but does not catch missing-topic failures and gives an inconsistent answer relative to most callers' startup checks.
- **Both flavors in lib.** Adds API surface and forces every adopting team to re-derive the trade-off on every probe wiring. A team with a genuine reason to want Dial-only can write a 3-line `Pinger` adapter in their own repo.

## Consequences

- Adopters of `healthkafka.MetadataPinger` are exposed to readiness flap during Kafka rolling broker restarts (typically 5–10 min for a 3-broker cluster). Operators MUST size `readinessProbe.failureThreshold` and `periodSeconds` accordingly — see ADR-0002 for recommended starting values.
- Lag metrics (`*_lag_seconds`) remain the primary alerting signal for Kafka health. Readiness is the deployment-blocking signal, not the paging signal — a long Kafka outage should page on-call via lag thresholds, not via readiness flap.
- The `onetrust-feeder` repo retains its own `internal/kafka.CheckTopicsExist` for startup gating; the two implementations may drift over time. If drift becomes painful, `CheckTopicsExist` should be promoted into this library so feeder uses one canonical implementation for both startup and probe.
