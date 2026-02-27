// Package response provides standardized HTTP response helpers for the
// Fiber web framework. It ensures every API endpoint returns responses
// in a consistent JSON format with a "success" flag, optional error
// information, and the response data.
//
// Success responses follow this structure:
//
//	{ "success": true, "data": { ... } }
//
// Error responses follow this structure:
//
//	{ "success": false, "error": { "code": "...", "message": "..." }, "data": null }
//
// All error helper functions (BadRequest, NotFound, etc.) delegate to the
// errors package for constructing AppError instances, ensuring consistent
// error codes and status codes across the application.
package response

import (
	"github.com/gofiber/fiber/v2"

	apperrors "github.com/samsonnaze5/aeternixth-go-lib/errors"
)

// SuccessResponse represents the standardized JSON envelope for successful
// API responses. The Error field is always nil/omitted for success responses.
//
// JSON structure:
//
//	{ "success": true, "data": { ... } }
type SuccessResponse struct {
	Success bool        `json:"success"`
	Error   interface{} `json:"error,omitempty"`
	Data    interface{} `json:"data"`
}

// Success sends a standardized 200 OK JSON response with the provided data.
// The response body will have "success": true and the data object serialized
// under the "data" key.
//
// Example:
//
//	return response.Success(c, fiber.Map{"user": user})
//	// Response: { "success": true, "data": { "user": { ... } } }
func Success(c *fiber.Ctx, data interface{}) error {
	return c.JSON(SuccessResponse{
		Success: true,
		Error:   nil,
		Data:    data,
	})
}

// SuccessWithStatus sends a standardized success JSON response with a custom
// HTTP status code. Use this when the default 200 OK is not appropriate
// (e.g., 201 Created after resource creation).
//
// Example:
//
//	return response.SuccessWithStatus(c, fiber.StatusCreated, newUser)
func SuccessWithStatus(c *fiber.Ctx, statusCode int, data interface{}) error {
	return c.Status(statusCode).JSON(SuccessResponse{
		Success: true,
		Error:   nil,
		Data:    data,
	})
}

// Created sends a standardized 201 Created JSON response. This is a
// convenience wrapper around SuccessWithStatus for the common pattern
// of returning a newly created resource.
//
// Example:
//
//	return response.Created(c, newUser)
func Created(c *fiber.Ctx, data interface{}) error {
	return SuccessWithStatus(c, fiber.StatusCreated, data)
}

// NoContent sends a 204 No Content response with no body. This is typically
// used after a successful DELETE operation or any operation where there is
// no meaningful data to return.
//
// Example:
//
//	return response.NoContent(c)
func NoContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

// Error sends a standardized error JSON response. It inspects the error type:
//
//   - If the error is an *apperrors.AppError, it uses the error's StatusCode,
//     Code, Message, and Details to build a structured error response.
//   - If the error is any other type, it returns a 500 Internal Server Error
//     with the error's message string as the error message.
//
// This is the core error response function — all other error helpers
// (BadRequest, NotFound, etc.) delegate to this function.
//
// Example:
//
//	return response.Error(c, apperrors.NewNotFound("user not found"))
//	// Response (404): { "success": false, "error": { "code": "NOT_FOUND", "message": "user not found" } }
func Error(c *fiber.Ctx, err error) error {
	if appErr, ok := err.(*apperrors.AppError); ok {
		return c.Status(appErr.StatusCode).JSON(apperrors.ErrorResponse{
			Success: false,
			Error: &apperrors.ErrorInfo{
				Code:    appErr.Code,
				Message: appErr.Message,
				Details: appErr.Details,
			},
			Data: nil,
		})
	}

	// For non-AppError errors, return as internal server error
	return c.Status(fiber.StatusInternalServerError).JSON(apperrors.ErrorResponse{
		Success: false,
		Error: &apperrors.ErrorInfo{
			Code:    apperrors.ErrCodeInternalServer,
			Message: err.Error(),
		},
		Data: nil,
	})
}

// BadRequest sends a 400 Bad Request error response with the given message.
// Use this when the client's request is malformed or contains invalid data.
//
// Example:
//
//	return response.BadRequest(c, "invalid JSON body")
func BadRequest(c *fiber.Ctx, message string) error {
	return Error(c, apperrors.NewBadRequest(message))
}

// Unauthorized sends a 401 Unauthorized error response with the given message.
// Use this when authentication is required but missing or invalid.
//
// Example:
//
//	return response.Unauthorized(c, "invalid or expired token")
func Unauthorized(c *fiber.Ctx, message string) error {
	return Error(c, apperrors.NewUnauthorized(message))
}

// Forbidden sends a 403 Forbidden error response with the given message.
// Use this when the authenticated user lacks permission for the action.
//
// Example:
//
//	return response.Forbidden(c, "admin access required")
func Forbidden(c *fiber.Ctx, message string) error {
	return Error(c, apperrors.NewForbidden(message))
}

// NotFound sends a 404 Not Found error response with the given message.
// Use this when the requested resource does not exist.
//
// Example:
//
//	return response.NotFound(c, "user not found")
func NotFound(c *fiber.Ctx, message string) error {
	return Error(c, apperrors.NewNotFound(message))
}

// Conflict sends a 409 Conflict error response with the given message.
// Use this when the request conflicts with existing resource state
// (e.g., duplicate email on registration).
//
// Example:
//
//	return response.Conflict(c, "email already registered")
func Conflict(c *fiber.Ctx, message string) error {
	return Error(c, apperrors.NewConflict(message))
}

// InternalServerError sends a 500 Internal Server Error response with the
// given message. Use this for unexpected server-side errors.
//
// Example:
//
//	return response.InternalServerError(c, "failed to process request")
func InternalServerError(c *fiber.Ctx, message string) error {
	return Error(c, apperrors.NewInternalServerError(message))
}

// ValidationError sends a 400 Bad Request error response with the code
// "VALIDATION_FAILED" and the provided details. The details parameter
// typically contains a map or list of field-level validation errors.
//
// Example:
//
//	return response.ValidationError(c, map[string]string{
//	    "email": "must be a valid email",
//	    "name":  "is required",
//	})
func ValidationError(c *fiber.Ctx, details interface{}) error {
	return Error(c, apperrors.NewValidationError(details))
}
