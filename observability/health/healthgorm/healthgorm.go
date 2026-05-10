package healthgorm

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// ErrNilDB is returned by NewPinger when the supplied *gorm.DB is nil.
// It is errors.Is-comparable.
var ErrNilDB = errors.New("healthgorm: nil *gorm.DB")

// Pinger adapts a *gorm.DB to the health.Pinger contract by calling
// PingContext on the underlying *sql.DB. Construct via NewPinger.
type Pinger struct {
	db *gorm.DB
}

// NewPinger validates the supplied *gorm.DB and returns a *Pinger ready
// to register in a readiness map. A nil DB is rejected with ErrNilDB —
// fail-fast at construction so misconfiguration surfaces in startup
// logs rather than as a permanent /readyz failure.
func NewPinger(db *gorm.DB) (*Pinger, error) {
	if db == nil {
		return nil, ErrNilDB
	}
	return &Pinger{db: db}, nil
}

// Ping reaches the underlying *sql.DB via gorm.DB.DB() and calls
// PingContext. Errors are wrapped under the "healthgorm:" prefix so the
// readiness response identifies the source of failure without further
// inspection.
func (p *Pinger) Ping(ctx context.Context) error {
	sqlDB, err := p.db.DB()
	if err != nil {
		return fmt.Errorf("healthgorm: resolve sql.DB: %w", err)
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("healthgorm: %w", err)
	}
	return nil
}
