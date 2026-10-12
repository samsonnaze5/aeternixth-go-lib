package otelfiber_test

import (
	"github.com/gofiber/fiber/v2"

	"github.com/samsonnaze5/aeternixth-go-lib/observability/otel/otelfiber"
)

func ExampleMiddleware() {
	app := fiber.New()
	app.Use(otelfiber.Middleware()) // first, so every later handler sees the span
	app.Get("/users/:id", func(c *fiber.Ctx) error {
		ctx := c.UserContext() // carries the server span — pass it down
		_ = ctx
		return c.SendStatus(fiber.StatusOK)
	})
}
