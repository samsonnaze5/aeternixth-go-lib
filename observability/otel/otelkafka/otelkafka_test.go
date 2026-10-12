package otelkafka_test

import (
	"context"
	"testing"

	kafkago "github.com/segmentio/kafka-go"
	otelapi "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"github.com/samsonnaze5/aeternixth-go-lib/observability/otel/otelkafka"
)

func TestHeaderCarrier(t *testing.T) {
	c := otelkafka.HeaderCarrier{{Key: "Traceparent", Value: []byte("old")}}

	tests := []struct {
		name string
		do   func()
		key  string
		want string
	}{
		{name: "get is case-insensitive", do: func() {}, key: "traceparent", want: "old"},
		{name: "get missing key", do: func() {}, key: "baggage", want: ""},
		{name: "set replaces existing", do: func() { c.Set("traceparent", "new") }, key: "TRACEPARENT", want: "new"},
		{name: "set appends new", do: func() { c.Set("baggage", "k=v") }, key: "baggage", want: "k=v"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.do()
			if got := c.Get(tt.key); got != tt.want {
				t.Errorf("Get(%q): want %q, got %q", tt.key, tt.want, got)
			}
		})
	}
	if keys := c.Keys(); len(keys) != 2 {
		t.Errorf("Keys: want 2 (replace must not duplicate), got %v", keys)
	}
}

func TestInjectExtract_RoundTrip(t *testing.T) {
	otelapi.SetTextMapPropagator(propagation.TraceContext{})
	tp := sdktrace.NewTracerProvider()
	defer tp.Shutdown(context.Background())

	ctx, span := tp.Tracer("test").Start(context.Background(), "produce")
	defer span.End()

	headers := otelkafka.Inject(ctx, []kafkago.Header{{Key: "app", Value: []byte("x")}})
	if len(headers) != 2 {
		t.Fatalf("headers: want app + traceparent, got %v", headers)
	}

	got := trace.SpanContextFromContext(otelkafka.Extract(context.Background(), headers))
	if !got.IsRemote() || got.TraceID() != span.SpanContext().TraceID() {
		t.Errorf("extracted: want remote span of trace %s, got %+v", span.SpanContext().TraceID(), got)
	}
}

func TestExtract_NoHeaders_ReturnsInvalidSpanContext(t *testing.T) {
	otelapi.SetTextMapPropagator(propagation.TraceContext{})
	ctx := otelkafka.Extract(context.Background(), nil)
	if trace.SpanContextFromContext(ctx).IsValid() {
		t.Error("want no span context when headers carry none")
	}
}
