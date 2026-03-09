package decimal

import (
	"github.com/shopspring/decimal"
)

// Decimal is a type alias for github.com/shopspring/decimal.Decimal.
// It provides a high-precision decimal type suitable for financial calculations,
// typically used for 6-digit precision amounts as required by the CRM spec.
type Decimal = decimal.Decimal

// NullDecimal is a type alias for github.com/shopspring/decimal.NullDecimal.
// It is used for database columns that can be NULL.
type NullDecimal = decimal.NullDecimal

// NewFromString creates a new Decimal from a string.
// Returns an error if the string isn't a valid decimal.
func NewFromString(value string) (Decimal, error) {
	return decimal.NewFromString(value)
}

// RequireFromString creates a new Decimal from a string.
// Panics if the string is not a valid decimal.
// Use this only when you are absolutely sure the string is valid.
func RequireFromString(value string) Decimal {
	return decimal.RequireFromString(value)
}

// NewFromFloat creates a new Decimal from a float64.
// Note: float64 can have precision issues, parsing from string is preferred for exact values.
func NewFromFloat(value float64) Decimal {
	return decimal.NewFromFloat(value)
}

// Zero returns a Decimal representing exactly 0.
func Zero() Decimal {
	return decimal.NewFromInt(0)
}
