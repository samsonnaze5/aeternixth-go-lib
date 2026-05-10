package healthfiber_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/samsonnaze5/aeternixth-go-lib/observability/health"
	"github.com/samsonnaze5/aeternixth-go-lib/observability/health/healthfiber"
	"github.com/samsonnaze5/aeternixth-go-lib/observability/health/internal/core"
)

type fakePinger struct{ err error }

func (f fakePinger) Ping(_ context.Context) error { return f.err }

func TestLiveHandler_ParityWithCore(t *testing.T) {
	app := fiber.New()
	app.Get(health.PathLivez, healthfiber.LiveHandler())

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, health.PathLivez, nil))
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: want 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != string(core.LiveBody) {
		t.Errorf("body: want %q, got %q (parity violation with core.LiveBody)", core.LiveBody, body)
	}
	if got := resp.Header.Get("Content-Type"); got != fiber.MIMEApplicationJSON {
		t.Errorf("Content-Type: want %q, got %q", fiber.MIMEApplicationJSON, got)
	}
}

func TestReadyHandler_HappyPath(t *testing.T) {
	app := fiber.New()
	app.Get(health.PathReadyz, healthfiber.ReadyHandler(map[string]health.Pinger{
		"db": fakePinger{},
	}))

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, health.PathReadyz, nil))
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: want 200, got %d", resp.StatusCode)
	}
}

func TestReadyHandler_FailurePath(t *testing.T) {
	app := fiber.New()
	app.Get(health.PathReadyz, healthfiber.ReadyHandler(map[string]health.Pinger{
		"db": fakePinger{err: errors.New("db down")},
	}))

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, health.PathReadyz, nil))
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status: want 503, got %d", resp.StatusCode)
	}
}

func TestMount_RegistersBothPaths(t *testing.T) {
	app := fiber.New()
	healthfiber.Mount(app, map[string]health.Pinger{"db": fakePinger{}})

	for _, path := range []string{health.PathLivez, health.PathReadyz} {
		resp, err := app.Test(httptest.NewRequest(http.MethodGet, path, nil))
		if err != nil {
			t.Fatalf("Test %s: %v", path, err)
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			t.Errorf("%s: want mounted, got 404", path)
		}
	}
}
