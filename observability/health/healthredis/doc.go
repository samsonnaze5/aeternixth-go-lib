// Package healthredis adapts a *redis.Client to the
// [github.com/samsonnaze5/aeternixth-go-lib/observability/health.Pinger]
// contract. Construct via [NewPinger]; the constructor rejects a nil
// client with [ErrNilClient].
package healthredis
