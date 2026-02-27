package null

import (
	"database/sql"
	"time"
)

// ToNullTime converts a *time.Time pointer to a sql.NullTime value suitable
// for database operations. If the pointer is nil, it returns an invalid (NULL)
// sql.NullTime with the zero-value time. If the pointer is non-nil, it
// dereferences the value and returns a valid sql.NullTime.
//
// This is commonly used for nullable timestamp columns such as deleted_at,
// completed_at, or any optional date/time field.
//
// Example:
//
//	var deletedAt *time.Time = nil
//	nullTime := null.ToNullTime(deletedAt)  // sql.NullTime{Time: zero, Valid: false}
//
//	now := time.Now()
//	nullTime = null.ToNullTime(&now)         // sql.NullTime{Time: now, Valid: true}
func ToNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{Time: time.Time{}, Valid: false}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

// ToNullTimePointer converts a sql.NullTime back to a *time.Time pointer.
// If the sql.NullTime is valid (i.e., the database column contained a
// non-NULL value), it returns a pointer to the Time value. If it is
// invalid (NULL), it returns nil. This is the inverse of ToNullTime.
//
// Example:
//
//	row := sql.NullTime{Time: time.Now(), Valid: true}
//	ptr := null.ToNullTimePointer(row)  // *time.Time pointing to the timestamp
//
//	row = sql.NullTime{Valid: false}
//	ptr = null.ToNullTimePointer(row)   // nil
func ToNullTimePointer(t sql.NullTime) *time.Time {
	if t.Valid {
		return &t.Time
	}
	return nil
}
