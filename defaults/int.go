// Package defaultutil provides helper functions that return a default value
// when a pointer is nil. These are useful for handling optional fields in
// request payloads, configuration structs, or any scenario where a nil
// pointer should fall back to a sensible default rather than causing a
// nil pointer dereference.
package defaultutil

// DefaultInt returns the value pointed to by ptr if it is non-nil, or the
// provided default value def if ptr is nil. This is a safe way to dereference
// an optional *int without risking a nil pointer panic.
//
// Example:
//
//	var pageSize *int = nil
//	size := defaultutil.DefaultInt(pageSize, 20)  // returns 20
//
//	v := 50
//	size = defaultutil.DefaultInt(&v, 20)          // returns 50
func DefaultInt(ptr *int, def int) int {
	if ptr != nil {
		return *ptr
	}
	return def
}
