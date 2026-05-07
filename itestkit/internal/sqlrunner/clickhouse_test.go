package sqlrunner

import (
	"reflect"
	"testing"
)

func TestSplitSQL_BasicStatements(t *testing.T) {
	got := splitSQL("SELECT 1; SELECT 2;")
	want := []string{"SELECT 1", "SELECT 2"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSplitSQL_RespectsQuotes(t *testing.T) {
	got := splitSQL("INSERT INTO t VALUES ('a;b'); SELECT 1;")
	want := []string{"INSERT INTO t VALUES ('a;b')", "SELECT 1"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSplitSQL_RespectsLineComment(t *testing.T) {
	got := splitSQL("SELECT 1; -- ; comment\nSELECT 2;")
	want := []string{"SELECT 1", "-- ; comment\nSELECT 2"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSplitSQL_RespectsBlockComment(t *testing.T) {
	got := splitSQL("SELECT 1; /* ; ; */ SELECT 2;")
	want := []string{"SELECT 1", "/* ; ; */ SELECT 2"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSplitSQL_NoTrailingSemicolon(t *testing.T) {
	got := splitSQL("SELECT 1")
	want := []string{"SELECT 1"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSplitSQL_Empty(t *testing.T) {
	got := splitSQL("   ;  ;  ")
	if len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
}
