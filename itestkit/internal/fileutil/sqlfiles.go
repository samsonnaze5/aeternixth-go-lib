// Package fileutil discovers SQL files under user-supplied migration and
// seed directories. Discovery rules are documented in itestkit's README and
// the project requirement spec; this package is internal to itestkit.
package fileutil

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CollectSQLFiles returns the .sql files directly inside paths, sorted in
// lexicographic order across the combined set. Hidden files (those starting
// with '.') and subdirectories are skipped; nested traversal is intentionally
// not performed so file ordering remains predictable.
//
// If a path does not exist, the behaviour depends on strict:
//
//	strict = true  -> returns an error.
//	strict = false -> the path is reported via missing and skipped.
//
// Files within a single path are stable-sorted by filename. Files across
// multiple paths are sorted by filename only — callers wishing to preserve
// directory order should pass directories whose filenames already prefix
// the desired global order.
func CollectSQLFiles(paths []string, strict bool) (files []string, missing []string, err error) {
	for _, p := range paths {
		info, statErr := os.Stat(p)
		if os.IsNotExist(statErr) {
			if strict {
				return nil, nil, fmt.Errorf("path does not exist: %s", p)
			}
			missing = append(missing, p)
			continue
		}
		if statErr != nil {
			return nil, nil, fmt.Errorf("stat %s: %w", p, statErr)
		}
		if !info.IsDir() {
			if strings.HasSuffix(strings.ToLower(p), ".sql") {
				files = append(files, p)
			}
			continue
		}
		entries, readErr := os.ReadDir(p)
		if readErr != nil {
			return nil, nil, fmt.Errorf("read dir %s: %w", p, readErr)
		}
		var local []string
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, ".") {
				continue
			}
			if e.IsDir() {
				continue
			}
			if !strings.HasSuffix(strings.ToLower(name), ".sql") {
				continue
			}
			local = append(local, filepath.Join(p, name))
		}
		sort.Strings(local)
		files = append(files, local...)
	}
	sort.SliceStable(files, func(i, j int) bool {
		return filepath.Base(files[i]) < filepath.Base(files[j])
	})
	return files, missing, nil
}

// ReadFile reads the entire file at path and returns its contents as a string.
// Errors are wrapped with the file path so failures inside SQL pipelines stay
// traceable.
func ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	return string(data), nil
}
