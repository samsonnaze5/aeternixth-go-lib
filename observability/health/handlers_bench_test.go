package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func BenchmarkLiveHandler(b *testing.B) {
	h := LiveHandler()
	req := httptest.NewRequest(http.MethodGet, PathLivez, nil)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}
}

func BenchmarkReadyHandler_AllPass_5Pingers(b *testing.B) {
	checks := map[string]Pinger{
		"db1":   &fakePinger{},
		"db2":   &fakePinger{},
		"redis": &fakePinger{},
		"cache": &fakePinger{},
		"kafka": &fakePinger{},
	}
	h := ReadyHandler(checks)
	req := httptest.NewRequest(http.MethodGet, PathReadyz, nil)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}
}

func BenchmarkReadyHandler_OneSlow_5Pingers(b *testing.B) {
	// One of the five takes longer than the others (still under the
	// 800 ms internal deadline). Measures the aggregator's behavior when
	// it must wait for the slowest pinger.
	checks := map[string]Pinger{
		"db1":   &fakePinger{},
		"db2":   &fakePinger{},
		"redis": &fakePinger{},
		"cache": &fakePinger{},
		"slow":  &slowPinger{}, // ~1ms; not flapping the deadline
	}
	h := ReadyHandler(checks)
	req := httptest.NewRequest(http.MethodGet, PathReadyz, nil)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}
}

// slowPinger returns nil after a small synthetic delay used by the
// "OneSlow" benchmark to model heterogeneous dependencies. Kept short
// (1 ms) so the benchmark completes in reasonable wall time.
type slowPinger struct{}

func (slowPinger) Ping(ctx context.Context) error {
	timer := time.NewTimer(time.Millisecond)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
