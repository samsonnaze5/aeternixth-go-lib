package defaultutil

// Set assigns fallback to *v when *v is the zero value for its type.
// Unlike DefaultInt/DefaultString which dereference a nil pointer,
// Set mutates the target in place — useful for applying defaults to
// config structs where fields are values (not pointers).
//
// Example:
//
//	var timeout time.Duration // zero value
//	defaultutil.Set(&timeout, 30*time.Second) // timeout is now 30s
//
//	timeout = 5 * time.Second
//	defaultutil.Set(&timeout, 30*time.Second) // timeout stays 5s
func Set[T comparable](v *T, fallback T) {
	var zero T
	if *v == zero {
		*v = fallback
	}
}
