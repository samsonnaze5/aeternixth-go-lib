package null

import (
	"database/sql"
)

// ToNullBoolean converts a *bool pointer to a sql.NullBool value suitable for
// database operations. If the pointer is nil, it returns an invalid (NULL)
// sql.NullBool with the zero value false. If the pointer is non-nil, it
// dereferences the value and returns a valid sql.NullBool.
//
// This is useful for nullable BOOLEAN database columns where the Go
// representation uses *bool for optional true/false fields (e.g., is_active,
// is_verified).
//
// Example:
//
//	var active *bool = nil
//	nullActive := null.ToNullBoolean(active)  // sql.NullBool{Bool: false, Valid: false}
//
//	v := true
//	nullActive = null.ToNullBoolean(&v)       // sql.NullBool{Bool: true, Valid: true}
func ToNullBoolean(i *bool) sql.NullBool {
	if i == nil {
		return sql.NullBool{Bool: false, Valid: false}
	}
	return sql.NullBool{Bool: *i, Valid: true}
}

// ToNullBooleanPointer converts a sql.NullBool back to a *bool pointer.
// If the sql.NullBool is valid (i.e., the database column contained a
// non-NULL value), it returns a pointer to the Bool value. If it is
// invalid (NULL), it returns nil. This is the inverse of ToNullBoolean.
//
// Example:
//
//	row := sql.NullBool{Bool: true, Valid: true}
//	ptr := null.ToNullBooleanPointer(row)  // *bool pointing to true
//
//	row = sql.NullBool{Valid: false}
//	ptr = null.ToNullBooleanPointer(row)   // nil
func ToNullBooleanPointer(n sql.NullBool) *bool {
	if n.Valid {
		return &n.Bool
	}
	return nil
}
