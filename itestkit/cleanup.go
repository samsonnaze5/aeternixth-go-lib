package itestkit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
)

// TruncatePostgres truncates the listed tables in a single statement using
// TRUNCATE ... RESTART IDENTITY CASCADE. Identifiers are double-quoted to
// be safe against case-sensitive names.
//
// If tables is empty, this function returns nil without opening a connection.
func TruncatePostgres(ctx context.Context, dsn string, tables ...string) error {
	if len(tables) == 0 {
		return nil
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open postgres: %w", err)
	}
	defer db.Close()

	parts := make([]string, len(tables))
	for i, t := range tables {
		parts[i] = quoteIdentifier(t)
	}
	stmt := fmt.Sprintf("TRUNCATE %s RESTART IDENTITY CASCADE", strings.Join(parts, ", "))
	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("truncate %s: %w", strings.Join(tables, ","), err)
	}
	return nil
}

// DropPostgresPublicSchema drops and recreates the `public` schema. This is
// used when DropPublicSchemaBeforeMigrate is set and is a fast way to reset
// the database before migrations run.
func DropPostgresPublicSchema(ctx context.Context, dsn string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open postgres: %w", err)
	}
	defer db.Close()

	if _, err := db.ExecContext(ctx, "DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;"); err != nil {
		return fmt.Errorf("drop public schema: %w", err)
	}
	return nil
}

// ExecClickHouse runs a single SQL statement against a ClickHouse instance.
func ExecClickHouse(ctx context.Context, dsn string, query string, args ...any) error {
	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return fmt.Errorf("open clickhouse: %w", err)
	}
	defer db.Close()

	if _, err := db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("exec clickhouse: %w", err)
	}
	return nil
}

// DropClickHouseTables issues a DROP TABLE IF EXISTS for every table in
// tables. The names are passed through unmodified — pass already-qualified
// names (e.g. `db.table`) when you need a non-default database.
func DropClickHouseTables(ctx context.Context, dsn string, tables ...string) error {
	if len(tables) == 0 {
		return nil
	}
	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return fmt.Errorf("open clickhouse: %w", err)
	}
	defer db.Close()

	for _, t := range tables {
		stmt := fmt.Sprintf("DROP TABLE IF EXISTS %s", t)
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("drop table %s: %w", t, err)
		}
	}
	return nil
}

// RedisFlushAll runs FLUSHALL against the Redis instance addressed by url.
// url accepts redis:// or rediss:// schemes (TLS).
func RedisFlushAll(ctx context.Context, url string) error {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return fmt.Errorf("parse redis url: %w", err)
	}
	client := redis.NewClient(opts)
	defer client.Close()

	if err := client.FlushAll(ctx).Err(); err != nil {
		return fmt.Errorf("flushall: %w", err)
	}
	return nil
}

// RedisSetJSON marshals value to JSON and sets it under key with the given
// TTL. Pass ttl=0 for no expiration. The Redis URL is parsed every call to
// keep the helper stateless and avoid leaking client connections.
func RedisSetJSON(ctx context.Context, url string, key string, value any, ttl time.Duration) error {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return fmt.Errorf("parse redis url: %w", err)
	}
	client := redis.NewClient(opts)
	defer client.Close()

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("set %s: %w", key, err)
	}
	return nil
}

// RedisGetJSON fetches key from Redis and unmarshals the result into out.
// If the key does not exist, it returns redis.Nil so callers can distinguish
// "missing key" from "decode error".
func RedisGetJSON(ctx context.Context, url string, key string, out any) error {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return fmt.Errorf("parse redis url: %w", err)
	}
	client := redis.NewClient(opts)
	defer client.Close()

	data, err := client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	return nil
}

// quoteIdentifier double-quotes a SQL identifier. Embedded double quotes are
// escaped by doubling, matching PostgreSQL's lexical rules.
func quoteIdentifier(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}
