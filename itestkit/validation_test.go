package itestkit

import (
	"errors"
	"testing"
)

func TestValidateOptions_NoServices(t *testing.T) {
	err := validateOptions(StackOptions{})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, ErrNoServiceConfigured) {
		t.Fatalf("expected ErrNoServiceConfigured, got %v", err)
	}
}

func TestValidateOptions_BadInstanceName(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"starts with digit", "1main"},
		{"uppercase", "Main"},
		{"contains space", "main db"},
		{"empty", ""},
		{"contains dot", "main.db"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			opts := StackOptions{
				Postgres: map[string]PostgresOptions{tc.key: {}},
			}
			err := validateOptions(opts)
			if err == nil {
				t.Fatalf("expected error for %q, got nil", tc.key)
			}
			if !errors.Is(err, ErrInvalidInstanceName) {
				t.Fatalf("expected ErrInvalidInstanceName, got %v", err)
			}
		})
	}
}

func TestValidateOptions_GoodInstanceName(t *testing.T) {
	good := []string{"main", "main_db", "main-db", "core1", "wallet_2_test"}
	for _, name := range good {
		t.Run(name, func(t *testing.T) {
			opts := StackOptions{
				Postgres: map[string]PostgresOptions{name: {}},
			}
			if err := validateOptions(opts); err != nil {
				t.Fatalf("expected no error for %q, got %v", name, err)
			}
		})
	}
}

func TestApplyDefaults(t *testing.T) {
	opts := applyDefaults(StackOptions{
		Postgres: map[string]PostgresOptions{"main": {}},
	})
	if opts.ProjectName != "itest" {
		t.Errorf("ProjectName = %q, want %q", opts.ProjectName, "itest")
	}
	if !opts.Network.Enabled {
		t.Errorf("Network.Enabled = false, want true")
	}
	if opts.Timeouts.StartupTimeout == 0 {
		t.Errorf("StartupTimeout zero, want non-zero default")
	}
	if opts.Timeouts.HTTPMockSetupTimeout == 0 {
		t.Errorf("HTTPMockSetupTimeout zero, want non-zero default")
	}
	if opts.Timeouts.LocalStackInitTimeout == 0 {
		t.Errorf("LocalStackInitTimeout zero, want non-zero default")
	}
}

func TestValidateOptions_HTTPMockOnly(t *testing.T) {
	opts := StackOptions{
		HTTPMocks: map[string]HTTPMockOptions{
			"exchange_rate": {Provider: HTTPMockProviderMockServer},
		},
	}
	if err := validateOptions(opts); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestValidateOptions_LocalStackOnly(t *testing.T) {
	opts := StackOptions{
		LocalStack: map[string]LocalStackOptions{
			"aws": {Services: []string{"s3"}},
		},
	}
	if err := validateOptions(opts); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestValidateOptions_InvalidHTTPMockProvider(t *testing.T) {
	opts := StackOptions{
		HTTPMocks: map[string]HTTPMockOptions{
			"foo": {Provider: "bogus"},
		},
	}
	err := validateOptions(opts)
	if err == nil {
		t.Fatal("expected error for invalid provider")
	}
	if !errors.Is(err, ErrInvalidHTTPMockProvider) {
		t.Fatalf("expected ErrInvalidHTTPMockProvider, got %v", err)
	}
}

func TestValidateOptions_WireMockExpectationsRejected(t *testing.T) {
	opts := StackOptions{
		HTTPMocks: map[string]HTTPMockOptions{
			"foo": {
				Provider:     HTTPMockProviderWireMock,
				Expectations: []HTTPExpectation{{Path: "/x"}},
			},
		},
	}
	err := validateOptions(opts)
	if err == nil {
		t.Fatal("expected error for WireMock + expectations")
	}
	if !errors.Is(err, ErrWireMockExpectations) {
		t.Fatalf("expected ErrWireMockExpectations, got %v", err)
	}
}

func TestValidateOptions_HTTPMockEmptyProviderDefaultsMockServer(t *testing.T) {
	opts := StackOptions{
		HTTPMocks: map[string]HTTPMockOptions{"foo": {}},
	}
	if err := validateOptions(opts); err != nil {
		t.Fatalf("expected nil (empty provider should default to mockserver), got %v", err)
	}
}
