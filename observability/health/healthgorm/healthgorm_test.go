package healthgorm_test

import (
	"errors"
	"testing"

	"gorm.io/gorm"

	"github.com/samsonnaze5/aeternixth-go-lib/observability/health/healthgorm"
)

func TestNewPinger_NilDB(t *testing.T) {
	_, err := healthgorm.NewPinger(nil)
	if !errors.Is(err, healthgorm.ErrNilDB) {
		t.Errorf("err: want ErrNilDB, got %v", err)
	}
}

func TestNewPinger_ValidDB(t *testing.T) {
	// gorm.DB literal works for the validation check (NewPinger only
	// inspects nil-ness; Ping behavior is exercised in the integration
	// test).
	p, err := healthgorm.NewPinger(&gorm.DB{})
	if err != nil {
		t.Fatalf("err: want nil, got %v", err)
	}
	if p == nil {
		t.Error("Pinger: want non-nil")
	}
}
