// Package errors provides a standardized error handling system for the
// application. It defines AppError — a structured error type that carries
// an error code, human-readable message, optional details, and an HTTP
// status code. This allows consistent error responses across all API
// endpoints.
//
// The package also provides convenient constructor functions for common
// HTTP error types (BadRequest, NotFound, etc.) and a set of predefined
// error code constants that can be used by clients for programmatic
// error handling.
package errors

import (
	"fmt"
	"net/http"
	"strings"
)

// AppError represents a standardized application error that carries both
// machine-readable and human-readable information. It implements the
// built-in error interface.
//
// Fields:
//   - Code:       A machine-readable error code string (e.g., "NOT_FOUND",
//     "VALIDATION_FAILED") that clients can use for programmatic
//     error handling.
//   - Message:    A human-readable error message describing what went wrong,
//     suitable for displaying to end users.
//   - Details:    Optional additional data providing more context about the
//     error (e.g., a list of validation errors). Omitted from JSON
//     when nil.
//   - StatusCode: The HTTP status code associated with this error (e.g., 404).
//     Excluded from JSON serialization (json:"-") because it is
//     used to set the HTTP response status, not included in the body.
type AppError struct {
	Code       string      `json:"code"`
	Message    string      `json:"message"`
	Field      string      `json:"field,omitempty"`
	Details    interface{} `json:"details,omitempty"`
	StatusCode int         `json:"-"`
}

// Error implements the built-in error interface. It returns a formatted
// string in the format "[CODE] message", which is useful for logging
// and error wrapping.
//
// Example output: "[NOT_FOUND] User with ID 123 was not found"
func (e *AppError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// ErrorResponse represents the standardized JSON error response envelope
// returned by all API endpoints. It wraps the error information inside a
// consistent structure with a success flag.
//
// JSON structure:
//
//	{
//	    "success": false,
//	    "error": { "code": "...", "message": "...", "details": ... },
//	    "data": null
//	}
type ErrorResponse struct {
	Success bool        `json:"success"`
	Error   *ErrorInfo  `json:"error,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// ErrorInfo contains the detailed error information nested inside an
// ErrorResponse. It is separated from AppError to provide a clean
// JSON structure without the HTTP status code.
type ErrorInfo struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Field   string      `json:"field,omitempty"`
	Details interface{} `json:"details,omitempty"`
}

// Common error code constants used throughout the application. These provide
// a consistent vocabulary for error types that clients can match against
// for programmatic error handling (e.g., showing different UI based on
// the error code).
const (
	// Client errors (4xx) — problems caused by the client's request.

	// ErrCodeBadRequest indicates the request was malformed or invalid.
	ErrCodeBadRequest = "BAD_REQUEST"
	// ErrCodeUnauthorized indicates the request lacks valid authentication.
	ErrCodeUnauthorized = "UNAUTHORIZED"
	// ErrCodeForbidden indicates the authenticated user lacks permission.
	ErrCodeForbidden = "FORBIDDEN"
	// ErrCodeNotFound indicates the requested resource does not exist.
	ErrCodeNotFound = "NOT_FOUND"
	// ErrCodeConflict indicates a conflict with the current resource state (e.g., duplicate).
	ErrCodeConflict = "CONFLICT"
	// ErrCodeValidationFailed indicates one or more fields failed validation.
	ErrCodeValidationFailed = "VALIDATION_FAILED"
	// ErrCodeInvalidCredentials indicates the provided login credentials are wrong.
	ErrCodeInvalidCredentials = "INVALID_CREDENTIALS"
	// ErrCodeTokenExpired indicates the authentication token has expired.
	ErrCodeTokenExpired = "TOKEN_EXPIRED"
	// ErrCodeInvalidToken indicates the authentication token is malformed or invalid.
	ErrCodeInvalidToken = "INVALID_TOKEN"
	// ErrCodeAccountInactive indicates the user's account is deactivated or suspended.
	ErrCodeAccountInactive = "ACCOUNT_INACTIVE"
	// ErrCodeResourceExists indicates the resource already exists (duplicate creation).
	ErrCodeResourceExists = "RESOURCE_EXISTS"
	// ErrCodeInvalidInput indicates the input data is semantically invalid.
	ErrCodeInvalidInput = "INVALID_INPUT"

	// Server errors (5xx) — problems on the server side.

	// ErrCodeInternalServer indicates an unexpected internal server error.
	ErrCodeInternalServer = "INTERNAL_SERVER_ERROR"
	// ErrCodeServiceUnavailable indicates the service is temporarily unavailable.
	ErrCodeServiceUnavailable = "SERVICE_UNAVAILABLE"
	// ErrCodeDatabaseError indicates a database operation failed.
	ErrCodeDatabaseError = "DATABASE_ERROR"
	// ErrCodeExternalServiceError indicates a call to an external service failed.
	ErrCodeExternalServiceError = "EXTERNAL_SERVICE_ERROR"
)

// registry maps error codes to HTTP status codes. Populated at startup
// via RegisterCodes so that New() can resolve status automatically.
var registry = map[string]int{}

// RegisterCodes registers error codes with their HTTP status codes.
// Must be called during application startup before any errors are returned.
//
// Example:
//
//	errors.RegisterCodes(map[string]int{
//	    "AUTH_LOGIN_INVALID_CREDENTIALS": 401,
//	    "IB_WALLET_NOT_FOUND":           404,
//	})
func RegisterCodes(codes map[string]int) {
	for k, v := range codes {
		registry[k] = v
	}
}

// New creates an AppError using the registry for HTTP status lookup.
// The code must have been registered via RegisterCodes; if not found,
// the status defaults to 500 (Internal Server Error).
//
// Example:
//
//	var ErrInvalidCredentials = errors.New("AUTH_LOGIN_INVALID_CREDENTIALS", "INVALID_EMAIL_OR_PASSWORD")
func New(code string, message string) *AppError {
	status, ok := registry[code]
	if !ok {
		status = 500
	}
	return &AppError{Code: code, Message: message, StatusCode: status}
}

// NewAppError creates a new AppError with the given error code, human-readable
// message, and HTTP status code. This is the base constructor — use the
// convenience functions (NewBadRequest, NewNotFound, etc.) for common cases.
//
// Example:
//
//	err := errors.NewAppError("RATE_LIMITED", "Too many requests", http.StatusTooManyRequests)
func NewAppError(code string, message string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
	}
}

// WithDetails returns a copy of the AppError with the given details attached.
// Safe for use with package-level error variables (does not mutate the original).
// The details can be any serializable value (e.g., a map of field errors).
//
// If the message contains {key} placeholders and details is a map, the
// placeholders are automatically replaced with values from the map.
//
// Example:
//
//	err := ErrOTPRateLimit.WithDetails(map[string]interface{}{
//	    "retry_after_seconds": 45,
//	})
//	// message: "Please wait before requesting a new OTP (45 Second)."
func (e *AppError) WithDetails(details interface{}) *AppError {
	msg := e.Message
	if m, ok := details.(map[string]interface{}); ok {
		for k, v := range m {
			msg = strings.ReplaceAll(msg, "{"+k+"}", fmt.Sprintf("%v", v))
		}
	}
	return &AppError{
		Code:       e.Code,
		Message:    msg,
		Field:      e.Field,
		Details:    details,
		StatusCode: e.StatusCode,
	}
}

// WithField returns a copy of the AppError with the field name set.
// Safe for use with package-level error variables (does not mutate the original).
// The field indicates which request field caused the error (e.g., "email", "password").
//
// Example:
//
//	return ErrInvalidCredentials.WithField("email")
func (e *AppError) WithField(field string) *AppError {
	return &AppError{
		Code:       e.Code,
		Message:    e.Message,
		Field:      field,
		Details:    e.Details,
		StatusCode: e.StatusCode,
	}
}

// NewBadRequest creates an AppError with code "BAD_REQUEST" and HTTP status 400.
// Use this when the client's request is malformed, has missing required fields,
// or is otherwise structurally invalid.
//
// Example:
//
//	return errors.NewBadRequest("request body is required")
func NewBadRequest(message string) *AppError {
	return NewAppError(ErrCodeBadRequest, message, http.StatusBadRequest)
}

// NewUnauthorized creates an AppError with code "UNAUTHORIZED" and HTTP status 401.
// Use this when the request lacks valid authentication credentials (missing or
// invalid token, expired session, etc.).
//
// Example:
//
//	return errors.NewUnauthorized("authentication token is missing")
func NewUnauthorized(message string) *AppError {
	return NewAppError(ErrCodeUnauthorized, message, http.StatusUnauthorized)
}

// NewForbidden creates an AppError with code "FORBIDDEN" and HTTP status 403.
// Use this when the authenticated user does not have permission to perform
// the requested action. Unlike 401, the user's identity is known — they
// simply lack the required authorization.
//
// Example:
//
//	return errors.NewForbidden("you do not have permission to delete this resource")
func NewForbidden(message string) *AppError {
	return NewAppError(ErrCodeForbidden, message, http.StatusForbidden)
}

// NewNotFound creates an AppError with code "NOT_FOUND" and HTTP status 404.
// Use this when the requested resource does not exist in the system.
//
// Example:
//
//	return errors.NewNotFound("user with ID 123 was not found")
func NewNotFound(message string) *AppError {
	return NewAppError(ErrCodeNotFound, message, http.StatusNotFound)
}

// NewConflict creates an AppError with code "CONFLICT" and HTTP status 409.
// Use this when the request conflicts with the current state of the resource,
// such as trying to create a resource that already exists or performing a
// concurrent modification.
//
// Example:
//
//	return errors.NewConflict("a user with this email already exists")
func NewConflict(message string) *AppError {
	return NewAppError(ErrCodeConflict, message, http.StatusConflict)
}

// NewInternalServerError creates an AppError with code "INTERNAL_SERVER_ERROR"
// and HTTP status 500. Use this for unexpected server-side errors that are
// not caused by the client's request.
//
// Example:
//
//	return errors.NewInternalServerError("failed to process the request")
func NewInternalServerError(message string) *AppError {
	return NewAppError(ErrCodeInternalServer, message, http.StatusInternalServerError)
}

// NewValidationError creates an AppError with code "VALIDATION_FAILED" and
// HTTP status 400, pre-populated with a generic message and the provided
// details. The details parameter typically contains a map or slice describing
// which fields failed validation and why.
//
// Example:
//
//	details := map[string]string{
//	    "email": "must be a valid email",
//	    "age":   "must be at least 18",
//	}
//	return errors.NewValidationError(details)
func NewValidationError(details interface{}) *AppError {
	return NewAppError(
		ErrCodeValidationFailed,
		"Validation failed for the provided data",
		http.StatusBadRequest,
	).WithDetails(details)
}

// ToHTTPStatusCode extracts the HTTP status code from an error. If the error
// is an *AppError, it returns the associated StatusCode. For any other error
// type, it defaults to 500 (Internal Server Error).
//
// This is useful in middleware or error-handling layers that need to determine
// the appropriate HTTP response status from a generic error interface.
//
// Example:
//
//	statusCode := errors.ToHTTPStatusCode(err)
//	c.Status(statusCode).JSON(errorResponse)
func ToHTTPStatusCode(err error) int {
	if appErr, ok := err.(*AppError); ok {
		return appErr.StatusCode
	}
	return http.StatusInternalServerError
}
