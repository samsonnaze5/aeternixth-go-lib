package health

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewServer_EmptyAddr_ReturnsNil(t *testing.T) {
	got := NewServer("", map[string]Pinger{"db": &fakePinger{}}, prometheus.NewRegistry())
	if got != nil {
		t.Errorf("want nil for empty addr, got %#v", got)
	}
}

func TestNewServer_HandlersMounted(t *testing.T) {
	srv := NewServer(":0", map[string]Pinger{"db": &fakePinger{}}, nil)
	if srv == nil {
		t.Fatal("want non-nil *http.Server, got nil")
	}
	if srv.Addr != ":0" {
		t.Errorf("Addr: want :0, got %q", srv.Addr)
	}
	if srv.ReadHeaderTimeout == 0 {
		t.Errorf("ReadHeaderTimeout: want non-zero (slowloris defence)")
	}

	mux, ok := srv.Handler.(*http.ServeMux)
	if !ok {
		t.Fatalf("Handler: want *http.ServeMux, got %T", srv.Handler)
	}

	// Probe both probe paths via the assigned mux.
	for _, p := range []string{PathLivez, PathReadyz} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, p, nil))
		if rec.Code == http.StatusNotFound {
			t.Errorf("%s: want mounted, got 404", p)
		}
	}
}

func TestNewServer_NilRegistry_OmitsMetrics(t *testing.T) {
	srv := NewServer(":0", map[string]Pinger{}, nil)
	mux := srv.Handler.(*http.ServeMux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, PathMetrics, nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("/metrics: want 404 (registry was nil), got %d", rec.Code)
	}
}

func TestNewServer_WithRegistry_MountsMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	srv := NewServer(":0", map[string]Pinger{}, registry)
	mux := srv.Handler.(*http.ServeMux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, PathMetrics, nil))
	if rec.Code != http.StatusOK {
		t.Errorf("/metrics: want 200 (registry supplied), got %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got == "" {
		t.Errorf("/metrics Content-Type: want non-empty (Prometheus exposition), got empty")
	}
}
