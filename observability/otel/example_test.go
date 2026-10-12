package otel_test

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/samsonnaze5/aeternixth-go-lib/observability/otel"
)

func ExampleNew() {
	registry := prometheus.NewRegistry()

	p, err := otel.New(context.Background(), otel.Config{
		Enabled:          false, // true in deployed environments
		ServiceName:      "my-service",
		ServiceVersion:   "v1.2.3",
		DeployEnv:        "staging",
		Exporter:         otel.ExporterOTLPHTTP,
		Endpoint:         "http://otel-collector:4318",
		SampleRate:       0.1,
		SpanQueueSize:    2048,
		PrometheusBridge: true,
		MetricsInterval:  30 * time.Second,
		LogsBridge:       true,
	}, otel.WithPrometheusGatherer(registry))
	if err != nil {
		panic(err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = p.Shutdown(ctx)
	}()

	_, span := p.Tracer("my-service/handler").Start(context.Background(), "op")
	span.End()
	fmt.Println(span.SpanContext().IsValid())
	// Output: false
}
