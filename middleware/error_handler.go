package middleware

import (
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"

	apperrors "github.com/samsonnaze5/aeternixth-go-lib/errors"
)

// Logger defines the logging contract used by middleware to log errors.
// It is designed to be compatible with structured loggers (e.g., Zap, Logrus).
type Logger interface {
	Errorf(format string, args ...interface{})
	Warnf(format string, args ...interface{})
}

// ErrorHandler returns a global [fiber.ErrorHandler] that converts errors into
// standardized JSON error responses using the [apperrors.ErrorResponse] format.
//
// It handles three categories of errors:
//   - [apperrors.AppError] — application-level errors created by the apperrors package.
//     The HTTP status code, error code, message, and optional details are all taken from
//     the AppError fields.
//   - [fiber.Error] — Fiber's built-in HTTP errors (e.g., from c.Status(404).SendString).
//     The status code is taken from the Fiber error and mapped to an error code via
//     [getErrorCodeFromStatus].
//   - All other errors — treated as unexpected internal server errors. The error message
//     is logged server-side and a generic "An unexpected error occurred" message is returned
//     to the client without exposing internal error details.
//
// Server errors (status >= 500) are additionally logged with the HTTP method, path, and
// status code for observability.
//
// The response Content-Type is always set to application/json.
//
// Usage — assign as the Fiber app's error handler:
//
//	app := fiber.New(fiber.Config{
//	    ErrorHandler: middleware.ErrorHandler(logger),
//	})
func ErrorHandler(logger Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		// Default error response
		statusCode := fiber.StatusInternalServerError
		errorResponse := apperrors.ErrorResponse{
			Success: false,
			Error: &apperrors.ErrorInfo{
				Code:    apperrors.ErrCodeInternalServer,
				Message: "An unexpected error occurred",
			},
			Data: nil,
		}

		// Handle AppError
		if appErr, ok := err.(*apperrors.AppError); ok {
			// Resolve status: use AppError's own status if set (e.g., NewBadRequest),
			// otherwise look up from registry (e.g., New(code) from generated errors).
			if appErr.StatusCode != 0 {
				statusCode = appErr.StatusCode
			} else {
				statusCode = apperrors.GetStatusCode(appErr.Code)
			}

			// Resolve translated message based on Accept-Language header
			lang := apperrors.ResolveLanguage(c.Get("Accept-Language"))
			errorResponse.Error = &apperrors.ErrorInfo{
				Code:    appErr.Code,
				Message: apperrors.ResolveMessage(appErr, lang),
				Field:   appErr.Field,
				Details: appErr.Details,
			}
		} else if fiberErr, ok := err.(*fiber.Error); ok {
			// Handle Fiber's built-in errors
			statusCode = fiberErr.Code
			errorResponse.Error = &apperrors.ErrorInfo{
				Code:    getErrorCodeFromStatus(statusCode),
				Message: fiberErr.Message,
			}
		} else {
			// Handle unexpected errors — log raw error but do NOT expose to client
			if logger != nil {
				logger.Errorf("Unexpected error: %v", err)
			} else {
				log.Printf("Unexpected error: %v", err)
			}
		}

		// Log errors (except client errors)
		if statusCode >= 500 {
			if logger != nil {
				logger.Errorf("%s %s - Status: %d, Error: %v", c.Method(), c.Path(), statusCode, err)
			} else {
				log.Printf("[ERROR] %s %s - Status: %d, Error: %v", c.Method(), c.Path(), statusCode, err)
			}
		} else if statusCode >= 400 && logger != nil {
			logger.Warnf("%s %s - Status: %d, Error: %v", c.Method(), c.Path(), statusCode, err)
		}

		// Set response headers
		c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

		// Return standardized error response
		return c.Status(statusCode).JSON(errorResponse)
	}
}

// getErrorCodeFromStatus maps an HTTP status code to the corresponding
// application error code string defined in the apperrors package.
//
// Supported mappings:
//   - 400 -> [apperrors.ErrCodeBadRequest]
//   - 401 -> [apperrors.ErrCodeUnauthorized]
//   - 403 -> [apperrors.ErrCodeForbidden]
//   - 404 -> [apperrors.ErrCodeNotFound]
//   - 409 -> [apperrors.ErrCodeConflict]
//   - 500 -> [apperrors.ErrCodeInternalServer]
//   - 503 -> [apperrors.ErrCodeServiceUnavailable]
//
// Any unrecognized status code defaults to [apperrors.ErrCodeInternalServer].
func getErrorCodeFromStatus(statusCode int) string {
	switch statusCode {
	case fiber.StatusBadRequest:
		return apperrors.ErrCodeBadRequest
	case fiber.StatusUnauthorized:
		return apperrors.ErrCodeUnauthorized
	case fiber.StatusForbidden:
		return apperrors.ErrCodeForbidden
	case fiber.StatusNotFound:
		return apperrors.ErrCodeNotFound
	case fiber.StatusConflict:
		return apperrors.ErrCodeConflict
	case fiber.StatusInternalServerError:
		return apperrors.ErrCodeInternalServer
	case fiber.StatusServiceUnavailable:
		return apperrors.ErrCodeServiceUnavailable
	default:
		return apperrors.ErrCodeInternalServer
	}
}

// RecoverMiddleware returns a Fiber middleware that recovers from panics occurring
// in downstream handlers and converts them into HTTP 500 Internal Server Error responses.
//
// When a panic is caught the middleware:
//  1. Converts the recovered value to an error (supports error, string, and any types).
//  2. Logs the panic with the HTTP method, path, and error message for debugging.
//  3. Responds with a standardized [apperrors.ErrorResponse] JSON body containing
//     a generic "A critical error occurred" message — the original panic value is never
//     exposed to the client.
//
// This middleware should be registered early in the middleware chain so that it can
// catch panics from all subsequent handlers and middleware.
//
// Example:
//
//	app := fiber.New(fiber.Config{
//	    ErrorHandler: middleware.ErrorHandler(logger),
//	})
//	app.Use(middleware.RecoverMiddleware())
func RecoverMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				var err error
				switch v := r.(type) {
				case error:
					err = v
				case string:
					err = fmt.Errorf("%s", v)
				default:
					err = fmt.Errorf("%v", v)
				}

				log.Printf("[PANIC RECOVERED] %s %s - Error: %v",
					c.Method(),
					c.Path(),
					err,
				)

				// Create internal server error
				appErr := apperrors.NewInternalServerError("A critical error occurred")
				c.Status(appErr.StatusCode).JSON(apperrors.ErrorResponse{
					Success: false,
					Error: &apperrors.ErrorInfo{
						Code:    appErr.Code,
						Message: appErr.Message,
					},
					Data: nil,
				})
			}
		}()

		return c.Next()
	}
}

// NotFoundHandler returns a Fiber handler that responds with an HTTP 404 Not Found
// error for any request that reaches it. The error message includes the requested path
// for easier debugging (e.g., "Route '/api/unknown' not found").
//
// Register this handler as the last route in the application so that it acts as a
// catch-all for undefined routes. The returned [apperrors.AppError] is handled by
// [ErrorHandler] which formats it into a standardized JSON response.
//
// Example:
//
//	app := fiber.New(fiber.Config{
//	    ErrorHandler: middleware.ErrorHandler(logger),
//	})
//	// ... register all routes ...
//	app.Use(middleware.NotFoundHandler()) // catch-all at the end
func NotFoundHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return apperrors.NewNotFound(fmt.Sprintf("Route '%s' not found", c.Path()))
	}
}
