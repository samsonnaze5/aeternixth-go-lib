package fiberutil

import (
	"github.com/gofiber/fiber/v2"

	apperrors "github.com/samsonnaze5/aeternixth-go-lib/errors"
	customvalidator "github.com/samsonnaze5/aeternixth-go-lib/validator"
)

// GetQueryParams parses and validates URL query parameters from a Fiber context
// into a strongly-typed struct of type T.
//
// The type parameter T should be a struct with appropriate Fiber query tags
// (e.g., `query:"page"`) and validation tags supported by the validator package
// (e.g., `validate:"required,min=1"`).
//
// The function performs two steps:
//  1. Parses query string values into the struct using Fiber's [fiber.Ctx.QueryParser].
//     If parsing fails, it returns an [apperrors.AppError] with HTTP 400 Bad Request.
//  2. Validates the parsed struct using [customvalidator.Validate].
//     If validation fails, it returns an [apperrors.AppError] with a detailed
//     validation error message produced by [customvalidator.FormatValidationError].
//
// On success it returns the populated and validated struct of type T and a nil error.
//
// Example usage:
//
//	type ListParams struct {
//	    Page    int    `query:"page" validate:"required,min=1"`
//	    Limit   int    `query:"limit" validate:"required,min=1,max=100"`
//	    Search  string `query:"search"`
//	}
//
//	app.Get("/items", func(c *fiber.Ctx) error {
//	    params, err := fiberutil.GetQueryParams[ListParams](c)
//	    if err != nil {
//	        return err // returns 400 with parsing or validation error
//	    }
//	    // use params.Page, params.Limit, params.Search ...
//	})
func GetQueryParams[T any](c *fiber.Ctx) (T, error) {
	var q T
	if err := c.QueryParser(&q); err != nil {
		var zero T
		return zero, apperrors.NewBadRequest("Invalid query parameters")
	}
	if err := customvalidator.Validate(&q); err != nil {
		return q, apperrors.NewValidationError(customvalidator.FormatValidationError(err))
	}
	return q, nil
}
