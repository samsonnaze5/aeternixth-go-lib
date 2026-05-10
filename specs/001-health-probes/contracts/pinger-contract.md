# Pinger Contract

**Phase**: 1 (Design & Contracts)
**Audience**: developers writing custom pingers OR maintainers extending lib adapters

The `Pinger` interface is the seam between the library's parallel-execution machinery and any individual dependency check. Every constraint listed here is a contract that adapter authors and lib maintainers must respect.

## Interface

```go
type Pinger interface {
    Ping(ctx context.Context) error
}
```

## Invariants — what `Ping` MUST guarantee

### I-1. Context respect

`Ping` MUST treat `ctx` as authoritative. It MUST return promptly when `ctx` is cancelled or its deadline elapses. The library hands every pinger a context with at most 800 ms remaining; a pinger that ignores `ctx` and runs longer is a bug.

Concretely: every blocking call inside `Ping` (network I/O, lock acquisition, etc.) MUST take a `ctx` parameter or be wrapped with `context.AfterFunc`-style cancellation. Sleeping with `time.Sleep(d)` instead of `select { case <-time.After(d): case <-ctx.Done(): }` is incorrect.

### I-2. No panics

`Ping` MUST NOT panic on internal errors (network failure, dep unreachable, malformed input). It MUST return a `non-nil error` describing the failure.

`Ping` MAY panic on programmer errors (nil receiver, internal contract violation), but adapters in this library MUST construct pingers via `NewPinger`-style validators that catch nil-input and return `ErrNilClient` instead of letting the receiver be nil.

### I-3. Error fidelity

When `Ping` returns a non-nil error, the error message MUST be useful for an operator reading the `/readyz` JSON response. "topic missing: feeder.deals.v1" is good; "error" is not. Library adapters wrap underlying errors with a package-prefixed sentinel (`healthkafka: ...`) so the source of the failure is identifiable from the message alone.

### I-4. Idempotent on repeat invocation

`Ping` MAY be called every probe interval (typically 15 s in production per ADR-0002). It MUST NOT mutate observable dependency state or accumulate per-call resources (e.g., open connections that aren't closed, growing in-memory caches). Stateless verification is the model.

### I-5. Concurrency safety

A single `Pinger` instance MAY be invoked concurrently from multiple goroutines (the library does this for the parallel readiness aggregator, and operator scripts may also call `/readyz` simultaneously from multiple sources). `Ping` MUST be safe for concurrent use, or its godoc MUST explicitly state otherwise (which would block adoption — the library's machinery assumes safety).

For wrapped clients that are themselves concurrency-safe (`*pgxpool.Pool`, `*redis.Client`, GORM `*gorm.DB`), the wrapping `Pinger` is automatically safe. Custom adapters MUST verify the underlying client's contract.

## Library-side guarantees — what callers can rely on

### G-1. Parallel execution under deadline

The library invokes every configured pinger in parallel via `errgroup.WithContext`. Each pinger receives a context derived from a `context.WithTimeout(parent, 800ms)`. The aggregator waits for all pingers to return OR the deadline to elapse, whichever comes first.

### G-2. Sibling isolation on failure

A pinger returning a non-nil error does **not** cancel sibling pingers. Every configured check is reported in the response, regardless of how many earlier checks failed. (The aggregator's goroutines return `nil` to `errgroup.Go` so the error path doesn't propagate.)

### G-3. Deadline-elapsed reporting

When the 800 ms deadline elapses with one or more pingers still running, those pingers are reported `ok: false` with a context-deadline-exceeded error. The aggregator does not wait for them to finish.

### G-4. No retries

The library does NOT retry pingers within a single `/readyz` call. If a pinger returns an error, the failure is reported and the operator's K8s probe spec (per ADR-0002) decides whether the failure is transient via `failureThreshold`.

### G-5. No caching

The library does NOT cache pinger results across calls. Every `/readyz` call invokes every pinger fresh.

## Custom pinger checklist

For services that need a pinger the lib doesn't ship (e.g., a custom HTTP API health check):

```go
type MyAPI struct{ Client *http.Client; URL string }

func (m *MyAPI) Ping(ctx context.Context) error {
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.URL, nil)
    if err != nil {
        return fmt.Errorf("myapi: build request: %w", err)
    }
    resp, err := m.Client.Do(req)
    if err != nil {
        return fmt.Errorf("myapi: request: %w", err)
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("myapi: unexpected status %d", resp.StatusCode)
    }
    return nil
}
```

Verify against:

- [ ] I-1: every blocking call accepts `ctx` (the request, not just `http.Get`)
- [ ] I-2: no panics on nil `Client` (handled by the request builder returning err)
- [ ] I-3: error messages name the source ("myapi: ...")
- [ ] I-4: stateless — no per-call accumulation
- [ ] I-5: `*http.Client` is safe for concurrent use → `MyAPI.Ping` is safe
