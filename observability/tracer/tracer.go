package tracer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	otelapi "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

// Exporter selects the OTel trace exporter.
type Exporter string

const (
	// ExporterNoop disables span export. Spans are still created with
	// trace IDs (so logs can correlate) but never leave the process.
	ExporterNoop Exporter = "noop"

	// ExporterStdout writes spans to os.Stdout — convenient for local
	// development.
	ExporterStdout Exporter = "stdout"

	// ExporterOTLPHTTP sends spans via OTLP/HTTP transport — the
	// production target for an OTel collector or vendor backend.
	ExporterOTLPHTTP Exporter = "otlp_http"
)

// OTel resource attribute keys (OpenTelemetry Resource Semantic Conventions).
const (
	resourceServiceName    = "service.name"
	resourceServiceVersion = "service.version"
	resourceDeployEnv      = "deployment.environment"
	resourceHostName       = "host.name"
)

// ErrUnsupportedExporter is returned by New when Config.Exporter is not
// one of the recognised values. Wrapped with the offending value via
// fmt.Errorf("%w: %q", ErrUnsupportedExporter, cfg.Exporter).
var ErrUnsupportedExporter = errors.New("tracer: unsupported exporter")

// Config drives Provider construction. Validate inputs at the caller's
// configuration layer before invoking New — this package does not
// perform schema-level validation beyond what the OTel SDK enforces.
type Config struct {
	// Enabled — when false, New returns a no-op Provider that never
	// opens connections or starts goroutines.
	Enabled bool

	// ServiceName is recorded as resource attribute service.name. When
	// empty, falls back to the binary's filename (path-stripped).
	ServiceName string

	// ServiceVersion is recorded as service.version. Typically supplied
	// at build time via -ldflags "-X main.Version=...".
	ServiceVersion string

	// DeployEnv is recorded as deployment.environment (e.g. "prod",
	// "staging", "dev").
	DeployEnv string

	// Exporter selects the trace export transport. See the Exporter*
	// constants.
	Exporter Exporter

	// Endpoint is the OTLP collector URL — used only when Exporter ==
	// ExporterOTLPHTTP. Ignored otherwise.
	Endpoint string

	// Insecure disables TLS for OTLP transport. Has no effect on stdout
	// or noop.
	Insecure bool

	// SampleRate is the TraceIDRatioBased sampler ratio in [0.0, 1.0].
	// Wrapped in ParentBased so child spans inherit their parent's
	// decision.
	SampleRate float64

	// SpanQueueSize bounds the BatchSpanProcessor's in-memory queue.
	SpanQueueSize int
}

// Provider owns the OpenTelemetry SDK lifecycle for one binary. A
// Provider is constructed once at startup via New and shut down via a
// deferred Provider.Shutdown on process exit.
//
// On the disabled path (Config.Enabled == false) every accessor returns
// the OTel SDK's no-op equivalent — Tracer().Start() returns
// non-recording spans with zero allocation; Shutdown is a nil-safe
// no-op. Callers can therefore pass *Provider unconditionally.
type Provider struct {
	tp       trace.TracerProvider
	mp       metric.MeterProvider
	res      *resource.Resource // nil on disabled path
	shutdown func(context.Context) error
}

// Option customises New construction.
type Option func(*opts)

type opts struct {
	spanExporterOverride sdktrace.SpanExporter
}

// WithSpanExporter overrides the trace exporter that New would
// otherwise build from cfg.Exporter. Intended for integration tests
// that inject tracetest.NewInMemoryExporter to inspect emitted spans.
//
// When the override is non-nil, the SDK uses SimpleSpanProcessor
// (synchronous) so tests see every span the moment span.End is called
// — no batching, no timer, no flush race.
func WithSpanExporter(exp sdktrace.SpanExporter) Option {
	return func(o *opts) {
		o.spanExporterOverride = exp
	}
}

// New builds a Provider from validated Config. The returned error is
// non-nil only for misconfigurations the caller's validation pass
// missed (unsupported exporter, OTLP exporter constructor failure).
//
// All construction succeeds before this function touches the OTel
// package globals — partial state is impossible.
func New(ctx context.Context, cfg Config, options ...Option) (*Provider, error) {
	if !cfg.Enabled {
		return newDisabledProvider(), nil
	}
	res := buildResource(cfg)

	o := opts{}
	for _, opt := range options {
		opt(&o)
	}

	useSyncProcessor := o.spanExporterOverride != nil
	var spanExporter sdktrace.SpanExporter
	if o.spanExporterOverride != nil {
		spanExporter = o.spanExporterOverride
	} else {
		var err error
		spanExporter, err = buildSpanExporter(ctx, cfg)
		if err != nil {
			return nil, err
		}
	}

	tpOpts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SampleRate))),
	}
	if spanExporter != nil {
		if useSyncProcessor {
			tpOpts = append(tpOpts, sdktrace.WithSyncer(spanExporter))
		} else {
			tpOpts = append(tpOpts, sdktrace.WithBatcher(spanExporter,
				sdktrace.WithMaxQueueSize(cfg.SpanQueueSize),
			))
		}
	}
	tp := sdktrace.NewTracerProvider(tpOpts...)
	mp := metricnoop.NewMeterProvider()

	// Install the OTel globals atomically. The W3C TraceContext
	// propagator is required for any kafka/HTTP producer instrumentation
	// that injects traceparent headers via the global propagator.
	otelapi.SetTracerProvider(tp)
	otelapi.SetMeterProvider(mp)
	otelapi.SetTextMapPropagator(propagation.TraceContext{})

	return &Provider{
		tp:       tp,
		mp:       mp,
		res:      res,
		shutdown: tp.Shutdown,
	}, nil
}

// buildSpanExporter constructs the OTel trace exporter for cfg.Exporter.
// Returns (nil, nil) for ExporterNoop so the caller skips
// BatchSpanProcessor wiring entirely.
func buildSpanExporter(ctx context.Context, cfg Config) (sdktrace.SpanExporter, error) {
	switch cfg.Exporter {
	case ExporterNoop:
		return nil, nil
	case ExporterStdout:
		return stdouttrace.New(stdouttrace.WithWriter(os.Stdout))
	case ExporterOTLPHTTP:
		exporterOpts := []otlptracehttp.Option{otlptracehttp.WithEndpointURL(cfg.Endpoint)}
		if cfg.Insecure {
			exporterOpts = append(exporterOpts, otlptracehttp.WithInsecure())
		}
		return otlptracehttp.New(ctx, exporterOpts...)
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedExporter, cfg.Exporter)
	}
}

// TracerProvider returns the Provider's tracer factory. Always non-nil;
// on the disabled path it returns the OTel SDK's no-op TracerProvider.
func (p *Provider) TracerProvider() trace.TracerProvider { return p.tp }

// Tracer returns a named Tracer from the Provider's TracerProvider.
// Callers pass an instrumentation-scope identifier (e.g.
// "myservice/handler"). On the disabled path the returned Tracer is a
// no-op — Start() returns non-recording spans with zero allocation.
func (p *Provider) Tracer(name string) trace.Tracer { return p.tp.Tracer(name) }

// MeterProvider returns the Provider's meter factory. Always non-nil;
// on the disabled path it returns the OTel SDK's no-op MeterProvider.
func (p *Provider) MeterProvider() metric.MeterProvider { return p.mp }

// Resource returns the OTel Resource attached to the enabled-path
// TracerProvider. Returns nil for a disabled Provider.
func (p *Provider) Resource() *resource.Resource { return p.res }

// Shutdown flushes any in-flight spans within the supplied context's
// deadline. Idempotent: a second invocation is a no-op that returns
// nil.
func (p *Provider) Shutdown(ctx context.Context) error {
	if p.shutdown == nil {
		return nil
	}
	fn := p.shutdown
	p.shutdown = nil
	return fn(ctx)
}

func newDisabledProvider() *Provider {
	return &Provider{
		tp:       tracenoop.NewTracerProvider(),
		mp:       metricnoop.NewMeterProvider(),
		shutdown: nil,
	}
}

// buildResource builds the OTel Resource from validated Config.
// service.name falls back to detectServiceName when cfg.ServiceName is
// empty. host.name is omitted on os.Hostname() error so operators can
// notice the missing attribute rather than have it silently disappear.
func buildResource(cfg Config) *resource.Resource {
	name := cfg.ServiceName
	if name == "" {
		name = detectServiceName()
	}
	attrs := []attribute.KeyValue{
		attribute.String(resourceServiceName, name),
		attribute.String(resourceServiceVersion, cfg.ServiceVersion),
		attribute.String(resourceDeployEnv, cfg.DeployEnv),
	}
	if host, err := os.Hostname(); err == nil {
		attrs = append(attrs, attribute.String(resourceHostName, host))
	} else {
		slog.Warn("tracer: hostname unavailable, omitting host.name", "err", err)
	}
	return resource.NewSchemaless(attrs...)
}

// detectServiceName returns the binary's filename, path-stripped and
// .exe-suffix-removed. Uses os.Executable() (resolves symlinks for
// container deployments) and falls back to os.Args[0] on error.
func detectServiceName() string {
	if exe, err := os.Executable(); err == nil {
		return strings.TrimSuffix(filepath.Base(exe), ".exe")
	}
	if len(os.Args) > 0 {
		return strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe")
	}
	return "unknown"
}
