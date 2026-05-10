package health_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/samsonnaze5/aeternixth-go-lib/observability/health"
)

// stubPinger is a minimal Pinger used only by the runnable examples.
type stubPinger struct{ err error }

func (s stubPinger) Ping(_ context.Context) error { return s.err }

func ExampleLiveHandler() {
	srv := httptest.NewServer(health.LiveHandler())
	defer srv.Close()

	resp, _ := http.Get(srv.URL)
	defer resp.Body.Close()
	fmt.Println(resp.StatusCode)
	// Output: 200
}

func ExampleReadyHandler() {
	checks := map[string]health.Pinger{
		"db":    stubPinger{},
		"cache": stubPinger{err: errors.New("connection refused")},
	}
	srv := httptest.NewServer(health.ReadyHandler(checks))
	defer srv.Close()

	resp, _ := http.Get(srv.URL + health.PathReadyz)
	defer resp.Body.Close()
	fmt.Println(resp.StatusCode)
	// Output: 503
}

func ExampleNewServer() {
	// Empty addr opts out — useful for one-shot binaries that don't
	// want a probe port.
	if srv := health.NewServer("", nil, nil); srv == nil {
		fmt.Println("opted out")
	}
	// Output: opted out
}

func ExamplePingFunc() {
	// Wrap a one-off function (HTTP API check, gRPC health, custom lag
	// threshold, ...) into a Pinger inline — no struct + method needed.
	checks := map[string]health.Pinger{
		"upstream-api": health.PingFunc(func(_ context.Context) error {
			// pretend we just dialled the API and it answered 200 OK
			return nil
		}),
	}

	srv := httptest.NewServer(health.ReadyHandler(checks))
	defer srv.Close()

	resp, _ := http.Get(srv.URL + health.PathReadyz)
	defer resp.Body.Close()
	fmt.Println(resp.StatusCode)
	// Output: 200
}
