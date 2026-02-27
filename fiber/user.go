package fiberutil

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	apperrors "github.com/samsonnaze5/aeternixth-go-lib/errors"
	"github.com/samsonnaze5/aeternixth-go-lib/middleware"
)

// GetUser retrieves the authenticated user information from a Fiber context.
//
// It reads the "user" value stored in [fiber.Ctx.Locals] by [middleware.JWTMiddleware]
// (or [middleware.OptionalJWTMiddleware]) and returns it as a [middleware.UserInfo] pointer.
//
// The function returns an [apperrors.AppError] in the following cases:
//   - HTTP 401 Unauthorized — if no "user" value is present in context (user not authenticated).
//   - HTTP 500 Internal Server Error — if the stored value cannot be type-asserted to [middleware.UserInfo].
//
// This function is intended for use in handlers protected by [middleware.JWTMiddleware].
// For handlers using [middleware.OptionalJWTMiddleware], check the returned error to determine
// whether a user is authenticated.
//
// Example:
//
//	app.Get("/profile", jwtMiddleware, func(c *fiber.Ctx) error {
//	    user, err := fiberutil.GetUser(c)
//	    if err != nil {
//	        return err
//	    }
//	    return c.JSON(user)
//	})
func GetUser(c *fiber.Ctx) (*middleware.UserInfo, error) {
	user := c.Locals("user")
	if user == nil {
		return nil, apperrors.NewUnauthorized("User not authenticated")
	}

	userInfo, ok := user.(middleware.UserInfo)
	if !ok {
		return nil, apperrors.NewInternalServerError("Invalid user context")
	}

	return &userInfo, nil
}

// GetUserID retrieves the authenticated user's UUID from a Fiber context.
//
// It is a convenience wrapper around [GetUser] that extracts only the UserID field.
// If the user is not authenticated or the context is invalid, it returns [uuid.Nil]
// instead of an error — making it safe to use in non-critical paths where the
// caller does not need to distinguish between error cases.
//
// For handlers that must enforce authentication, prefer [GetUser] or [MustGetUserID] instead.
//
// Example:
//
//	userID := fiberutil.GetUserID(c)
//	if userID == uuid.Nil {
//	    // user is not authenticated
//	}
func GetUserID(c *fiber.Ctx) uuid.UUID {
	user, err := GetUser(c)
	if err != nil {
		return uuid.Nil
	}

	return user.UserID
}

// GetUsername retrieves the authenticated user's username from a Fiber context.
//
// It is a convenience wrapper around [GetUser] that extracts only the Username field.
// If the user is not authenticated or the context is invalid, it returns an empty string
// instead of an error — making it safe to use in non-critical paths.
//
// For handlers that must enforce authentication, prefer [GetUser] instead.
func GetUsername(c *fiber.Ctx) string {
	user, err := GetUser(c)
	if err != nil {
		return ""
	}

	return user.Username
}

// GetUserRole retrieves the authenticated user's role from a Fiber context.
//
// It is a convenience wrapper around [GetUser] that extracts only the Role field.
// If the user is not authenticated or the context is invalid, it returns an empty string
// instead of an error — making it safe to use in non-critical paths such as
// conditional UI rendering or optional audit logging.
//
// For handlers that must enforce authentication, prefer [GetUser] instead.
func GetUserRole(c *fiber.Ctx) string {
	user, err := GetUser(c)
	if err != nil {
		return ""
	}

	return user.Role
}

// MustGetUser retrieves the authenticated user from a Fiber context and panics if not found.
//
// This function behaves identically to [GetUser] but panics with the returned
// [apperrors.AppError] instead of returning it. Use this only in handlers where
// authentication is guaranteed by a preceding [middleware.JWTMiddleware] — in such
// cases a missing user indicates a programming error, not a client error.
//
// If there is any possibility that the user may not be authenticated (e.g., with
// [middleware.OptionalJWTMiddleware]), use [GetUser] instead and handle the error gracefully.
//
// The panic is recoverable by [middleware.RecoverMiddleware], which converts it into
// an HTTP 500 response.
func MustGetUser(c *fiber.Ctx) *middleware.UserInfo {
	user, err := GetUser(c)
	if err != nil {
		panic(err)
	}

	return user
}

// MustGetUserID retrieves the authenticated user's UUID from a Fiber context
// and panics if not found.
//
// This function is a convenience wrapper that combines [MustGetUser] with UserID extraction.
// It panics if the user is not authenticated or the context is invalid.
// Use this only in handlers protected by [middleware.JWTMiddleware].
//
// See [MustGetUser] for details on panic behavior and recovery.
func MustGetUserID(c *fiber.Ctx) uuid.UUID {
	user := MustGetUser(c)
	return user.UserID
}
