// Package otelgorm records GORM queries as child spans of the caller's
// active span.
//
// It wraps github.com/uptrace/opentelemetry-go-extra/otelgorm with two
// fixed policies:
//
//   - Bind-parameter values are never recorded (WithoutQueryVariables),
//     so db.statement carries no PII.
//   - A query issued without an active span produces no span. Upstream
//     otelgorm would start a new root trace per query, which turns every
//     background poller (outbox, reconcilers, schedulers) into a stream
//     of single-span traces.
//
// Pass the request/message context with db.WithContext(ctx) — a query
// on context.Background() is, by design, not traced.
package otelgorm

import (
	"context"
	"errors"

	uptraceotelgorm "github.com/uptrace/opentelemetry-go-extra/otelgorm"
	otelapi "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"
	"gorm.io/gorm"
)

// ErrNilDB is returned by Register when db is nil.
var ErrNilDB = errors.New("otelgorm: nil *gorm.DB")

// Register installs the tracing plugin on db using the global tracer
// provider (resolved lazily, so it may run before otel.New).
func Register(db *gorm.DB) error {
	if db == nil {
		return ErrNilDB
	}
	return db.Use(uptraceotelgorm.NewPlugin(
		uptraceotelgorm.WithoutQueryVariables(),
		uptraceotelgorm.WithTracerProvider(childOnlyTracerProvider{tp: otelapi.GetTracerProvider()}),
	))
}

// childOnlyTracerProvider hands out tracers that start a span only when
// ctx already carries a valid span context.
type childOnlyTracerProvider struct {
	embedded.TracerProvider
	tp trace.TracerProvider
}

func (p childOnlyTracerProvider) Tracer(name string, opts ...trace.TracerOption) trace.Tracer {
	return childOnlyTracer{tr: p.tp.Tracer(name, opts...)}
}

type childOnlyTracer struct {
	embedded.Tracer
	tr trace.Tracer
}

func (t childOnlyTracer) Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if !trace.SpanContextFromContext(ctx).IsValid() {
		return ctx, trace.SpanFromContext(ctx)
	}
	return t.tr.Start(ctx, name, opts...)
}
