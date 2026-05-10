package healthkafka

import (
	"context"
	"crypto/tls"
	"errors"
)

// Sentinel errors. All are errors.Is-comparable.
var (
	// ErrEmptyBrokers is returned by NewMetadataPinger when the brokers
	// slice is nil or zero-length.
	ErrEmptyBrokers = errors.New("healthkafka: brokers list is empty")

	// ErrEmptyTopics is returned by NewMetadataPinger when the topics
	// slice is nil or zero-length.
	ErrEmptyTopics = errors.New("healthkafka: topics list is empty")

	// ErrTopicMissing wraps an inner error when a topic is requested
	// but absent from the cluster's metadata response. The offending
	// topic name is appended after the wrapped error: e.g.
	// "healthkafka: topic missing: feeder.deals.v1".
	ErrTopicMissing = errors.New("healthkafka: topic missing")
)

// MetadataPinger satisfies the health.Pinger contract by performing a
// Kafka Metadata RPC and asserting that every configured topic is
// present. Construct via NewMetadataPinger.
//
// Per ADR-0001 in this repository, only the Metadata RPC variant ships
// — Dial-only is rejected because it cannot detect a topic deleted out
// from under the service.
type MetadataPinger struct {
	brokers []string
	topics  []string
	tls     *tls.Config
}

// NewMetadataPinger validates the supplied configuration and returns a
// pinger ready to register in a readiness map. brokers and topics MUST
// each be non-empty; tls may be nil to disable TLS.
//
// Validation is done at construction so a misconfigured binary refuses
// to start instead of reporting a permanent 503 from /health/readyz.
func NewMetadataPinger(brokers, topics []string, tlsConfig *tls.Config) (*MetadataPinger, error) {
	if len(brokers) == 0 {
		return nil, ErrEmptyBrokers
	}
	if len(topics) == 0 {
		return nil, ErrEmptyTopics
	}
	return &MetadataPinger{
		brokers: brokers,
		topics:  topics,
		tls:     tlsConfig,
	}, nil
}

// Ping dials the first reachable broker, runs a Metadata RPC, and
// asserts that every configured topic is present. See checkTopicsExist
// for the implementation details.
func (p *MetadataPinger) Ping(ctx context.Context) error {
	return checkTopicsExist(ctx, p.brokers, p.topics, p.tls)
}
