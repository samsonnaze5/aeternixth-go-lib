package defaultutil

// DefaultString returns the value pointed to by ptr if it is non-nil, or the
// provided default value def if ptr is nil. This is a safe way to dereference
// an optional *string without risking a nil pointer panic.
//
// Example:
//
//	var sortBy *string = nil
//	col := defaultutil.DefaultString(sortBy, "created_at")  // returns "created_at"
//
//	v := "name"
//	col = defaultutil.DefaultString(&v, "created_at")        // returns "name"
func DefaultString(ptr *string, def string) string {
	if ptr != nil {
		return *ptr
	}
	return def
}
