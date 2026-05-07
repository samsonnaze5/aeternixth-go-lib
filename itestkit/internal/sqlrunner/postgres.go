// Package sqlrunner executes SQL files against PostgreSQL and ClickHouse
// databases. It deliberately avoids splitting statements client-side for
// PostgreSQL: pgx accepts multi-statement scripts in a single Exec call.
// ClickHouse's driver requires statement-by-statement execution, so this
// package also provides a small SQL splitter that respects strings and
// comments.
package sqlrunner

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// ExecPostgresFile runs script against the PostgreSQL instance addressed by
// dsn. label is included in returned errors so the caller can identify which
// instance and file caused the failure (e.g. "postgres[core] migration 001_init.sql").
//
// The connection is opened, used once, and closed — the runner is stateless
// and safe to call repeatedly in a sequence.
func ExecPostgresFile(ctx context.Context, dsn string, script string, label string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open %s: %w", label, err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping %s: %w", label, err)
	}

	if _, err := db.ExecContext(ctx, script); err != nil {
		return fmt.Errorf("exec %s: %w", label, err)
	}
	return nil
}
