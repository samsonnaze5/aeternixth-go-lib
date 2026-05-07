package fileutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollectSQLFiles_Lexicographic(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"002_b.sql", "001_a.sql", "003_c.sql", "ignored.txt", ".hidden.sql"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("SELECT 1;"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	files, missing, err := CollectSQLFiles([]string{dir}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(missing) != 0 {
		t.Errorf("expected no missing paths, got %v", missing)
	}
	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %v", files)
	}
	wantOrder := []string{"001_a.sql", "002_b.sql", "003_c.sql"}
	for i, f := range files {
		if filepath.Base(f) != wantOrder[i] {
			t.Errorf("files[%d] = %s, want %s", i, filepath.Base(f), wantOrder[i])
		}
	}
}

func TestCollectSQLFiles_StrictMissing(t *testing.T) {
	_, _, err := CollectSQLFiles([]string{"/nonexistent/path/here"}, true)
	if err == nil {
		t.Fatal("expected error for missing path with strict=true")
	}
}

func TestCollectSQLFiles_NonStrictMissing(t *testing.T) {
	files, missing, err := CollectSQLFiles([]string{"/nonexistent/path/here"}, false)
	if err != nil {
		t.Fatalf("expected no error with strict=false, got %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected no files, got %v", files)
	}
	if len(missing) != 1 {
		t.Errorf("expected 1 missing path, got %v", missing)
	}
}

func TestCollectSQLFiles_IgnoresSubdirs(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "subdir")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "001.sql"), []byte("SELECT 1;"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "002.sql"), []byte("SELECT 2;"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, _, err := CollectSQLFiles([]string{dir}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 file (subdir ignored), got %v", files)
	}
}
