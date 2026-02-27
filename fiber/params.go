package fiberutil

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	apperrors "github.com/samsonnaze5/aeternixth-go-lib/errors"
)

// GetParamsStringID extracts a required string route parameter from a Fiber context.
// It reads the parameter identified by key from the URL path and returns its value.
//
// If the parameter is empty or missing, it returns an [apperrors.AppError] with
// HTTP 400 Bad Request status indicating that the parameter is required.
//
// Example usage in a Fiber handler:
//
//	// Route: GET /users/:id
//	app.Get("/users/:id", func(c *fiber.Ctx) error {
//	    id, err := fiberutil.GetParamsStringID(c, "id")
//	    if err != nil {
//	        return err // returns 400: "id parameter is required"
//	    }
//	    // use id ...
//	})
func GetParamsStringID(c *fiber.Ctx, key string) (string, error) {
	paramId := c.Params(key)
	if paramId == "" {
		return "", apperrors.NewBadRequest(key + " parameter is required")
	}
	return paramId, nil
}

// GetParamsUUID extracts a required UUID route parameter from a Fiber context.
// It reads the parameter identified by key from the URL path, validates that it is
// a well-formed UUID (RFC 4122), and returns the parsed [uuid.UUID].
//
// The function returns an [apperrors.AppError] with HTTP 400 Bad Request status in
// the following cases:
//   - The parameter is empty or missing — error message: "<key> parameter is required"
//   - The parameter is not a valid UUID — error message: "<key> parameter is not a valid UUID"
//
// On success it returns the parsed UUID and a nil error.
//
// Example usage in a Fiber handler:
//
//	// Route: DELETE /orders/:orderID
//	app.Delete("/orders/:orderID", func(c *fiber.Ctx) error {
//	    orderID, err := fiberutil.GetParamsUUID(c, "orderID")
//	    if err != nil {
//	        return err // returns 400 with descriptive message
//	    }
//	    // use orderID (uuid.UUID) ...
//	})
func GetParamsUUID(c *fiber.Ctx, key string) (uuid.UUID, error) {
	paramId := c.Params(key)
	if paramId == "" {
		return uuid.Nil, apperrors.NewBadRequest(key + " parameter is required")
	}
	uid, err := uuid.Parse(paramId)
	if err != nil {
		return uuid.Nil, apperrors.NewBadRequest(key + " parameter is not a valid UUID")
	}
	return uid, nil
}
