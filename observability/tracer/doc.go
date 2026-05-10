// Package tracer constructs an OpenTelemetry trace Provider configured
// for the fleet's deployment patterns: OTLP-HTTP / stdout / no-op
// exporter selection, ParentBased sampling, deterministic shutdown, and
// W3C TraceContext propagation.
//
// Constructed once per binary at startup; shut down via a deferred
// Provider.Shutdown on process exit. When Config.Enabled is false the
// Provider returns no-op TracerProvider and MeterProvider, never starts
// a background goroutine, and never opens a network connection — so
// callers can pass *Provider unconditionally.
//
// Test seam: WithSpanExporter overrides the constructed exporter with a
// caller-supplied one. Combined with otel/sdk/trace/tracetest, this
// lets tests assert exactly which spans the production code emits
// without batching, timers, or flush races.
package tracer
