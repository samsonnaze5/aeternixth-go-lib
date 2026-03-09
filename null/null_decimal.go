package null

import (
	"github.com/samsonnaze5/aeternixth-go-lib/decimal"
)

// ToNullDecimal converts a *decimal.Decimal pointer to a decimal.NullDecimal value suitable
// for database operations. If the pointer is nil, it returns an invalid (NULL)
// decimal.NullDecimal. If the pointer is non-nil, it dereferences the value and returns a valid decimal.NullDecimal.
//
// Example:
//
//	var amount *decimal.Decimal = nil
//	nullAmount := null.ToNullDecimal(amount)  // decimal.NullDecimal{Valid: false}
//
//	val := decimal.RequireFromString("10.50")
//	nullAmount = null.ToNullDecimal(&val)     // decimal.NullDecimal{Decimal: "10.50", Valid: true}
func ToNullDecimal(d *decimal.Decimal) decimal.NullDecimal {
	if d == nil {
		return decimal.NullDecimal{Valid: false}
	}
	return decimal.NullDecimal{Decimal: *d, Valid: true}
}

// ToNullDecimalPointer converts a decimal.NullDecimal back to a *decimal.Decimal pointer.
// If the decimal.NullDecimal is valid (i.e., the database column contained a
// non-NULL value), it returns a pointer to the Decimal value. If it is
// invalid (NULL), it returns nil. This is the inverse of ToNullDecimal.
//
// Example:
//
//	val := decimal.RequireFromString("10.50")
//	row := decimal.NullDecimal{Decimal: val, Valid: true}
//	ptr := null.ToNullDecimalPointer(row)  // *decimal.Decimal pointing to "10.50"
//
//	row = decimal.NullDecimal{Valid: false}
//	ptr = null.ToNullDecimalPointer(row)   // nil
func ToNullDecimalPointer(d decimal.NullDecimal) *decimal.Decimal {
	if d.Valid {
		return &d.Decimal
	}
	return nil
}
