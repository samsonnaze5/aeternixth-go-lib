package healthkafka_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/samsonnaze5/aeternixth-go-lib/observability/health/healthkafka"
)

func TestNewMetadataPinger_EmptyBrokers(t *testing.T) {
	_, err := healthkafka.NewMetadataPinger(nil, []string{"t1"}, nil)
	if !errors.Is(err, healthkafka.ErrEmptyBrokers) {
		t.Errorf("nil brokers: want ErrEmptyBrokers, got %v", err)
	}

	_, err = healthkafka.NewMetadataPinger([]string{}, []string{"t1"}, nil)
	if !errors.Is(err, healthkafka.ErrEmptyBrokers) {
		t.Errorf("empty brokers: want ErrEmptyBrokers, got %v", err)
	}
}

func TestNewMetadataPinger_EmptyTopics(t *testing.T) {
	_, err := healthkafka.NewMetadataPinger([]string{"b1:9092"}, nil, nil)
	if !errors.Is(err, healthkafka.ErrEmptyTopics) {
		t.Errorf("nil topics: want ErrEmptyTopics, got %v", err)
	}

	_, err = healthkafka.NewMetadataPinger([]string{"b1:9092"}, []string{}, nil)
	if !errors.Is(err, healthkafka.ErrEmptyTopics) {
		t.Errorf("empty topics: want ErrEmptyTopics, got %v", err)
	}
}

func TestNewMetadataPinger_ValidConfig(t *testing.T) {
	p, err := healthkafka.NewMetadataPinger([]string{"b1:9092"}, []string{"t1"}, nil)
	if err != nil {
		t.Fatalf("err: want nil, got %v", err)
	}
	if p == nil {
		t.Error("MetadataPinger: want non-nil")
	}
}

// TestPing_UnreachableBroker exercises the failure-path wiring without a
// real cluster. Dialling 127.0.0.1:1 will fail with a connection error;
// we verify the adapter wraps it under the "healthkafka:" prefix.
func TestPing_UnreachableBroker(t *testing.T) {
	p, err := healthkafka.NewMetadataPinger([]string{"127.0.0.1:1"}, []string{"any"}, nil)
	if err != nil {
		t.Fatalf("NewMetadataPinger: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
	defer cancel()

	err = p.Ping(ctx)
	if err == nil {
		t.Fatal("Ping: want error against unreachable broker, got nil")
	}
	if !strings.HasPrefix(err.Error(), "healthkafka:") {
		t.Errorf("error prefix: want healthkafka:, got %q", err.Error())
	}
}
