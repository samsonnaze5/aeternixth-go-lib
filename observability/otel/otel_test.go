package otel_test

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	otelapi "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/samsonnaze5/aeternixth-go-lib/observability/otel"
)

func enabledConfig(exp otel.Exporter) otel.Config {
	return otel.Config{
		Enabled:         true,
		ServiceName:     "test-svc",
		ServiceVersion:  "v0.0.0",
		DeployEnv:       "test",
		Exporter:        exp,
		Endpoint:        "http://localhost:4318",
		Insecure:        true,
		SampleRate:      1.0,
		SpanQueueSize:   64,
		MetricsInterval: time.Hour,
	}
}

func goroutineCount() int {
	runtime.GC()
	return runtime.NumGoroutine()
}

func TestNew_Disabled_ReturnsNoOpProvider(t *testing.T) {
	before := goroutineCount()
	p, err := otel.New(context.Background(), otel.Config{Enabled: false, PrometheusBridge: true, LogsBridge: true})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if p.Resource() != nil {
		t.Errorf("Resource: want nil on disabled provider, got %v", p.Resource())
	}
	if p.TracerProvider() == nil || p.MeterProvider() == nil || p.LoggerProvider() == nil {
		t.Error("accessors: want non-nil no-op providers")
	}
	if err := p.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown: %v", err)
	}
	if after := goroutineCount(); after != before {
		t.Errorf("goroutines: before=%d after=%d", before, after)
	}
}

func TestNew_Errors(t *testing.T) {
	tests := []struct {
		name string
		cfg  otel.Config
		want error
	}{
		{
			name: "unsupported exporter",
			cfg:  enabledConfig("pulsar"),
			want: otel.ErrUnsupportedExporter,
		},
		{
			name: "prometheus bridge without gatherer",
			cfg: func() otel.Config {
				c := enabledConfig(otel.ExporterOTLPHTTP)
				c.PrometheusBridge = true
				return c
			}(),
			want: otel.ErrMissingGatherer,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := otel.New(context.Background(), tt.cfg)
			if !errors.Is(err, tt.want) {
				t.Errorf("err: want %v, got %v", tt.want, err)
			}
		})
	}
}

func TestNew_Enabled_ShutdownFlushesAndIsIdempotent(t *testing.T) {
	tests := []struct {
		name string
		cfg  otel.Config
	}{
		{name: "noop exporter", cfg: enabledConfig(otel.ExporterNoop)},
		{name: "stdout exporter", cfg: enabledConfig(otel.ExporterStdout)},
		{name: "otlp with both bridges", cfg: func() otel.Config {
			c := enabledConfig(otel.ExporterOTLPHTTP)
			c.PrometheusBridge = true
			c.LogsBridge = true
			return c
		}()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := otel.New(context.Background(), tt.cfg,
				otel.WithPrometheusGatherer(prometheus.NewRegistry()))
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			if p.Resource() == nil {
				t.Error("Resource: want non-nil on enabled provider")
			}
			// Unreachable collector: Shutdown may report an export error,
			// but must return within the deadline and be idempotent.
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = p.Shutdown(ctx)
			if err := p.Shutdown(ctx); err != nil {
				t.Errorf("second Shutdown: want nil, got %v", err)
			}
		})
	}
}

func TestNew_BridgesIgnoredWithoutOTLP(t *testing.T) {
	// Bridges only make sense with an OTLP collector; with stdout/noop the
	// gatherer is not required and the meter/logger stay no-op.
	cfg := enabledConfig(otel.ExporterNoop)
	cfg.PrometheusBridge = true
	cfg.LogsBridge = true
	p, err := otel.New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer p.Shutdown(context.Background())
	if _, ok := p.LoggerProvider().(interface{ Shutdown(context.Context) error }); ok {
		t.Error("LoggerProvider: want no-op without OTLP exporter")
	}
}

func TestNew_WithSpanExporter_EmitsSpansSync(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	p, err := otel.New(context.Background(), enabledConfig(otel.ExporterOTLPHTTP), otel.WithSpanExporter(exp))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer p.Shutdown(context.Background())

	_, span := p.Tracer("unit").Start(context.Background(), "op")
	span.End()

	if got := exp.GetSpans(); len(got) != 1 || got[0].Name != "op" {
		t.Fatalf("spans: want [op], got %v", got)
	}
}

func TestNew_InstallsTraceContextAndBaggagePropagator(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	p, err := otel.New(context.Background(), enabledConfig(otel.ExporterNoop), otel.WithSpanExporter(exp))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer p.Shutdown(context.Background())

	member, _ := baggage.NewMember("tenant", "a")
	bag, _ := baggage.New(member)
	ctx, span := p.Tracer("unit").Start(baggage.ContextWithBaggage(context.Background(), bag), "op")
	defer span.End()

	carrier := propagation.MapCarrier{}
	otelapi.GetTextMapPropagator().Inject(ctx, carrier)
	if carrier.Get("traceparent") == "" {
		t.Error("traceparent: want injected")
	}
	if carrier.Get("baggage") != "tenant=a" {
		t.Errorf("baggage: want %q, got %q", "tenant=a", carrier.Get("baggage"))
	}
}
