package itestkit

import (
	"fmt"
	"os"
	"strings"

	"github.com/samsonnaze5/aeternixth-go-lib/itestkit/internal/stringutil"
)

// ExportEnv returns a map of environment variables for every resource in
// the stack. Naming convention: <SERVICE>_<INSTANCE>_<FIELD>, with the
// instance name uppercased and dashes replaced with underscores. Examples:
//
//	POSTGRES_CORE_DSN
//	CLICKHOUSE_EVENTS_HTTP_DSN
//	REDIS_CACHE_ADDR
//	KAFKA_MAIN_BROKERS  // comma-joined
//
// The returned map is freshly allocated; the caller may mutate it freely.
func (s *Stack) ExportEnv() map[string]string {
	out := map[string]string{}
	for name, r := range s.Postgres {
		key := fmt.Sprintf("POSTGRES_%s_DSN", stringutil.EnvFragment(name))
		out[key] = r.DSN
	}
	for name, r := range s.ClickHouse {
		frag := stringutil.EnvFragment(name)
		out["CLICKHOUSE_"+frag+"_DSN"] = r.DSN
		out["CLICKHOUSE_"+frag+"_HTTP_DSN"] = r.HTTPDSN
	}
	for name, r := range s.Redis {
		frag := stringutil.EnvFragment(name)
		out["REDIS_"+frag+"_ADDR"] = r.Addr
		out["REDIS_"+frag+"_URL"] = r.URL
	}
	for name, r := range s.Kafka {
		key := fmt.Sprintf("KAFKA_%s_BROKERS", stringutil.EnvFragment(name))
		out[key] = strings.Join(r.Brokers, ",")
	}
	for name, r := range s.HTTPMocks {
		key := fmt.Sprintf("HTTPMOCK_%s_BASE_URL", stringutil.EnvFragment(name))
		out[key] = r.BaseURL
	}
	for name, r := range s.LocalStack {
		frag := stringutil.EnvFragment(name)
		out["LOCALSTACK_"+frag+"_ENDPOINT"] = r.Endpoint
		out["LOCALSTACK_"+frag+"_REGION"] = r.Region
		out["LOCALSTACK_"+frag+"_ACCESS_KEY_ID"] = r.AccessKeyID
		out["LOCALSTACK_"+frag+"_SECRET_ACCESS_KEY"] = r.SecretAccessKey
	}
	return out
}

// ApplyEnv writes every variable from ExportEnv into the current process
// environment via os.Setenv. The change persists for the lifetime of the
// process; tests that want isolation should set t.Setenv before calling
// StartStack so the values are restored automatically.
func (s *Stack) ApplyEnv() {
	for k, v := range s.ExportEnv() {
		_ = os.Setenv(k, v)
	}
}
