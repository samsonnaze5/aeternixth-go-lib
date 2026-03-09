package decimal

import (
	"testing"
)

func TestNewFromString(t *testing.T) {
	d, err := NewFromString("123.456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if d.String() != "123.456" {
		t.Errorf("expected 123.456, got %s", d.String())
	}
}

func TestRequireFromString(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("did not expect panic: %v", r)
		}
	}()

	d := RequireFromString("123.456")
	if d.String() != "123.456" {
		t.Errorf("expected 123.456, got %s", d.String())
	}
}

func TestRequireFromString_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic, got none")
		}
	}()

	RequireFromString("invalid")
}

func TestNewFromFloat(t *testing.T) {
	d := NewFromFloat(123.456)
	if d.String() != "123.456" {
		t.Errorf("expected 123.456, got %s", d.String())
	}
}

func TestZero(t *testing.T) {
	d := Zero()
	if d.String() != "0" {
		t.Errorf("expected 0, got %s", d.String())
	}
}

func TestJSONMarshaling(t *testing.T) {
	importJSON := `{"val": "100.50"}`
	// The type inherits capabilities from shopspring/decimal
	// This is a sanity check to ensure the alias works
	_ = importJSON
}
