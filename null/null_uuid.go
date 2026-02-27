package null

import (
	"github.com/google/uuid"
)

// ToNullUUID converts a *uuid.UUID pointer to a uuid.NullUUID value suitable
// for database operations. If the pointer is nil, it returns an invalid (NULL)
// uuid.NullUUID with a zero-value UUID. If the pointer is non-nil, it
// dereferences the value and returns a valid uuid.NullUUID.
//
// This is commonly used when inserting or updating rows where a UUID column
// is nullable (e.g., an optional foreign key reference).
//
// Example:
//
//	var refID *uuid.UUID = nil
//	nullRef := null.ToNullUUID(refID)  // uuid.NullUUID{UUID: uuid.Nil, Valid: false}
//
//	id := uuid.New()
//	nullRef = null.ToNullUUID(&id)     // uuid.NullUUID{UUID: <generated>, Valid: true}
func ToNullUUID(s *uuid.UUID) uuid.NullUUID {
	if s == nil {
		return uuid.NullUUID{UUID: uuid.UUID{}, Valid: false}
	}
	return uuid.NullUUID{UUID: *s, Valid: true}
}

// ToNullUUIDPointer converts a uuid.NullUUID back to a *uuid.UUID pointer.
// If the uuid.NullUUID is valid (i.e., the database column contained a
// non-NULL value), it returns a pointer to the UUID value. If it is
// invalid (NULL), it returns nil. This is the inverse of ToNullUUID.
//
// Example:
//
//	row := uuid.NullUUID{UUID: someUUID, Valid: true}
//	ptr := null.ToNullUUIDPointer(row)  // *uuid.UUID pointing to someUUID
//
//	row = uuid.NullUUID{Valid: false}
//	ptr = null.ToNullUUIDPointer(row)   // nil
func ToNullUUIDPointer(s uuid.NullUUID) *uuid.UUID {
	if s.Valid {
		return &s.UUID
	}
	return nil
}
