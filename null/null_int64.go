// Package null provides helper functions to convert between Go pointer types
// and database/sql Null* types (sql.NullInt64, sql.NullString, etc.).
// These are useful when working with database columns that allow NULL values,
// bridging the gap between Go's idiomatic *T pointers and the sql.Null* types
// required by database/sql drivers.
package null

import (
	"database/sql"
)

// ToNullInt64 converts a *int64 pointer to a sql.NullInt64 value suitable for
// database operations. If the pointer is nil, it returns an invalid (NULL)
// sql.NullInt64 with the zero value 0. If the pointer is non-nil, it
// dereferences the value and returns a valid sql.NullInt64.
//
// Example:
//
//	var age *int64 = nil
//	nullAge := null.ToNullInt64(age)  // sql.NullInt64{Int64: 0, Valid: false}
//
//	v := int64(25)
//	nullAge = null.ToNullInt64(&v)    // sql.NullInt64{Int64: 25, Valid: true}
func ToNullInt64(i *int64) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{Int64: 0, Valid: false}
	}
	return sql.NullInt64{Int64: *i, Valid: true}
}

// ToNullInt64Pointer converts a sql.NullInt64 back to a *int64 pointer.
// If the sql.NullInt64 is valid (i.e., the database column contained a
// non-NULL value), it returns a pointer to the Int64 value. If it is
// invalid (NULL), it returns nil. This is the inverse of ToNullInt64.
//
// Example:
//
//	row := sql.NullInt64{Int64: 42, Valid: true}
//	ptr := null.ToNullInt64Pointer(row)  // *int64 pointing to 42
//
//	row = sql.NullInt64{Valid: false}
//	ptr = null.ToNullInt64Pointer(row)   // nil
func ToNullInt64Pointer(n sql.NullInt64) *int64 {
	if n.Valid {
		return &n.Int64
	}
	return nil
}
