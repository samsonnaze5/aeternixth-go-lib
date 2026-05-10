package healthfiber_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/samsonnaze5/aeternixth-go-lib/observability/health"
	"github.com/samsonnaze5/aeternixth-go-lib/observability/health/healthfiber"
)

// TestMount_CoexistsWithMetrics simulates the production layout where a
// Fiber app already has /metrics mounted (typically via fiberprometheus)
// and the lib's probes are added alongside. Verifies all three paths
// respond correctly without route conflict — the operator-facing
// guarantee the Fiber adapter must uphold per US2 acceptance scenario 1.
//
// fiberprometheus itself is not pulled in as a test dep; a stub /metrics
// route is sufficient to prove Mount does not collide with arbitrary
// pre-existing routes on the same app.
func TestMount_CoexistsWithMetrics(t *testing.T) {
	app := fiber.New()

	// Stub for whatever has already mounted /metrics in the real app.
	app.Get("/metrics", func(c *fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, "text/plain; version=0.0.4")
		return c.SendString("# stub metrics\n")
	})

	// Lib mount lands afterwards — must not conflict.
	healthfiber.Mount(app, map[string]health.Pinger{
		"db": fakePinger{},
	})

	// All three paths respond.
	for _, tc := range []struct {
		path       string
		wantStatus int
	}{
		{"/metrics", http.StatusOK},
		{health.PathLivez, http.StatusOK},
		{health.PathReadyz, http.StatusOK},
	} {
		t.Run(tc.path, func(t *testing.T) {
			resp, err := app.Test(httptest.NewRequest(http.MethodGet, tc.path, nil))
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			resp.Body.Close()
			if resp.StatusCode != tc.wantStatus {
				t.Errorf("status: want %d, got %d", tc.wantStatus, resp.StatusCode)
			}
		})
	}
}
