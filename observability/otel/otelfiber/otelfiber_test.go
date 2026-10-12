package otelfiber_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	otelapi "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/samsonnaze5/aeternixth-go-lib/observability/otel/otelfiber"
)

const incomingTraceparent = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"

func newApp(t *testing.T) (*fiber.App, *tracetest.InMemoryExporter) {
	t.Helper()
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	prevTP, prevProp := otelapi.GetTracerProvider(), otelapi.GetTextMapPropagator()
	otelapi.SetTracerProvider(tp)
	otelapi.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() {
		otelapi.SetTracerProvider(prevTP)
		otelapi.SetTextMapPropagator(prevProp)
		_ = tp.Shutdown(context.Background())
	})

	app := fiber.New()
	app.Use(otelfiber.Middleware())
	app.Get("/users/:id", func(c *fiber.Ctx) error {
		if !trace.SpanFromContext(c.UserContext()).SpanContext().IsValid() {
			return fiber.NewError(fiber.StatusTeapot, "no span in UserContext")
		}
		return c.SendString("ok")
	})
	app.Get("/down", func(c *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusServiceUnavailable, "down")
	})
	return app, exp
}

func attr(span tracetest.SpanStub, key string) (attribute.Value, bool) {
	for _, kv := range span.Attributes {
		if string(kv.Key) == key {
			return kv.Value, true
		}
	}
	return attribute.Value{}, false
}

func TestMiddleware(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		traceparent string
		wantStatus  int
		wantName    string
		wantRoute   string
		wantError   bool
	}{
		{name: "matched route", path: "/users/42", wantStatus: 200, wantName: "GET /users/:id", wantRoute: "/users/:id"},
		{name: "continues incoming trace", path: "/users/42", traceparent: incomingTraceparent, wantStatus: 200, wantName: "GET /users/:id", wantRoute: "/users/:id"},
		{name: "handler error goes through ErrorHandler", path: "/down", wantStatus: 503, wantName: "GET /down", wantRoute: "/down", wantError: true},
		{name: "unmatched route named after the middleware route", path: "/nope/42", wantStatus: 404, wantName: "GET /", wantRoute: "/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, exp := newApp(t)

			req := httptest.NewRequest("GET", tt.path, nil)
			if tt.traceparent != "" {
				req.Header.Set("traceparent", tt.traceparent)
			}
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			if resp.StatusCode != tt.wantStatus {
				t.Errorf("response status: want %d, got %d", tt.wantStatus, resp.StatusCode)
			}

			spans := exp.GetSpans()
			if len(spans) != 1 {
				t.Fatalf("spans: want 1, got %d", len(spans))
			}
			s := spans[0]
			if s.Name != tt.wantName {
				t.Errorf("name: want %q, got %q", tt.wantName, s.Name)
			}
			if s.SpanKind != trace.SpanKindServer {
				t.Errorf("kind: want server, got %v", s.SpanKind)
			}
			if v, _ := attr(s, "http.response.status_code"); v.AsInt64() != int64(tt.wantStatus) {
				t.Errorf("status attribute: want %d, got %v", tt.wantStatus, v.Emit())
			}
			if route, _ := attr(s, "http.route"); route.AsString() != tt.wantRoute {
				t.Errorf("http.route: want %q, got %q", tt.wantRoute, route.AsString())
			}
			if (s.Status.Code == codes.Error) != tt.wantError {
				t.Errorf("span error status: want %v, got %v", tt.wantError, s.Status.Code)
			}
			if tt.traceparent != "" && s.Parent.TraceID().String() != "4bf92f3577b34da6a3ce929d0e0e4736" {
				t.Errorf("trace id: want continued from traceparent, got %s", s.Parent.TraceID())
			}
			for _, kv := range s.Attributes {
				if tt.path != tt.wantRoute && kv.Value.Emit() == tt.path {
					t.Errorf("attribute %s records the raw path %q", kv.Key, tt.path)
				}
			}
		})
	}
}
