// Package validator provides a thin wrapper around the go-playground/validator
// library for struct validation. It initializes a singleton validator instance
// at package load time and exposes simple functions to validate structs and
// format validation errors into human-readable messages.
package validator

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// validate is the package-level singleton validator instance. It is initialized
// once in init() and reused across all Validate() calls for performance,
// since creating a new validator on every call would be wasteful.
var validate *validator.Validate

// init initializes the singleton validator instance when the package is first
// imported. This ensures the validator is ready to use without any explicit
// setup by the caller. The go-playground/validator library is safe for
// concurrent use after initialization.
func init() {
	validate = validator.New()
}

// Validate checks a struct against its validation tags (e.g., `validate:"required,email"`)
// and returns an error if any field fails validation. The input must be a struct
// or a pointer to a struct; passing other types will cause the underlying
// validator to return an InvalidValidationError.
//
// This function delegates to validator.Validate.Struct() from the
// go-playground/validator library. The returned error can be type-asserted
// to validator.ValidationErrors for detailed per-field error inspection,
// or passed to FormatValidationError() for a human-readable summary.
//
// Example:
//
//	type CreateUserRequest struct {
//	    Email string `validate:"required,email"`
//	    Name  string `validate:"required,min=2"`
//	}
//
//	err := validator.Validate(req)
//	if err != nil {
//	    msg := validator.FormatValidationError(err)
//	    // msg: "Email must be a valid email address; Name must be at least 2 characters"
//	}
func Validate(data interface{}) error {
	return validate.Struct(data)
}

// FormatValidationError converts a validation error into a single
// human-readable string. If the error is a validator.ValidationErrors
// (as returned by Validate), each field error is formatted individually
// and joined with "; ". If the error is not a ValidationErrors type,
// the raw err.Error() string is returned as-is.
//
// Example output: "Email is required; Password must be at least 8 characters"
func FormatValidationError(err error) string {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		var messages []string
		for _, e := range validationErrors {
			messages = append(messages, formatFieldError(e))
		}
		return strings.Join(messages, "; ")
	}
	return err.Error()
}

// formatFieldError converts a single validator.FieldError into a human-readable
// message string. It maps common validation tags to descriptive messages:
//
//   - "required" -> "{Field} is required"
//   - "email"    -> "{Field} must be a valid email address"
//   - "min"      -> "{Field} must be at least {Param} characters"
//   - "max"      -> "{Field} must not exceed {Param} characters"
//   - (default)  -> "{Field} is invalid"
//
// This function is called internally by FormatValidationError for each
// individual field error.
func formatFieldError(e validator.FieldError) string {
	field := e.Field()

	switch e.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", field, e.Param())
	case "max":
		return fmt.Sprintf("%s must not exceed %s characters", field, e.Param())
	default:
		return fmt.Sprintf("%s is invalid", field)
	}
}
