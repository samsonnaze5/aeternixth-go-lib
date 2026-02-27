package middleware

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	apperrors "github.com/samsonnaze5/aeternixth-go-lib/errors"
	jwtutil "github.com/samsonnaze5/aeternixth-go-lib/jwt"
)

// Claims represents the JWT claims payload for user authentication in this application.
// It embeds [jwt.RegisteredClaims] to include standard JWT fields (exp, iat, nbf) and
// adds application-specific fields for user identification and authorization.
//
// Claims implements [jwt.Claims] and can be used as the type parameter for
// [jwtutil.JWTService] to create a type-safe JWT service:
//
//	jwtSvc := jwtutil.NewJWTService[*Claims](secretKey, middleware.NewEmptyClaims)
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// NewEmptyClaims is a factory function that returns a new zero-value [Claims] pointer.
// It is intended to be passed as the newClaims argument when constructing a
// [jwtutil.JWTService], allowing the JWT service to create fresh Claims instances
// for token parsing.
//
// Example:
//
//	jwtSvc := jwtutil.NewJWTService[*Claims](secretKey, middleware.NewEmptyClaims)
func NewEmptyClaims() *Claims {
	return &Claims{}
}

// NewClaims creates a fully populated [Claims] instance with the given user fields
// and token expiry duration. The registered claims (ExpiresAt, IssuedAt, NotBefore)
// are all set relative to the current time.
//
// Parameters:
//   - userID: the unique identifier of the user (typically a UUID string).
//   - username: the user's display name or login name.
//   - email: the user's email address.
//   - role: the user's authorization role (e.g., "admin", "user").
//   - expiry: how long the token should be valid from now.
//
// Example:
//
//	claims := middleware.NewClaims(
//	    user.ID.String(), user.Username, user.Email, "admin",
//	    24*time.Hour,
//	)
//	token, err := jwtSvc.GenerateToken(claims)
func NewClaims(userID, username, email, role string, expiry time.Duration) *Claims {
	now := time.Now()
	return &Claims{
		UserID:   userID,
		Username: username,
		Email:    email,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
}

// UserInfo represents the authenticated user information extracted from a validated
// JWT token and stored in [fiber.Ctx.Locals] under the key "user".
//
// It is set by [JWTMiddleware] (and [OptionalJWTMiddleware] when a valid token is present)
// and can be retrieved in downstream handlers using [fiberutil.GetUser] or
// the convenience helpers [fiberutil.GetUserID], [fiberutil.GetUsername], and
// [fiberutil.GetUserRole].
//
// Unlike [Claims], UserInfo uses [uuid.UUID] for the UserID field (parsed from
// the string-based Claims.UserID) so that downstream code can work with typed UUIDs
// directly.
type UserInfo struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
}

// JWTMiddleware returns a Fiber middleware that enforces JWT Bearer token authentication.
//
// For every incoming request the middleware:
//  1. Reads the "Authorization" header and expects the format "Bearer <token>".
//     Returns HTTP 401 if the header is missing or malformed.
//  2. Validates the token using the provided [jwtutil.JWTService].
//     Returns HTTP 401 if the token is expired ([jwtutil.ErrExpiredToken]) or otherwise invalid.
//  3. Parses the UserID claim as a [uuid.UUID].
//     Returns HTTP 401 if the UserID is not a valid UUID.
//  4. Stores a [UserInfo] struct in [fiber.Ctx.Locals] under the key "user" so that
//     downstream handlers can retrieve it via [fiberutil.GetUser] and related helpers.
//  5. Calls c.Next() to continue to the next handler.
//
// This middleware should be used on routes that require authentication. For routes
// where authentication is optional, use [OptionalJWTMiddleware] instead.
//
// Example:
//
//	jwtSvc := jwtutil.NewJWTService[*middleware.Claims](secretKey, middleware.NewEmptyClaims)
//	authRequired := middleware.JWTMiddleware(jwtSvc)
//
//	app.Get("/profile", authRequired, profileHandler)
//	app.Put("/profile", authRequired, updateProfileHandler)
func JWTMiddleware(jwtService *jwtutil.JWTService[*Claims]) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return apperrors.NewUnauthorized("Missing authorization header")
		}

		// Check if it's a Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			return apperrors.NewUnauthorized("Invalid authorization header format")
		}

		tokenString := parts[1]

		// Validate token
		claims, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			if err == jwtutil.ErrExpiredToken {
				return apperrors.NewUnauthorized("Token has expired")
			}
			return apperrors.NewUnauthorized("Invalid token")
		}

		// Parse user ID
		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			return apperrors.NewUnauthorized("Invalid user ID in token")
		}

		// Store user info in context
		userInfo := UserInfo{
			UserID:   userID,
			Username: claims.Username,
			Email:    claims.Email,
			Role:     claims.Role,
		}

		c.Locals("user", userInfo)

		// Continue to next handler
		return c.Next()
	}
}

// OptionalJWTMiddleware returns a Fiber middleware that performs JWT Bearer token
// authentication when a token is present, but does not reject requests without one.
//
// This is useful for endpoints that behave differently for authenticated vs. anonymous
// users — for example, a public product listing that shows personalized recommendations
// when logged in.
//
// Behavior:
//   - If the "Authorization" header is missing, the request continues without setting user info.
//   - If the header is present but malformed (not "Bearer <token>"), the request continues
//     without setting user info — it does NOT return an error.
//   - If the token is present but invalid or expired, the request continues without user info.
//   - If the token is valid but the UserID cannot be parsed as a UUID, the request continues
//     without user info.
//   - If the token is valid, a [UserInfo] struct is stored in [fiber.Ctx.Locals] under the
//     key "user", identical to [JWTMiddleware].
//
// Downstream handlers can check whether a user is authenticated by calling
// [fiberutil.GetUser] and inspecting the returned error, or by checking if
// [fiberutil.GetUserID] returns [uuid.Nil].
//
// Example:
//
//	jwtSvc := jwtutil.NewJWTService[*middleware.Claims](secretKey, middleware.NewEmptyClaims)
//	optionalAuth := middleware.OptionalJWTMiddleware(jwtSvc)
//
//	app.Get("/products", optionalAuth, func(c *fiber.Ctx) error {
//	    user, err := fiberutil.GetUser(c)
//	    if err == nil {
//	        // authenticated — show personalized content
//	    }
//	    // ... public content ...
//	})
func OptionalJWTMiddleware(jwtService *jwtutil.JWTService[*Claims]) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			// No token provided, continue without user info
			return c.Next()
		}

		// Check if it's a Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			// Invalid format, but don't fail - just continue
			return c.Next()
		}

		tokenString := parts[1]

		// Validate token
		claims, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			// Invalid token, but don't fail - just continue
			return c.Next()
		}

		// Parse user ID
		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			// Invalid user ID, but don't fail - just continue
			return c.Next()
		}

		// Store user info in context
		userInfo := UserInfo{
			UserID:   userID,
			Username: claims.Username,
			Email:    claims.Email,
			Role:     claims.Role,
		}

		c.Locals("user", userInfo)

		// Continue to next handler
		return c.Next()
	}
}
