package null

import (
	"database/sql"
	"time"
)

// DateLayout defines the expected date format used throughout this package.
// It follows Go's reference time layout "2006-01-02" which represents
// the ISO 8601 date format (YYYY-MM-DD). All date string parsing and
// formatting functions in this file use this layout.
const DateLayout = "2006-01-02"

// ToNullDate converts a *string pointer containing a date in "YYYY-MM-DD"
// format to a sql.NullTime value suitable for database operations.
//
// The function handles three cases:
//   - If the pointer is nil or points to an empty string, it returns an
//     invalid (NULL) sql.NullTime.
//   - If the string cannot be parsed as a valid date, it returns an invalid
//     (NULL) sql.NullTime (silently ignoring the parse error).
//   - If the string is a valid date, it returns a valid sql.NullTime with
//     the parsed time.Time value.
//
// This is commonly used when receiving date strings from JSON request bodies
// or query parameters that need to be stored in nullable DATE columns.
//
// Example:
//
//	dateStr := "2024-03-15"
//	nullDate := null.ToNullDate(&dateStr)  // sql.NullTime{Time: 2024-03-15, Valid: true}
//
//	nullDate = null.ToNullDate(nil)         // sql.NullTime{Valid: false}
func ToNullDate(dateStr *string) sql.NullTime {
	if dateStr == nil || *dateStr == "" {
		return sql.NullTime{Time: time.Time{}, Valid: false}
	}

	parsedTime, err := time.Parse(DateLayout, *dateStr)
	if err != nil {
		return sql.NullTime{Time: time.Time{}, Valid: false}
	}

	return sql.NullTime{Time: parsedTime, Valid: true}
}

// StringToTimePointer converts a *string pointer containing a date in
// "YYYY-MM-DD" format (e.g., "2024-03-15") to a *time.Time pointer.
//
// Returns nil in the following cases:
//   - The pointer is nil.
//   - The pointer points to an empty string.
//   - The string cannot be parsed as a valid "YYYY-MM-DD" date.
//
// This is useful when receiving optional date fields from JSON request
// bodies as *string and needing to convert them to *time.Time for
// domain logic or database storage.
//
// Example:
//
//	dateStr := "2024-03-15"
//	t := null.StringToTimePointer(&dateStr)  // *time.Time pointing to 2024-03-15
//
//	t = null.StringToTimePointer(nil)         // nil
func StringToTimePointer(dateStr *string) *time.Time {
	if dateStr == nil || *dateStr == "" {
		return nil
	}

	parsedTime, err := time.Parse(DateLayout, *dateStr)
	if err != nil {
		return nil
	}

	return &parsedTime
}

// TimePointerToString converts a *time.Time pointer to a *string pointer
// containing the date formatted as "YYYY-MM-DD". Returns nil if the time
// pointer is nil. This is the inverse of StringToTimePointer.
//
// This is useful when serializing optional date fields to JSON where the
// frontend expects date strings rather than full RFC 3339 timestamps.
//
// Example:
//
//	t := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
//	s := null.TimePointerToString(&t)  // *string pointing to "2024-03-15"
//
//	s = null.TimePointerToString(nil)   // nil
func TimePointerToString(t *time.Time) *string {
	if t == nil {
		return nil
	}

	dateStr := t.Format(DateLayout)
	return &dateStr
}

// ToNullDatePointer converts a sql.NullTime back to a *string pointer
// containing the date formatted as "YYYY-MM-DD". If the sql.NullTime is
// invalid (NULL), it returns nil. This is the inverse of ToNullDate.
//
// This is useful when reading nullable DATE columns from the database and
// converting them to string pointers for JSON serialization.
//
// Example:
//
//	row := sql.NullTime{Time: time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC), Valid: true}
//	ptr := null.ToNullDatePointer(row)  // *string pointing to "2024-03-15"
//
//	row = sql.NullTime{Valid: false}
//	ptr = null.ToNullDatePointer(row)   // nil
func ToNullDatePointer(nt sql.NullTime) *string {
	if !nt.Valid {
		return nil
	}

	dateStr := nt.Time.Format(DateLayout)
	return &dateStr
}
