package healthkafka

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

// dialTimeout is a soft cap on the bootstrap dial. It is intentionally
// shorter than health's 800 ms internal probe deadline so the dial can
// fail fast and leave time for a metadata RPC if dial succeeds.
const dialTimeout = 400 * time.Millisecond

// checkTopicsExist dials the first reachable broker, fetches partition
// metadata for the supplied topics, and verifies that every requested
// topic is present in the cluster. Missing topics produce an error
// wrapping ErrTopicMissing with the offending topic name; other errors
// (dial, RPC) are wrapped with descriptive prefixes.
//
// This implementation is owned by the lib (per ADR-0001 consequence
// section). Onetrust-feeder retains its own internal/kafka.CheckTopicsExist
// for startup gating; the two may drift over time.
func checkTopicsExist(ctx context.Context, brokers, topics []string, tlsConfig *tls.Config) error {
	dialer := &kafka.Dialer{
		Timeout:   dialTimeout,
		DualStack: true,
		TLS:       tlsConfig,
	}

	conn, err := dialer.DialContext(ctx, "tcp", brokers[0])
	if err != nil {
		return fmt.Errorf("healthkafka: dial %s: %w", brokers[0], err)
	}
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			return fmt.Errorf("healthkafka: set deadline: %w", err)
		}
	}

	parts, err := conn.ReadPartitions(topics...)
	if err != nil {
		return fmt.Errorf("healthkafka: metadata RPC: %w", err)
	}

	present := make(map[string]struct{}, len(parts))
	for _, p := range parts {
		present[p.Topic] = struct{}{}
	}
	for _, t := range topics {
		if _, ok := present[t]; !ok {
			return fmt.Errorf("%w: %s", ErrTopicMissing, t)
		}
	}
	return nil
}
