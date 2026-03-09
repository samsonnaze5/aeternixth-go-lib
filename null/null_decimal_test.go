package null

import (
	"testing"

	"github.com/samsonnaze5/aeternixth-go-lib/decimal"
)

func TestToNullDecimal(t *testing.T) {
	// Test with nil pointer
	var nilPtr *decimal.Decimal = nil
	nullDec := ToNullDecimal(nilPtr)
	if nullDec.Valid {
		t.Errorf("Expected invalid NullDecimal for nil pointer")
	}

	// Test with valid pointer
	val := decimal.RequireFromString("123.456")
	validPtr := &val
	nullDecValid := ToNullDecimal(validPtr)
	if !nullDecValid.Valid {
		t.Errorf("Expected valid NullDecimal for non-nil pointer")
	}
	if !nullDecValid.Decimal.Equal(val) {
		t.Errorf("Expected Decimal %v, got %v", val, nullDecValid.Decimal)
	}
}

func TestToNullDecimalPointer(t *testing.T) {
	// Test with invalid (NULL) NullDecimal
	invalidNullDec := decimal.NullDecimal{Valid: false}
	if ToNullDecimalPointer(invalidNullDec) != nil {
		t.Errorf("Expected nil pointer for invalid NullDecimal")
	}

	// Test with valid NullDecimal
	val := decimal.RequireFromString("123.456")
	validNullDec := decimal.NullDecimal{Decimal: val, Valid: true}
	ptr := ToNullDecimalPointer(validNullDec)
	if ptr == nil {
		t.Errorf("Expected non-nil pointer for valid NullDecimal")
	} else if !ptr.Equal(val) {
		t.Errorf("Expected pointer to %v, got pointer to %v", val, *ptr)
	}
}
