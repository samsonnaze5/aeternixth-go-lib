package otelgorm_test

import (
	"context"

	"gorm.io/gorm"

	"github.com/samsonnaze5/aeternixth-go-lib/observability/otel/otelgorm"
)

func ExampleRegister() {
	var db *gorm.DB // from gorm.Open(...)
	if err := otelgorm.Register(db); err != nil {
		return // ErrNilDB here; a real *gorm.DB registers the plugin
	}

	// Traced only when ctx carries an active span (request/message ctx).
	var ctx context.Context
	db.WithContext(ctx).Exec("SELECT 1")
}
