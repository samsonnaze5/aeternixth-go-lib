package healthfiber_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/gofiber/fiber/v2"

	"github.com/samsonnaze5/aeternixth-go-lib/observability/health"
	"github.com/samsonnaze5/aeternixth-go-lib/observability/health/healthfiber"
)

type stubPinger struct{}

func (stubPinger) Ping(_ context.Context) error { return nil }

func ExampleMount() {
	app := fiber.New()

	// Imagine fiberprometheus or another /metrics setup happening here.
	// The lib's probes coexist with it.
	healthfiber.Mount(app, map[string]health.Pinger{
		"postgres": stubPinger{},
	})

	resp, _ := app.Test(httptest.NewRequest(http.MethodGet, health.PathLivez, nil))
	defer resp.Body.Close()
	fmt.Println(resp.StatusCode)
	// Output: 200
}
