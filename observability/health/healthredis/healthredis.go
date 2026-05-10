package healthredis

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// ErrNilClient is returned by NewPinger when the supplied *redis.Client
// is nil. It is errors.Is-comparable.
var ErrNilClient = errors.New("healthredis: nil *redis.Client")

// Pinger adapts a *redis.Client to the
// health.Pinger contract. Construct via NewPinger.
type Pinger struct {
	client *redis.Client
}

// NewPinger validates the supplied client and returns a *Pinger ready to
// register in a readiness map. A nil client is rejected with
// ErrNilClient — fail-fast at construction so misconfiguration surfaces
// in startup logs rather than as a permanent /readyz failure.
func NewPinger(client *redis.Client) (*Pinger, error) {
	if client == nil {
		return nil, ErrNilClient
	}
	return &Pinger{client: client}, nil
}

// Ping reaches the Redis broker and returns an error wrapped with the
// "healthredis:" prefix on failure. The underlying client's PING command
// is used; ctx cancellation propagates to the dial / read deadline.
func (p *Pinger) Ping(ctx context.Context) error {
	if err := p.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("healthredis: %w", err)
	}
	return nil
}
