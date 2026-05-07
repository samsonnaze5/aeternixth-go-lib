package itestkit

import (
	"context"
	"fmt"

	"github.com/samsonnaze5/aeternixth-go-lib/itestkit/internal/fileutil"
	"github.com/samsonnaze5/aeternixth-go-lib/itestkit/internal/sqlrunner"
)

// ExecPostgresFile executes the SQL file at filePath against the PostgreSQL
// database addressed by dsn. The whole file is executed in one Exec call,
// so multi-statement scripts work naturally.
//
// Errors are wrapped with the file path so caller-level diagnostics can
// pinpoint which file failed.
func ExecPostgresFile(ctx context.Context, dsn string, filePath string) error {
	script, err := fileutil.ReadFile(filePath)
	if err != nil {
		return err
	}
	return sqlrunner.ExecPostgresFile(ctx, dsn, script, filePath)
}

// ExecPostgresDir executes every .sql file in dirPath against dsn in
// lexicographic order. Hidden files and subdirectories are skipped. Set
// strict=true to error on a missing directory; false will log nothing and
// return nil.
func ExecPostgresDir(ctx context.Context, dsn string, dirPath string, strict bool) error {
	files, _, err := fileutil.CollectSQLFiles([]string{dirPath}, strict)
	if err != nil {
		return fmt.Errorf("collect %s: %w", dirPath, err)
	}
	for _, f := range files {
		if err := ExecPostgresFile(ctx, dsn, f); err != nil {
			return err
		}
	}
	return nil
}

// ExecClickHouseFile executes the SQL file at filePath against the
// ClickHouse database addressed by dsn. The script is split on top-level
// semicolons (string- and comment-aware) and each statement is executed
// in order.
func ExecClickHouseFile(ctx context.Context, dsn string, filePath string) error {
	script, err := fileutil.ReadFile(filePath)
	if err != nil {
		return err
	}
	return sqlrunner.ExecClickHouseFile(ctx, dsn, script, filePath)
}

// ExecClickHouseDir executes every .sql file in dirPath against dsn in
// lexicographic order.
func ExecClickHouseDir(ctx context.Context, dsn string, dirPath string, strict bool) error {
	files, _, err := fileutil.CollectSQLFiles([]string{dirPath}, strict)
	if err != nil {
		return fmt.Errorf("collect %s: %w", dirPath, err)
	}
	for _, f := range files {
		if err := ExecClickHouseFile(ctx, dsn, f); err != nil {
			return err
		}
	}
	return nil
}
