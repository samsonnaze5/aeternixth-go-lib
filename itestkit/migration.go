package itestkit

import (
	"context"
	"fmt"

	"github.com/samsonnaze5/aeternixth-go-lib/itestkit/internal/fileutil"
	"github.com/samsonnaze5/aeternixth-go-lib/itestkit/internal/sqlrunner"
)

// applyPostgresMigrations runs every migration directory configured for each
// PostgreSQL instance, in lexicographic file order. It stops at the first
// error and wraps the failure with the instance name and file path so the
// caller can pinpoint the offending statement.
func applyPostgresMigrations(ctx context.Context, opts StackOptions, res map[string]*PostgresResource, log Logger) error {
	for name, cfg := range opts.Postgres {
		if !cfg.ApplyMigrations || len(cfg.MigrationPaths) == 0 {
			continue
		}
		r, ok := res[name]
		if !ok {
			continue
		}
		if cfg.DropPublicSchemaBeforeMigrate {
			if err := DropPostgresPublicSchema(ctx, r.DSN); err != nil {
				return fmt.Errorf("apply postgres[%s] drop public schema: %w", name, err)
			}
		}
		files, missing, err := fileutil.CollectSQLFiles(cfg.MigrationPaths, cfg.StrictPath)
		if err != nil {
			return fmt.Errorf("apply postgres[%s] collect migrations: %w", name, err)
		}
		for _, m := range missing {
			log.Printf("itestkit: postgres[%s] migration path missing, skipping: %s", name, m)
		}
		for _, f := range files {
			if err := execPostgresFile(ctx, r.DSN, f, fmt.Sprintf("postgres[%s] migration file %s", name, f)); err != nil {
				return fmt.Errorf("apply postgres[%s] migration file %s: %w", name, f, err)
			}
		}
	}
	return nil
}

// applyPostgresSeeds mirrors applyPostgresMigrations for seed paths. Seeds
// always run after migrations.
func applyPostgresSeeds(ctx context.Context, opts StackOptions, res map[string]*PostgresResource, log Logger) error {
	for name, cfg := range opts.Postgres {
		if !cfg.ApplySeeds || len(cfg.SeedPaths) == 0 {
			continue
		}
		r, ok := res[name]
		if !ok {
			continue
		}
		if len(cfg.TruncateTablesBeforeSeed) > 0 {
			if err := TruncatePostgres(ctx, r.DSN, cfg.TruncateTablesBeforeSeed...); err != nil {
				return fmt.Errorf("apply postgres[%s] truncate before seed: %w", name, err)
			}
		}
		files, missing, err := fileutil.CollectSQLFiles(cfg.SeedPaths, cfg.StrictPath)
		if err != nil {
			return fmt.Errorf("apply postgres[%s] collect seeds: %w", name, err)
		}
		for _, m := range missing {
			log.Printf("itestkit: postgres[%s] seed path missing, skipping: %s", name, m)
		}
		for _, f := range files {
			if err := execPostgresFile(ctx, r.DSN, f, fmt.Sprintf("postgres[%s] seed file %s", name, f)); err != nil {
				return fmt.Errorf("apply postgres[%s] seed file %s: %w", name, f, err)
			}
		}
	}
	return nil
}

// applyClickHouseMigrations runs ClickHouse migration paths in lexicographic
// order. Behaviour mirrors the PostgreSQL counterpart.
func applyClickHouseMigrations(ctx context.Context, opts StackOptions, res map[string]*ClickHouseResource, log Logger) error {
	for name, cfg := range opts.ClickHouse {
		if !cfg.ApplyMigrations || len(cfg.MigrationPaths) == 0 {
			continue
		}
		r, ok := res[name]
		if !ok {
			continue
		}
		files, missing, err := fileutil.CollectSQLFiles(cfg.MigrationPaths, cfg.StrictPath)
		if err != nil {
			return fmt.Errorf("apply clickhouse[%s] collect migrations: %w", name, err)
		}
		for _, m := range missing {
			log.Printf("itestkit: clickhouse[%s] migration path missing, skipping: %s", name, m)
		}
		for _, f := range files {
			if err := execClickHouseFile(ctx, r.DSN, f, fmt.Sprintf("clickhouse[%s] migration file %s", name, f)); err != nil {
				return fmt.Errorf("apply clickhouse[%s] migration file %s: %w", name, f, err)
			}
		}
	}
	return nil
}

// applyClickHouseSeeds runs ClickHouse seed paths in lexicographic order.
func applyClickHouseSeeds(ctx context.Context, opts StackOptions, res map[string]*ClickHouseResource, log Logger) error {
	for name, cfg := range opts.ClickHouse {
		if !cfg.ApplySeeds || len(cfg.SeedPaths) == 0 {
			continue
		}
		r, ok := res[name]
		if !ok {
			continue
		}
		files, missing, err := fileutil.CollectSQLFiles(cfg.SeedPaths, cfg.StrictPath)
		if err != nil {
			return fmt.Errorf("apply clickhouse[%s] collect seeds: %w", name, err)
		}
		for _, m := range missing {
			log.Printf("itestkit: clickhouse[%s] seed path missing, skipping: %s", name, m)
		}
		for _, f := range files {
			if err := execClickHouseFile(ctx, r.DSN, f, fmt.Sprintf("clickhouse[%s] seed file %s", name, f)); err != nil {
				return fmt.Errorf("apply clickhouse[%s] seed file %s: %w", name, f, err)
			}
		}
	}
	return nil
}

func execPostgresFile(ctx context.Context, dsn string, path string, label string) error {
	script, err := fileutil.ReadFile(path)
	if err != nil {
		return err
	}
	return sqlrunner.ExecPostgresFile(ctx, dsn, script, label)
}

func execClickHouseFile(ctx context.Context, dsn string, path string, label string) error {
	script, err := fileutil.ReadFile(path)
	if err != nil {
		return err
	}
	return sqlrunner.ExecClickHouseFile(ctx, dsn, script, label)
}
