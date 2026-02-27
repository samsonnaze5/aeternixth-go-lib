package null

import (
	"database/sql"
)

// ToNullInt32 converts a *int32 pointer to a sql.NullInt32 value suitable for
// database operations. If the pointer is nil, it returns an invalid (NULL)
// sql.NullInt32 with the zero value 0. If the pointer is non-nil, it
// dereferences the value and returns a valid sql.NullInt32.
//
// This is useful for nullable INTEGER database columns where the Go
// representation uses *int32 for optional fields.
//
// Example:
//
//	var quantity *int32 = nil
//	nullQty := null.ToNullInt32(quantity)  // sql.NullInt32{Int32: 0, Valid: false}
//
//	v := int32(50)
//	nullQty = null.ToNullInt32(&v)         // sql.NullInt32{Int32: 50, Valid: true}
func ToNullInt32(i *int32) sql.NullInt32 {
	if i == nil {
		return sql.NullInt32{Int32: 0, Valid: false}
	}
	return sql.NullInt32{Int32: *i, Valid: true}
}

// ToNullInt32Pointer converts a sql.NullInt32 back to a *int32 pointer.
// If the sql.NullInt32 is valid (i.e., the database column contained a
// non-NULL value), it returns a pointer to the Int32 value. If it is
// invalid (NULL), it returns nil. This is the inverse of ToNullInt32.
//
// Example:
//
//	row := sql.NullInt32{Int32: 200, Valid: true}
//	ptr := null.ToNullInt32Pointer(row)  // *int32 pointing to 200
//
//	row = sql.NullInt32{Valid: false}
//	ptr = null.ToNullInt32Pointer(row)   // nil
func ToNullInt32Pointer(n sql.NullInt32) *int32 {
	if n.Valid {
		return &n.Int32
	}
	return nil
}
