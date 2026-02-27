// Package timeutil provides convenience functions for working with time in UTC.
// All functions in this package return time values in the UTC timezone,
// ensuring consistency across different server environments and avoiding
// timezone-related bugs.
package timeutil

import (
	"time"
)

// Now returns the current time in UTC. This is a convenience wrapper around
// time.Now().UTC() that ensures all timestamps in the application are
// consistently in UTC, regardless of the server's local timezone setting.
//
// Use this function instead of time.Now() throughout the application to
// maintain timezone consistency for database timestamps, logging, and
// token expiration calculations.
//
// Example:
//
//	createdAt := timeutil.Now()  // e.g., 2024-03-15 10:30:00 +0000 UTC
func Now() time.Time {
	return time.Now().UTC()
}

// NowPlusHour returns the current UTC time plus the specified number of hours.
// This is commonly used for setting expiration times for tokens, OTPs,
// cache entries, or any time-limited resource.
//
// The hour parameter accepts int64 to accommodate large hour values. Negative
// values can be used to get a time in the past.
//
// Example:
//
//	expiresAt := timeutil.NowPlusHour(24)  // 24 hours from now in UTC
//	pastTime := timeutil.NowPlusHour(-1)   // 1 hour ago in UTC
func NowPlusHour(hour int64) time.Time {
	return Now().Add(time.Duration(hour*60*60) * time.Second)
}
