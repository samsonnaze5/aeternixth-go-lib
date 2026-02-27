package fiberutil

import (
	"github.com/gofiber/fiber/v2"

	apperrors "github.com/samsonnaze5/aeternixth-go-lib/errors"
	customvalidator "github.com/samsonnaze5/aeternixth-go-lib/validator"
)

// GetRequestBody parses and validates the HTTP request body from a Fiber context
// into a strongly-typed struct of type T.
//
// The type parameter T should be a struct with appropriate JSON tags
// (e.g., `json:"name"`) and validation tags supported by the validator package
// (e.g., `validate:"required,email"`).
//
// The function performs two steps:
//  1. Parses the request body into the struct using Fiber's [fiber.Ctx.BodyParser].
//     It supports JSON, XML, and form data depending on the Content-Type header.
//     If parsing fails, it returns an [apperrors.AppError] with HTTP 400 Bad Request.
//  2. Validates the parsed struct using [customvalidator.Validate].
//     If validation fails, it returns an [apperrors.AppError] with a detailed
//     validation error message produced by [customvalidator.FormatValidationError].
//
// On success it returns the populated and validated struct of type T and a nil error.
//
// Example usage:
//
//	type CreateUserRequest struct {
//	    Name     string `json:"name" validate:"required,min=2"`
//	    Email    string `json:"email" validate:"required,email"`
//	    Password string `json:"password" validate:"required,min=8"`
//	}
//
//	app.Post("/users", func(c *fiber.Ctx) error {
//	    req, err := fiberutil.GetRequestBody[CreateUserRequest](c)
//	    if err != nil {
//	        return err // returns 400 with parsing or validation error
//	    }
//	    // use req.Name, req.Email, req.Password ...
//	})
func GetRequestBody[T any](c *fiber.Ctx) (T, error) {
	var req T
	if err := c.BodyParser(&req); err != nil {
		var zero T
		return zero, apperrors.NewBadRequest("Invalid request body")
	}
	if err := customvalidator.Validate(&req); err != nil {
		return req, apperrors.NewValidationError(customvalidator.FormatValidationError(err))
	}
	return req, nil
}
