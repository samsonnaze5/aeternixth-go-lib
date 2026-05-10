package healthredis_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/samsonnaze5/aeternixth-go-lib/observability/health/healthredis"
)

func TestNewPinger_NilClient(t *testing.T) {
	_, err := healthredis.NewPinger(nil)
	if !errors.Is(err, healthredis.ErrNilClient) {
		t.Errorf("err: want ErrNilClient, got %v", err)
	}
}

func TestNewPinger_ValidClient(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	defer client.Close()

	p, err := healthredis.NewPinger(client)
	if err != nil {
		t.Fatalf("err: want nil, got %v", err)
	}
	if p == nil {
		t.Error("Pinger: want non-nil")
	}
}

// TestPinger_UnreachableBroker exercises the failure-path wiring without
// requiring a real Redis or testcontainers. Pointing the client at an
// almost-certainly-unused port returns a connection error from the
// underlying go-redis client; we verify the adapter wraps it under the
// "healthredis:" prefix as documented.
func TestPinger_UnreachableBroker(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:1", // tcpmux — should not be running
		DialTimeout: 200 * time.Millisecond,
	})
	defer client.Close()

	p, err := healthredis.NewPinger(client)
	if err != nil {
		t.Fatalf("NewPinger: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
	defer cancel()

	err = p.Ping(ctx)
	if err == nil {
		t.Fatal("Ping: want error against unreachable broker, got nil")
	}
	if !strings.HasPrefix(err.Error(), "healthredis:") {
		t.Errorf("error prefix: want healthredis:, got %q", err.Error())
	}
}
