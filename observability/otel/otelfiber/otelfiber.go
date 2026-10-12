// Package otelfiber opens the server root span for every Fiber request,
// continuing the caller's trace when the request carries a W3C
// traceparent header.
//
// Every string taken from the Fiber context is cloned before it reaches
// a span: Fiber returns strings that alias fasthttp's reusable request
// buffer, while spans are exported asynchronously after the handler has
// returned and the buffer has been recycled.
package otelfiber

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	otelapi "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "github.com/samsonnaze5/aeternixth-go-lib/observability/otel/otelfiber"

// Semantic-convention attribute keys (HTTP server spans).
const (
	attrMethod     = "http.request.method"
	attrRoute      = "http.route"
	attrStatusCode = "http.response.status_code"
)

// Option customises Middleware.
type Option func(*options)

type options struct{ public bool }

// WithPublicEndpoint marks the app as reachable by callers outside the
// fleet (e.g. third-party webhooks). An incoming traceparent is then only
// recorded as a span link: the server span starts a new trace sampled by
// the local sampler, so a caller cannot force sampling or inject baggage
// that would travel on through Kafka headers.
func WithPublicEndpoint() Option {
	return func(o *options) { o.public = true }
}

// Middleware returns a handler that starts a SpanKindServer span named
// "<METHOD> <route template>" (e.g. "GET /users/:id"), stores it in
// c.UserContext(), and records the response status code. 5xx responses
// mark the span as an error.
//
// Register it with app.Use before any other middleware that reads
// c.UserContext(). When a downstream handler returns an error the
// middleware invokes the app's ErrorHandler itself — so the span records
// the status the client actually receives — and returns nil.
//
// The raw request path is never recorded: it may carry identifiers. A
// request that matches no handler is named after the last middleware
// route it passed through (e.g. "GET /" for app.Use without a prefix).
func Middleware(opts ...Option) fiber.Handler {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	tracer := otelapi.GetTracerProvider().Tracer(tracerName)

	return func(c *fiber.Ctx) error {
		method := strings.Clone(c.Method())
		startOpts := []trace.SpanStartOption{
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(attribute.String(attrMethod, method)),
		}
		ctx := c.UserContext()
		remote := otelapi.GetTextMapPropagator().Extract(ctx, requestCarrier{c: c})
		if !o.public {
			ctx = remote
		} else if sc := trace.SpanContextFromContext(remote); sc.IsValid() {
			startOpts = append(startOpts, trace.WithLinks(trace.Link{SpanContext: sc}))
		}
		ctx, span := tracer.Start(ctx, method, startOpts...)
		defer span.End()
		c.SetUserContext(ctx)

		if err := c.Next(); err != nil {
			span.RecordError(err)
			if handlerErr := c.App().ErrorHandler(c, err); handlerErr != nil {
				_ = c.SendStatus(fiber.StatusInternalServerError)
			}
		}

		route := c.Route().Path
		status := c.Response().StatusCode()
		span.SetName(method + " " + route)
		span.SetAttributes(
			attribute.String(attrRoute, route),
			attribute.Int(attrStatusCode, status),
		)
		if status >= fiber.StatusInternalServerError {
			span.SetStatus(codes.Error, "")
		}
		return nil
	}
}

// requestCarrier reads propagation headers from the Fiber request,
// cloning each value out of the fasthttp buffer.
type requestCarrier struct{ c *fiber.Ctx }

var _ propagation.TextMapCarrier = requestCarrier{}

func (r requestCarrier) Get(key string) string { return strings.Clone(r.c.Get(key)) }

// Set is a no-op: the carrier is only used for extraction.
func (requestCarrier) Set(string, string) {}

func (r requestCarrier) Keys() []string {
	var keys []string
	r.c.Request().Header.VisitAll(func(k, _ []byte) {
		keys = append(keys, string(k))
	})
	return keys
}
