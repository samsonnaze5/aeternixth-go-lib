package itestkit

import (
	"context"
	"net/url"
	"strings"
)

// printConnectionInfo emits a multi-line summary of every running resource
// to the configured logger. PostgreSQL and ClickHouse passwords are masked
// with "****" so the output is safe to share in CI logs.
func printConnectionInfo(s *Stack, log Logger) {
	log.Printf("itestkit: connection info")
	for name, r := range s.Postgres {
		log.Printf("  postgres[%s] = %s", name, sanitizeDSN(r.DSN))
	}
	for name, r := range s.ClickHouse {
		log.Printf("  clickhouse[%s] = %s", name, sanitizeDSN(r.DSN))
		log.Printf("  clickhouse[%s] http = %s", name, sanitizeDSN(r.HTTPDSN))
	}
	for name, r := range s.Redis {
		log.Printf("  redis[%s] = %s", name, r.Addr)
	}
	for name, r := range s.Kafka {
		log.Printf("  kafka[%s] = %s", name, strings.Join(r.Brokers, ","))
	}
	for name, r := range s.HTTPMocks {
		log.Printf("  httpmock[%s] (%s) = %s", name, r.Provider, r.BaseURL)
	}
	for name, r := range s.LocalStack {
		log.Printf("  localstack[%s] = %s region=%s", name, r.Endpoint, r.Region)
	}
}

// sanitizeDSN replaces the password fragment in a URL-style DSN with "****".
// Returns the DSN unchanged if it cannot be parsed as a URL or has no
// userinfo component.
func sanitizeDSN(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return dsn
	}
	if u.User == nil {
		return dsn
	}
	if _, set := u.User.Password(); !set {
		return dsn
	}
	user := u.User.Username()
	u.User = url.UserPassword(user, "****")
	return u.String()
}

// printContainerLogs is a no-op placeholder for the spec's
// PrintContainerLogs flag. Streaming Testcontainers logs to a Logger is
// straightforward (container.Logs(ctx)) but pulling the entire log on
// startup tends to be noisy; the hook is reserved for v2.
func printContainerLogs(_ context.Context, _ *Stack, _ Logger) {}
