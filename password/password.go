// Package passwordutil provides secure password hashing and verification
// functions using the bcrypt algorithm. Bcrypt is an adaptive hash function
// that is deliberately slow to compute, making it resistant to brute-force
// and rainbow-table attacks. The cost factor automatically makes the
// algorithm slower as hardware improves.
package passwordutil

import (
	"golang.org/x/crypto/bcrypt"
)

// HashPassword takes a plain-text password and returns its bcrypt hash.
// It uses bcrypt.DefaultCost (currently 10) as the work factor, which
// provides a good balance between security and performance.
//
// The returned hash string includes the algorithm version, cost factor,
// salt, and hash — everything needed to verify the password later. This
// means you do not need to store the salt separately.
//
// Returns an error if the password exceeds bcrypt's maximum length of
// 72 bytes or if an internal hashing error occurs.
//
// Example:
//
//	hash, err := passwordutil.HashPassword("my-secret-password")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	// Store hash in your database
func HashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

// VerifyPassword compares a previously hashed password with a plain-text
// password candidate. It returns nil if the password matches the hash,
// or bcrypt.ErrMismatchedHashAndPassword if they do not match.
//
// This function is timing-safe — it always takes roughly the same amount
// of time regardless of whether the password is correct, which prevents
// timing-based side-channel attacks.
//
// Example:
//
//	err := passwordutil.VerifyPassword(storedHash, "user-input-password")
//	if err != nil {
//	    // Password does not match
//	    return errors.New("invalid credentials")
//	}
//	// Password matches — proceed with authentication
func VerifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
