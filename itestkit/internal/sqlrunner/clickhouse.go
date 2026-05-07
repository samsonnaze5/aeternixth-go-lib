package sqlrunner

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/ClickHouse/clickhouse-go/v2"
)

// ExecClickHouseFile runs the contents of script against the ClickHouse
// instance addressed by dsn. The driver does not accept multi-statement
// scripts in a single Exec, so the script is split on top-level ';'
// boundaries (string and comment-aware) and each statement is executed in
// order. Empty trimmed statements are skipped.
//
// label is included in returned errors so the caller can identify which
// instance and file caused the failure.
func ExecClickHouseFile(ctx context.Context, dsn string, script string, label string) error {
	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return fmt.Errorf("open %s: %w", label, err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping %s: %w", label, err)
	}

	for _, stmt := range splitSQL(script) {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("exec %s: %w", label, err)
		}
	}
	return nil
}

// splitSQL splits a multi-statement SQL script on top-level ';' boundaries.
// Semicolons inside single-quoted, double-quoted, or backtick-quoted strings
// and inside line ('--') or block ('/* */') comments are preserved. The
// returned slice contains only non-empty trimmed statements.
func splitSQL(script string) []string {
	var (
		out      []string
		buf      []rune
		inSingle bool
		inDouble bool
		inBack   bool
		inLine   bool
		inBlock  bool
	)
	runes := []rune(script)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		next := rune(0)
		if i+1 < len(runes) {
			next = runes[i+1]
		}
		switch {
		case inLine:
			buf = append(buf, r)
			if r == '\n' {
				inLine = false
			}
		case inBlock:
			buf = append(buf, r)
			if r == '*' && next == '/' {
				buf = append(buf, next)
				i++
				inBlock = false
			}
		case inSingle:
			buf = append(buf, r)
			if r == '\'' {
				inSingle = false
			}
		case inDouble:
			buf = append(buf, r)
			if r == '"' {
				inDouble = false
			}
		case inBack:
			buf = append(buf, r)
			if r == '`' {
				inBack = false
			}
		default:
			switch {
			case r == '-' && next == '-':
				inLine = true
				buf = append(buf, r, next)
				i++
			case r == '/' && next == '*':
				inBlock = true
				buf = append(buf, r, next)
				i++
			case r == '\'':
				inSingle = true
				buf = append(buf, r)
			case r == '"':
				inDouble = true
				buf = append(buf, r)
			case r == '`':
				inBack = true
				buf = append(buf, r)
			case r == ';':
				stmt := trimSpace(string(buf))
				if stmt != "" {
					out = append(out, stmt)
				}
				buf = buf[:0]
			default:
				buf = append(buf, r)
			}
		}
	}
	if stmt := trimSpace(string(buf)); stmt != "" {
		out = append(out, stmt)
	}
	return out
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && isSpace(s[start]) {
		start++
	}
	for end > start && isSpace(s[end-1]) {
		end--
	}
	return s[start:end]
}

func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f' || b == '\v'
}
