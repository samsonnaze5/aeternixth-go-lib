// Package otel constructs the OpenTelemetry SDK for one binary in the
// shape every fleet service uses: traces (OTLP-HTTP / stdout / no-op,
// ParentBased ratio sampling), an optional Prometheus → OTLP metric
// bridge, and an optional OTLP log pipeline for the otelslog / otelzap
// bridges — all behind one Provider with one Shutdown.
//
// The package name shadows go.opentelemetry.io/otel; import the upstream
// API under an alias (e.g. otelapi) in files that need both.
//
// Instrumentation adapters live in sub-packages:
//
//   - otelkafka: W3C trace-context carrier for segmentio/kafka-go headers.
//   - otelgorm: GORM query spans that only appear under a parent span.
//   - otelfiber: Fiber middleware that opens the server root span.
//
// When Config.Enabled is false New returns no-op providers, never starts
// a goroutine, and never opens a network connection.
package otel
