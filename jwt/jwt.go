// Package jwtutil provides a generic, type-safe JWT (JSON Web Token) service
// for generating and validating tokens using HMAC-SHA256 signing. It wraps
// the golang-jwt/jwt/v5 library and uses Go generics so that the caller's
// custom claims type is preserved throughout — no type assertions needed.
//
// Example:
//
//	type MyClaims struct {
//	    UserID string `json:"user_id"`
//	    jwt.RegisteredClaims
//	}
//
//	svc := jwtutil.NewJWTService("secret", func() *MyClaims { return &MyClaims{} })
//	token, _ := svc.GenerateToken(&MyClaims{
//	    UserID: "abc",
//	    RegisteredClaims: jwt.RegisteredClaims{
//	        ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
//	    },
//	})
//	claims, err := svc.ValidateToken(token)
package jwtutil

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrInvalidToken is returned when the token cannot be parsed, has an
	// unsupported signing method, or fails signature verification. Callers
	// can use errors.Is(err, jwtutil.ErrInvalidToken) for matching.
	ErrInvalidToken = errors.New("invalid token")

	// ErrExpiredToken is returned when the token's "exp" claim has passed.
	// This is a specific subcase of validation failure — the token was
	// structurally valid but is no longer within its validity window.
	ErrExpiredToken = errors.New("token has expired")
)

// JWTService is a generic JWT service that handles token generation and
// validation for a specific claims type T. The type parameter T must
// satisfy the jwt.Claims interface, which is typically achieved by
// embedding jwt.RegisteredClaims in a custom struct.
//
// JWTService uses HMAC-SHA256 (HS256) as the signing method. The secret
// key is stored in memory and used for both signing (GenerateToken) and
// verification (ValidateToken).
//
// Thread safety: JWTService is safe for concurrent use because it only
// reads its secretKey field after construction.
type JWTService[T jwt.Claims] struct {
	secretKey string
	newClaims func() T
}

// NewJWTService creates a new JWTService instance configured with the given
// HMAC secret key and a claims factory function.
//
// Parameters:
//   - secretKey: The HMAC-SHA256 secret key used for signing and verifying
//     tokens. This should be a strong, randomly generated string
//     (at least 32 bytes recommended).
//   - newClaims: A factory function that returns a new zero-value instance
//     of the claims type T. This is needed by the JWT parser to
//     allocate the correct concrete type during token validation.
//     Example: func() *MyClaims { return &MyClaims{} }
//
// Example:
//
//	svc := jwtutil.NewJWTService("my-secret-key", func() *MyClaims {
//	    return &MyClaims{}
//	})
func NewJWTService[T jwt.Claims](secretKey string, newClaims func() T) *JWTService[T] {
	return &JWTService[T]{
		secretKey: secretKey,
		newClaims: newClaims,
	}
}

// GenerateToken creates a new JWT token string by signing the provided claims
// with HMAC-SHA256 using the service's secret key.
//
// The claims parameter should include at least the standard RegisteredClaims
// fields (ExpiresAt, IssuedAt, etc.) in addition to any custom fields.
// The function does NOT validate the claims — if you omit ExpiresAt, the
// token will have no expiration.
//
// Returns the signed token string (three base64url-encoded parts separated
// by dots), or an error if signing fails.
//
// Example:
//
//	token, err := svc.GenerateToken(&MyClaims{
//	    UserID: "user-123",
//	    RegisteredClaims: jwt.RegisteredClaims{
//	        ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
//	        IssuedAt:  jwt.NewNumericDate(time.Now()),
//	    },
//	})
func (s *JWTService[T]) GenerateToken(claims T) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.secretKey))
}

// ValidateToken parses and validates a JWT token string, returning the
// typed claims if the token is valid.
//
// The function performs the following checks:
//  1. Parses the token and verifies its structure (three base64url parts).
//  2. Ensures the signing method is HMAC (HS256/HS384/HS512) — rejects
//     tokens signed with other algorithms (e.g., RSA, none) to prevent
//     algorithm confusion attacks.
//  3. Verifies the signature using the service's secret key.
//  4. Validates standard claims (expiration, not-before, etc.).
//
// Returns:
//   - The typed claims T on success.
//   - ErrExpiredToken if the token's "exp" claim has passed.
//   - ErrInvalidToken for all other validation failures (bad signature,
//     malformed token, unsupported algorithm, etc.).
//
// Example:
//
//	claims, err := svc.ValidateToken(tokenString)
//	if errors.Is(err, jwtutil.ErrExpiredToken) {
//	    // handle expired token (e.g., prompt re-login)
//	}
//	if err != nil {
//	    // handle invalid token
//	}
//	fmt.Println(claims.UserID)
func (s *JWTService[T]) ValidateToken(tokenString string) (T, error) {
	claims := s.newClaims()
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(s.secretKey), nil
	})

	if err != nil {
		var zero T
		if errors.Is(err, jwt.ErrTokenExpired) {
			return zero, ErrExpiredToken
		}
		return zero, ErrInvalidToken
	}

	typedClaims, ok := token.Claims.(T)
	if !ok || !token.Valid {
		var zero T
		return zero, ErrInvalidToken
	}

	return typedClaims, nil
}
