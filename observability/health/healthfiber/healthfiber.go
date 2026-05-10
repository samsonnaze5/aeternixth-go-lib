package healthfiber

import (
	"github.com/gofiber/fiber/v2"

	"github.com/samsonnaze5/aeternixth-go-lib/observability/health"
	"github.com/samsonnaze5/aeternixth-go-lib/observability/health/internal/core"
)

// LiveHandler returns a Fiber handler for /health/livez. It writes the
// same body as the net/http LiveHandler (core.LiveBody) and sets
// Content-Type to application/json — byte-for-byte identical to the
// worker variant per the spec's parity requirement.
func LiveHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
		c.Status(fiber.StatusOK)
		return c.Send(core.LiveBody)
	}
}

// ReadyHandler returns a Fiber handler for /health/readyz. It delegates
// to core.Evaluate, which runs every Pinger in parallel under the
// 800 ms internal deadline (same logic as the net/http variant).
//
// The Fiber request's UserContext() is forwarded to Evaluate so request
// cancellation (deadlines, client disconnect) propagates into the
// pingers via the standard context.Context contract.
func ReadyHandler(checks map[string]health.Pinger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		body, status := core.Evaluate(c.UserContext(), checks)
		c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
		c.Status(status)
		return c.Send(body)
	}
}

// Mount registers /health/livez and /health/readyz on the supplied
// fiber.Router using the canonical path constants from the health
// package. It does NOT manage /metrics — Fiber adopters typically have
// fiberprometheus mounted at /metrics already, and Mount leaves that
// path untouched.
//
// Pass a fiber.Router (not a *fiber.App) when mounting on a sub-route;
// pass the App directly when probes share the root.
func Mount(app fiber.Router, checks map[string]health.Pinger) {
	app.Get(health.PathLivez, LiveHandler())
	app.Get(health.PathReadyz, ReadyHandler(checks))
}
