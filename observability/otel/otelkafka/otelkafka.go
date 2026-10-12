// Package otelkafka carries W3C trace context through segmentio/kafka-go
// message headers, so a consumer's spans join the producer's trace.
//
// Both helpers use the global propagator installed by otel.New.
package otelkafka

import (
	"context"
	"strings"

	kafkago "github.com/segmentio/kafka-go"
	otelapi "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// HeaderCarrier adapts []kafkago.Header to propagation.TextMapCarrier.
// Keys match case-insensitively. Use a pointer: Set may append.
type HeaderCarrier []kafkago.Header

var _ propagation.TextMapCarrier = (*HeaderCarrier)(nil)

// Get returns the value of the first header matching key, or "".
func (c *HeaderCarrier) Get(key string) string {
	for _, h := range *c {
		if strings.EqualFold(h.Key, key) {
			return string(h.Value)
		}
	}
	return ""
}

// Set replaces the value of an existing header matching key, or appends
// a new one.
func (c *HeaderCarrier) Set(key, value string) {
	for i, h := range *c {
		if strings.EqualFold(h.Key, key) {
			(*c)[i].Value = []byte(value)
			return
		}
	}
	*c = append(*c, kafkago.Header{Key: key, Value: []byte(value)})
}

// Keys returns every header key.
func (c *HeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(*c))
	for _, h := range *c {
		keys = append(keys, h.Key)
	}
	return keys
}

// Inject writes the trace context of ctx into headers and returns the
// result. Existing trace headers are overwritten in place, so pass a
// slice the caller owns.
func Inject(ctx context.Context, headers []kafkago.Header) []kafkago.Header {
	carrier := HeaderCarrier(headers)
	otelapi.GetTextMapPropagator().Inject(ctx, &carrier)
	return carrier
}

// Extract returns ctx enriched with the remote trace context carried in
// headers. Start the consumer span from the returned context.
func Extract(ctx context.Context, headers []kafkago.Header) context.Context {
	carrier := HeaderCarrier(headers)
	return otelapi.GetTextMapPropagator().Extract(ctx, &carrier)
}
