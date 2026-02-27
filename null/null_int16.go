package null

import (
	"database/sql"
)

// ToNullInt16 converts a *int16 pointer to a sql.NullInt16 value suitable for
// database operations. If the pointer is nil, it returns an invalid (NULL)
// sql.NullInt16 with the zero value 0. If the pointer is non-nil, it
// dereferences the value and returns a valid sql.NullInt16.
//
// This is useful for nullable SMALLINT database columns where the Go
// representation uses *int16 for optional fields.
//
// Example:
//
//	var status *int16 = nil
//	nullStatus := null.ToNullInt16(status)  // sql.NullInt16{Int16: 0, Valid: false}
//
//	v := int16(1)
//	nullStatus = null.ToNullInt16(&v)       // sql.NullInt16{Int16: 1, Valid: true}
func ToNullInt16(i *int16) sql.NullInt16 {
	if i == nil {
		return sql.NullInt16{Int16: 0, Valid: false}
	}
	return sql.NullInt16{Int16: *i, Valid: true}
}

// ToNullInt16Pointer converts a sql.NullInt16 back to a *int16 pointer.
// If the sql.NullInt16 is valid (i.e., the database column contained a
// non-NULL value), it returns a pointer to the Int16 value. If it is
// invalid (NULL), it returns nil. This is the inverse of ToNullInt16.
//
// Example:
//
//	row := sql.NullInt16{Int16: 100, Valid: true}
//	ptr := null.ToNullInt16Pointer(row)  // *int16 pointing to 100
//
//	row = sql.NullInt16{Valid: false}
//	ptr = null.ToNullInt16Pointer(row)   // nil
func ToNullInt16Pointer(n sql.NullInt16) *int16 {
	if n.Valid {
		return &n.Int16
	}
	return nil
}
