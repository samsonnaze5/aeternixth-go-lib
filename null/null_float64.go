package null

import (
	"database/sql"
)

// ToNullFloat64 converts a *float64 pointer to a sql.NullFloat64 value suitable
// for database operations. If the pointer is nil, it returns an invalid (NULL)
// sql.NullFloat64 with the zero value 0. If the pointer is non-nil, it
// dereferences the value and returns a valid sql.NullFloat64.
//
// This is commonly used for nullable numeric columns that store decimal values
// such as price, rating, latitude/longitude, etc.
//
// Example:
//
//	var price *float64 = nil
//	nullPrice := null.ToNullFloat64(price)  // sql.NullFloat64{Float64: 0, Valid: false}
//
//	v := 99.99
//	nullPrice = null.ToNullFloat64(&v)      // sql.NullFloat64{Float64: 99.99, Valid: true}
func ToNullFloat64(i *float64) sql.NullFloat64 {
	if i == nil {
		return sql.NullFloat64{Float64: 0, Valid: false}
	}
	return sql.NullFloat64{Float64: *i, Valid: true}
}

// ToNullFloat64Pointer converts a sql.NullFloat64 back to a *float64 pointer.
// If the sql.NullFloat64 is valid (i.e., the database column contained a
// non-NULL value), it returns a pointer to the Float64 value. If it is
// invalid (NULL), it returns nil. This is the inverse of ToNullFloat64.
//
// Example:
//
//	row := sql.NullFloat64{Float64: 3.14, Valid: true}
//	ptr := null.ToNullFloat64Pointer(row)  // *float64 pointing to 3.14
//
//	row = sql.NullFloat64{Valid: false}
//	ptr = null.ToNullFloat64Pointer(row)   // nil
func ToNullFloat64Pointer(n sql.NullFloat64) *float64 {
	if n.Valid {
		return &n.Float64
	}
	return nil
}
