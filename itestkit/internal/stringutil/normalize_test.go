package stringutil

import "testing"

func TestNormalizeIdentifier(t *testing.T) {
	tests := map[string]string{
		"":               "",
		"main":           "main",
		"Main":           "main",
		"WALLET":         "wallet",
		"wallet-service": "wallet-service",
		"wallet_service": "wallet_service",
		"Wallet Service": "wallet_service",
		"order-service!": "order-service_",
		"a/b":            "a_b",
	}
	for in, want := range tests {
		if got := NormalizeIdentifier(in); got != want {
			t.Errorf("NormalizeIdentifier(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEnvFragment(t *testing.T) {
	tests := map[string]string{
		"":        "",
		"main":    "MAIN",
		"main-db": "MAIN_DB",
		"main_db": "MAIN_DB",
		"core":    "CORE",
	}
	for in, want := range tests {
		if got := EnvFragment(in); got != want {
			t.Errorf("EnvFragment(%q) = %q, want %q", in, got, want)
		}
	}
}
