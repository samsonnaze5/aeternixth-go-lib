package health

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// TestReadyHandler_ConcurrentInvocations exercises the parallel-execution
// path under -race to confirm there is no shared state between
// concurrent /readyz requests. Each request constructs its own
// readyResponse and mutex inside runChecks; if a future change leaks a
// shared map or counter into the call path, this test is the first place
// it fires.
func TestReadyHandler_ConcurrentInvocations(t *testing.T) {
	checks := map[string]Pinger{
		"db":    &fakePinger{},
		"cache": &fakePinger{},
		"kafka": &fakePinger{},
	}
	h := ReadyHandler(checks)
	req := httptest.NewRequest(http.MethodGet, PathReadyz, nil)

	const goroutines = 64
	const iterations = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				rec := httptest.NewRecorder()
				h.ServeHTTP(rec, req)
				if rec.Code != http.StatusOK {
					t.Errorf("status: want 200, got %d", rec.Code)
					return
				}
			}
		}()
	}
	wg.Wait()
}
