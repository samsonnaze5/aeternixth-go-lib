// Package ratelimit provides an in-memory, key-based rate limiter with a
// fixed cooldown window. It is designed for scenarios where you want to
// limit how frequently a specific action can be performed per key (e.g.,
// per user ID, IP address, or API key).
//
// The limiter uses sync.Map for lock-free concurrent access and runs a
// background goroutine to periodically evict expired entries, preventing
// unbounded memory growth.
//
// Example:
//
//	limiter := ratelimit.NewLimiter(60 * time.Second)
//
//	allowed, retryAfter := limiter.Allow("user:123")
//	if !allowed {
//	    fmt.Printf("Rate limited. Try again in %v\n", retryAfter)
//	}
package ratelimit

import (
	"sync"
	"time"
)

// entry stores the expiration timestamp for a single rate-limit key.
// When a key is first seen (or its previous entry has expired), a new entry
// is created with expiresAt set to now + cooldown duration. Subsequent
// requests with the same key are blocked until expiresAt has passed.
type entry struct {
	expiresAt time.Time
}

// Limiter provides in-memory, key-based rate limiting with a fixed cooldown
// window. Each unique key (e.g., user ID, IP address) is allowed one action
// per cooldown period. After the action is performed, subsequent attempts
// with the same key are blocked until the cooldown expires.
//
// Limiter is safe for concurrent use from multiple goroutines. It uses
// sync.Map internally for lock-free read/write access.
//
// Important: Limiter starts a background cleanup goroutine that runs for
// the lifetime of the Limiter. There is no Stop method, so the goroutine
// will run until the process exits.
type Limiter struct {
	cooldown time.Duration
	store    sync.Map
}

// NewLimiter creates a new rate limiter with the specified cooldown duration.
// After a key is allowed, it will be blocked for the cooldown period before
// being allowed again.
//
// A background goroutine is started that runs every 5 minutes to evict
// expired entries from the internal store, preventing unbounded memory
// growth for short-lived keys.
//
// Example:
//
//	// Allow each user to send an OTP at most once per minute
//	otpLimiter := ratelimit.NewLimiter(60 * time.Second)
func NewLimiter(cooldown time.Duration) *Limiter {
	l := &Limiter{
		cooldown: cooldown,
	}

	go l.cleanup(5 * time.Minute)

	return l
}

// Allow checks whether the given key is allowed to proceed based on the
// cooldown window. It atomically checks and updates the rate limit state.
//
// Return values:
//   - allowed (bool): true if the action is permitted, false if rate-limited.
//   - retryAfter (time.Duration): if allowed is false, this indicates how long
//     the caller should wait before retrying. If allowed is true, this is 0.
//
// When allowed is true, the key is recorded with a new cooldown expiration.
// When allowed is false, the existing cooldown is not extended — the caller
// only needs to wait for the original cooldown to expire.
//
// This method is safe for concurrent use.
//
// Example:
//
//	allowed, retryAfter := limiter.Allow("user:456")
//	if !allowed {
//	    return fmt.Errorf("rate limited, retry after %v", retryAfter)
//	}
//	// proceed with the action
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	now := time.Now()
	newEntry := entry{expiresAt: now.Add(l.cooldown)}

	actual, loaded := l.store.LoadOrStore(key, newEntry)
	if !loaded {
		return true, 0
	}

	e := actual.(entry)
	if now.Before(e.expiresAt) {
		return false, e.expiresAt.Sub(now)
	}

	// Entry expired — replace it
	l.store.Store(key, newEntry)
	return true, 0
}

// Reset removes a key from the rate limit store, immediately clearing its
// cooldown. The next call to Allow with this key will return true regardless
// of whether the previous cooldown had expired.
//
// This is useful when you want to explicitly allow a key before its cooldown
// naturally expires, for example after a successful verification or when
// an admin manually resets a user's rate limit.
//
// Example:
//
//	limiter.Reset("user:456")  // clear the cooldown for user 456
func (l *Limiter) Reset(key string) {
	l.store.Delete(key)
}

// cleanup runs in a background goroutine and periodically scans the store
// to delete entries whose cooldown has expired. This prevents the sync.Map
// from growing unboundedly when keys are used once and never seen again
// (e.g., IP addresses from one-time visitors).
//
// The interval parameter controls how often the cleanup runs. A shorter
// interval keeps memory usage tighter but consumes more CPU; a longer
// interval is more efficient but allows more stale entries to accumulate.
func (l *Limiter) cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		l.store.Range(func(key, value any) bool {
			e := value.(entry)
			if now.After(e.expiresAt) {
				l.store.Delete(key)
			}
			return true
		})
	}
}
