// Package healthgorm adapts a *gorm.DB to the
// [github.com/samsonnaze5/aeternixth-go-lib/observability/health.Pinger]
// contract. Construct via [NewPinger]; the constructor rejects a nil
// *gorm.DB with [ErrNilDB].
package healthgorm
