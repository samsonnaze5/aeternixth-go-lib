package null

import (
	"database/sql"
)

// ToNullString converts a *string pointer to a sql.NullString value suitable
// for database operations. If the pointer is nil, it returns an invalid (NULL)
// sql.NullString with the placeholder string "NULL". If the pointer is non-nil,
// it dereferences the value and returns a valid sql.NullString.
//
// Note: When the pointer is nil, the String field is set to "NULL" (a literal
// string), but the Valid field is false, so the database driver will correctly
// insert a SQL NULL — the "NULL" string value is never persisted.
//
// Example:
//
//	var name *string = nil
//	nullName := null.ToNullString(name)  // sql.NullString{String: "NULL", Valid: false}
//
//	v := "Alice"
//	nullName = null.ToNullString(&v)     // sql.NullString{String: "Alice", Valid: true}
func ToNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{String: "NULL", Valid: false}
	}
	return sql.NullString{String: *s, Valid: true}
}

// ToNullStringPointer converts a sql.NullString back to a *string pointer.
// If the sql.NullString is valid (i.e., the database column contained a
// non-NULL value), it returns a pointer to the String value. If it is
// invalid (NULL), it returns nil. This is the inverse of ToNullString.
//
// Example:
//
//	row := sql.NullString{String: "hello", Valid: true}
//	ptr := null.ToNullStringPointer(row)  // *string pointing to "hello"
//
//	row = sql.NullString{Valid: false}
//	ptr = null.ToNullStringPointer(row)   // nil
func ToNullStringPointer(s sql.NullString) *string {
	if s.Valid {
		return &s.String
	}
	return nil
}
