package health_test

import (
	"context"
	"errors"
	"testing"

	"github.com/samsonnaze5/aeternixth-go-lib/observability/health"
)

func TestPingFunc_AdaptsToInterface(t *testing.T) {
	called := false
	var p health.Pinger = health.PingFunc(func(_ context.Context) error {
		called = true
		return nil
	})

	if err := p.Ping(context.Background()); err != nil {
		t.Errorf("err: want nil, got %v", err)
	}
	if !called {
		t.Error("PingFunc body was not invoked")
	}
}

func TestPingFunc_PropagatesError(t *testing.T) {
	want := errors.New("boom")
	var p health.Pinger = health.PingFunc(func(_ context.Context) error {
		return want
	})

	got := p.Ping(context.Background())
	if !errors.Is(got, want) {
		t.Errorf("err: want %v, got %v", want, got)
	}
}

func TestPingFunc_RespectsContext(t *testing.T) {
	// Verify the function receives the same context the caller passes —
	// adapters that drop ctx would break the pinger contract (I-1).
	type ctxKey struct{}
	caller := context.WithValue(context.Background(), ctxKey{}, "expected")

	var got context.Context
	p := health.PingFunc(func(c context.Context) error {
		got = c
		return nil
	})

	_ = p.Ping(caller)
	if got.Value(ctxKey{}) != "expected" {
		t.Errorf("ctx value: want %q, got %v", "expected", got.Value(ctxKey{}))
	}
}
