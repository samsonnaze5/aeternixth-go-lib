package tracer_test

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/samsonnaze5/aeternixth-go-lib/observability/tracer"
)

func TestNew_Disabled_ReturnsNoOpProvider(t *testing.T) {
	p, err := tracer.New(context.Background(), tracer.Config{Enabled: false})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if p == nil {
		t.Fatal("Provider: want non-nil even when disabled")
	}
	// Disabled path — no resource, but accessors are still safe to call.
	if p.Resource() != nil {
		t.Errorf("Resource on disabled provider: want nil, got %v", p.Resource())
	}
	tr := p.Tracer("test")
	if tr == nil {
		t.Errorf("Tracer on disabled provider: want non-nil no-op, got nil")
	}
	if err := p.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown on disabled provider: want nil, got %v", err)
	}
}

func TestNew_UnsupportedExporter_ReturnsErr(t *testing.T) {
	_, err := tracer.New(context.Background(), tracer.Config{
		Enabled:     true,
		ServiceName: "test",
		Exporter:    "this-does-not-exist",
		SampleRate:  1.0,
	})
	if !errors.Is(err, tracer.ErrUnsupportedExporter) {
		t.Errorf("err: want ErrUnsupportedExporter, got %v", err)
	}
}

func TestNew_WithSpanExporter_EmitsSpansSync(t *testing.T) {
	// In-memory test seam — spans land synchronously when WithSpanExporter
	// is supplied. Verifies the test-seam contract documented in
	// tracer.WithSpanExporter godoc.
	exp := tracetest.NewInMemoryExporter()
	p, err := tracer.New(context.Background(),
		tracer.Config{
			Enabled:        true,
			ServiceName:    "test-svc",
			ServiceVersion: "v0.0.0",
			DeployEnv:      "test",
			Exporter:       tracer.ExporterOTLPHTTP, // ignored — overridden by option
			Endpoint:       "ignored",
			SampleRate:     1.0,
			SpanQueueSize:  1024,
		},
		tracer.WithSpanExporter(exp),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer p.Shutdown(context.Background())

	_, span := p.Tracer("unit").Start(context.Background(), "op")
	span.End()

	got := exp.GetSpans()
	if len(got) != 1 {
		t.Fatalf("spans: want 1, got %d", len(got))
	}
	if got[0].Name != "op" {
		t.Errorf("span name: want %q, got %q", "op", got[0].Name)
	}
}

func TestProvider_Shutdown_Idempotent(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	p, err := tracer.New(context.Background(),
		tracer.Config{
			Enabled:    true,
			Exporter:   tracer.ExporterNoop,
			SampleRate: 1.0,
		},
		tracer.WithSpanExporter(exp),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := p.Shutdown(context.Background()); err != nil {
		t.Errorf("first Shutdown: %v", err)
	}
	if err := p.Shutdown(context.Background()); err != nil {
		t.Errorf("second Shutdown (idempotent): %v", err)
	}
}

func TestNew_ServiceNameDetection(t *testing.T) {
	// When ServiceName is empty, buildResource calls detectServiceName.
	// Verify Provider construction succeeds end-to-end and Resource
	// carries a non-empty service.name attribute.
	exp := tracetest.NewInMemoryExporter()
	p, err := tracer.New(context.Background(),
		tracer.Config{
			Enabled:    true,
			Exporter:   tracer.ExporterNoop,
			SampleRate: 1.0,
		},
		tracer.WithSpanExporter(exp),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer p.Shutdown(context.Background())

	res := p.Resource()
	if res == nil {
		t.Fatal("Resource: want non-nil for enabled provider")
	}
	for _, kv := range res.Attributes() {
		if string(kv.Key) == "service.name" && kv.Value.AsString() == "" {
			t.Error("service.name: want non-empty (detected from binary), got empty")
		}
	}
}
