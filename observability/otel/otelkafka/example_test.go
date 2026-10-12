package otelkafka_test

import (
	"context"

	kafkago "github.com/segmentio/kafka-go"

	"github.com/samsonnaze5/aeternixth-go-lib/observability/otel/otelkafka"
)

func Example() {
	// Producer side: stamp the current trace onto the outgoing message.
	ctx := context.Background()
	msg := kafkago.Message{Topic: "orders", Value: []byte("{}")}
	msg.Headers = otelkafka.Inject(ctx, msg.Headers)

	// Consumer side: continue the producer's trace.
	consumeCtx := otelkafka.Extract(context.Background(), msg.Headers)
	_ = consumeCtx // tracer.Start(consumeCtx, "orders.consume", ...)
}
