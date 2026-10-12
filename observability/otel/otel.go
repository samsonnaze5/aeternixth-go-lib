package otel

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	prombridge "go.opentelemetry.io/contrib/bridges/prometheus"
	otelapi "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	otellog "go.opentelemetry.io/otel/log"
	lognoop "go.opentelemetry.io/otel/log/noop"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"

	"github.com/prometheus/client_golang/prometheus"
)

// Exporter selects the OTel export transport.
type Exporter string

const (
	// ExporterNoop disables span export. Spans are still created with
	// trace IDs (so logs can correlate) but never leave the process.
	ExporterNoop Exporter = "noop"

	// ExporterStdout writes spans to os.Stdout — convenient for local
	// development.
	ExporterStdout Exporter = "stdout"

	// ExporterOTLPHTTP sends spans (and, when enabled, metrics and logs)
	// via OTLP/HTTP — the production target for an OTel collector.
	ExporterOTLPHTTP Exporter = "otlp_http"
)

// OTel resource attribute keys (OpenTelemetry Resource Semantic Conventions).
const (
	resourceServiceName    = "service.name"
	resourceServiceVersion = "service.version"
	resourceDeployEnv      = "deployment.environment"
	resourceHostName       = "host.name"
)

// otlpLogsPath is the OTLP/HTTP logs path. otlploghttp's WithEndpointURL
// resets the path to "" when the URL carries none, so it is restored
// explicitly.
const otlpLogsPath = "/v1/logs"

var (
	// ErrUnsupportedExporter is returned by New when Config.Exporter is
	// not one of the Exporter* constants.
	ErrUnsupportedExporter = errors.New("otel: unsupported exporter")

	// ErrMissingGatherer is returned by New when Config.PrometheusBridge
	// is true but no WithPrometheusGatherer option was supplied.
	ErrMissingGatherer = errors.New("otel: prometheus bridge enabled without a gatherer")
)

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

	// ServiceVersion is recorded as service.version.
	ServiceVersion string

	// DeployEnv is recorded as deployment.environment.
	DeployEnv string

	// Exporter selects the export transport. See the Exporter* constants.
	Exporter Exporter

	// Endpoint is the OTLP collector base URL (e.g.
	// "http://otel-collector:4318"). Used only with ExporterOTLPHTTP.
	Endpoint string

	// Insecure disables TLS for OTLP transport.
	Insecure bool

	// SampleRate is the TraceIDRatioBased sampler ratio in [0.0, 1.0],
	// wrapped in ParentBased so child spans follow their parent.
	SampleRate float64

	// SpanQueueSize bounds the BatchSpanProcessor's in-memory queue. Zero
	// (or negative) keeps the SDK default (2048) — a zero-length queue would
	// drop almost every span silently.
	SpanQueueSize int

	// PrometheusBridge pushes every metric of the gatherer supplied via
	// WithPrometheusGatherer to the collector every MetricsInterval.
	// Only takes effect with ExporterOTLPHTTP.
	PrometheusBridge bool

	// MetricsInterval is the push period for PrometheusBridge.
	MetricsInterval time.Duration

	// LogsBridge builds an OTLP log pipeline exposed via
	// Provider.LoggerProvider, for use with the otelslog / otelzap
	// bridges. Only takes effect with ExporterOTLPHTTP.
	LogsBridge bool
}

// Provider owns the OpenTelemetry SDK lifecycle for one binary: traces,
// the optional Prometheus metric bridge, and the optional log pipeline.
// Constructed once at startup via New; shut down via a deferred
// Provider.Shutdown on process exit.
//
// On the disabled path every accessor returns the OTel no-op
// equivalent, so callers can pass *Provider unconditionally.
type Provider struct {
	tp       trace.TracerProvider
	mp       metric.MeterProvider
	lp       otellog.LoggerProvider
	res      *resource.Resource // nil on disabled path
	shutdown func(context.Context) error
}

// Option customises New construction.
type Option func(*opts)

type opts struct {
	spanExporterOverride sdktrace.SpanExporter
	gatherer             prometheus.Gatherer
}

// WithSpanExporter overrides the trace exporter that New would otherwise
// build from Config.Exporter. Intended for tests that inject
// tracetest.NewInMemoryExporter: the SDK then uses a synchronous span
// processor so every span is visible the moment span.End is called.
func WithSpanExporter(exp sdktrace.SpanExporter) Option {
	return func(o *opts) { o.spanExporterOverride = exp }
}

// WithPrometheusGatherer supplies the registry bridged into OTLP when
// Config.PrometheusBridge is true. Pass the same *prometheus.Registry the
// binary's collectors are registered on.
func WithPrometheusGatherer(g prometheus.Gatherer) Option {
	return func(o *opts) { o.gatherer = g }
}

// New builds a Provider from validated Config and installs it as the
// OTel global tracer provider, meter provider, and W3C TraceContext +
// Baggage propagator.
//
// Every fallible step (exporter construction) runs before any SDK
// processor or reader is started, and before the globals are touched:
// on error nothing is left running and the globals are unchanged.
func New(ctx context.Context, cfg Config, options ...Option) (*Provider, error) {
	if !cfg.Enabled {
		return newDisabledProvider(), nil
	}

	o := opts{}
	for _, opt := range options {
		opt(&o)
	}
	otlp := cfg.Exporter == ExporterOTLPHTTP
	bridgeMetrics := cfg.PrometheusBridge && otlp
	if bridgeMetrics && o.gatherer == nil {
		return nil, ErrMissingGatherer
	}

	var built []func(context.Context) error
	fail := func(err error) (*Provider, error) {
		for _, shutdown := range built {
			_ = shutdown(ctx)
		}
		return nil, err
	}

	spanExp := o.spanExporterOverride
	if spanExp == nil {
		var err error
		if spanExp, err = buildSpanExporter(ctx, cfg); err != nil {
			return fail(err)
		}
	}
	if spanExp != nil {
		built = append(built, spanExp.Shutdown)
	}

	var metricExp sdkmetric.Exporter
	if bridgeMetrics {
		metricOpts := []otlpmetrichttp.Option{otlpmetrichttp.WithEndpointURL(cfg.Endpoint)}
		if cfg.Insecure {
			metricOpts = append(metricOpts, otlpmetrichttp.WithInsecure())
		}
		var err error
		if metricExp, err = otlpmetrichttp.New(ctx, metricOpts...); err != nil {
			return fail(fmt.Errorf("otel: build metric exporter: %w", err))
		}
		built = append(built, metricExp.Shutdown)
	}

	var logExp sdklog.Exporter
	if cfg.LogsBridge && otlp {
		logOpts := []otlploghttp.Option{
			otlploghttp.WithEndpointURL(cfg.Endpoint),
			otlploghttp.WithURLPath(otlpLogsPath),
		}
		if cfg.Insecure {
			logOpts = append(logOpts, otlploghttp.WithInsecure())
		}
		var err error
		if logExp, err = otlploghttp.New(ctx, logOpts...); err != nil {
			return fail(fmt.Errorf("otel: build log exporter: %w", err))
		}
	}

	// Infallible from here on: start processors/readers, then globals.
	res := buildResource(cfg)
	shutdowns := make([]func(context.Context) error, 0, 3)

	tpOpts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SampleRate))),
	}
	switch {
	case o.spanExporterOverride != nil:
		tpOpts = append(tpOpts, sdktrace.WithSyncer(spanExp))
	case spanExp != nil:
		var bspOpts []sdktrace.BatchSpanProcessorOption
		if cfg.SpanQueueSize > 0 {
			bspOpts = append(bspOpts, sdktrace.WithMaxQueueSize(cfg.SpanQueueSize))
		}
		tpOpts = append(tpOpts, sdktrace.WithBatcher(spanExp, bspOpts...))
	}
	tp := sdktrace.NewTracerProvider(tpOpts...)
	shutdowns = append(shutdowns, tp.Shutdown)

	var mp metric.MeterProvider = metricnoop.NewMeterProvider()
	if metricExp != nil {
		reader := sdkmetric.NewPeriodicReader(metricExp,
			sdkmetric.WithInterval(cfg.MetricsInterval),
			sdkmetric.WithProducer(prombridge.NewMetricProducer(prombridge.WithGatherer(o.gatherer))),
		)
		sdkMP := sdkmetric.NewMeterProvider(sdkmetric.WithResource(res), sdkmetric.WithReader(reader))
		mp = sdkMP
		shutdowns = append(shutdowns, sdkMP.Shutdown)
	}

	var lp otellog.LoggerProvider = lognoop.NewLoggerProvider()
	if logExp != nil {
		sdkLP := sdklog.NewLoggerProvider(
			sdklog.WithResource(res),
			sdklog.WithProcessor(sdklog.NewBatchProcessor(logExp)),
		)
		lp = sdkLP
		shutdowns = append(shutdowns, sdkLP.Shutdown)
	}

	otelapi.SetTracerProvider(tp)
	otelapi.SetMeterProvider(mp)
	otelapi.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &Provider{
		tp:  tp,
		mp:  mp,
		lp:  lp,
		res: res,
		shutdown: func(ctx context.Context) error {
			var errs []error
			for _, shutdown := range shutdowns {
				if err := shutdown(ctx); err != nil {
					errs = append(errs, err)
				}
			}
			return errors.Join(errs...)
		},
	}, nil
}

// buildSpanExporter returns (nil, nil) for ExporterNoop so New skips
// span-processor wiring entirely.
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
		exp, err := otlptracehttp.New(ctx, exporterOpts...)
		if err != nil {
			return nil, fmt.Errorf("otel: build span exporter: %w", err)
		}
		return exp, nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedExporter, cfg.Exporter)
	}
}

// TracerProvider returns the Provider's tracer factory. Always non-nil.
func (p *Provider) TracerProvider() trace.TracerProvider { return p.tp }

// Tracer returns a named Tracer from the Provider's TracerProvider.
func (p *Provider) Tracer(name string) trace.Tracer { return p.tp.Tracer(name) }

// MeterProvider returns the Provider's meter factory. Always non-nil; a
// no-op unless the Prometheus bridge is active.
func (p *Provider) MeterProvider() metric.MeterProvider { return p.mp }

// LoggerProvider returns the OTel log pipeline for the otelslog / otelzap
// bridges. Always non-nil; a no-op unless Config.LogsBridge is active.
func (p *Provider) LoggerProvider() otellog.LoggerProvider { return p.lp }

// Resource returns the OTel Resource of the enabled path. Returns nil
// for a disabled Provider.
func (p *Provider) Resource() *resource.Resource { return p.res }

// Shutdown flushes in-flight spans, metrics, and log records within the
// supplied context's deadline. Idempotent: a second call returns nil.
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
		tp: tracenoop.NewTracerProvider(),
		mp: metricnoop.NewMeterProvider(),
		lp: lognoop.NewLoggerProvider(),
	}
}

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
		slog.Warn("otel: hostname unavailable, omitting host.name", "err", err)
	}
	return resource.NewSchemaless(attrs...)
}

// detectServiceName returns the binary's filename, path-stripped and
// .exe-suffix-removed.
func detectServiceName() string {
	if exe, err := os.Executable(); err == nil {
		return strings.TrimSuffix(filepath.Base(exe), ".exe")
	}
	if len(os.Args) > 0 {
		return strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe")
	}
	return "unknown"
}
