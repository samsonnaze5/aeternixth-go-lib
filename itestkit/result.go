package itestkit

import "context"

// Stack is the value returned by StartStack. It exposes one resource map per
// service type, an AppTestConfig with connection strings, and a Cleanup
// function that terminates every container created during startup.
//
// Cleanup is also registered with t.Cleanup, so callers do not need to call
// it explicitly in test code. The exported Cleanup field is provided for
// callers that need to terminate the stack early.
type Stack struct {
	Postgres   map[string]*PostgresResource
	ClickHouse map[string]*ClickHouseResource
	Redis      map[string]*RedisResource
	Kafka      map[string]*KafkaResource

	HTTPMocks  map[string]*HTTPMockResource
	LocalStack map[string]*LocalStackResource

	Config AppTestConfig

	Cleanup func(ctx context.Context) error
}

// PostgresResource is the connection information for a single PostgreSQL
// instance. Container is the underlying Testcontainers container; it is
// typed as `any` to keep the public API stable across Testcontainers
// upgrades and to avoid leaking implementation detail.
type PostgresResource struct {
	Name      string
	DSN       string
	Host      string
	Port      string
	Database  string
	Username  string
	Password  string
	Container any
}

// ClickHouseResource is the connection information for a single ClickHouse
// instance. NativePort serves the binary protocol used by clickhouse-go;
// HTTPPort serves the HTTP interface and is exposed as HTTPDSN.
type ClickHouseResource struct {
	Name       string
	DSN        string
	HTTPDSN    string
	Host       string
	NativePort string
	HTTPPort   string
	Database   string
	Username   string
	Password   string
	Container  any
}

// RedisResource is the connection information for a single Redis instance.
type RedisResource struct {
	Name      string
	Addr      string
	URL       string
	Host      string
	Port      string
	Container any
}

// KafkaResource is the broker list and container handle for a single Kafka
// instance. Brokers contains the host:port endpoints reachable from the
// machine running the test (not from inside the Docker network).
type KafkaResource struct {
	Name      string
	Brokers   []string
	Container any
}

// HTTPMockResource is the connection information for a single HTTP mock
// instance. BaseURL is the address from the host machine; provider tests
// can also discriminate behaviour with the Provider field.
type HTTPMockResource struct {
	Name     string
	Provider HTTPMockProvider
	BaseURL  string
	Host     string
	Port     string

	Container any
}

// LocalStackResource is the connection information for a single LocalStack
// instance. Endpoint is the unified host:port URL where every enabled AWS
// service is reachable; AccessKeyID and SecretAccessKey are LocalStack's
// fixed test credentials ("test"/"test").
type LocalStackResource struct {
	Name     string
	Endpoint string
	Region   string

	AccessKeyID     string
	SecretAccessKey string

	Container any
}

// AppTestConfig collects the connection strings keyed by instance name.
// Helper methods return empty strings rather than panicking for unknown
// names, so test code can validate them inline.
type AppTestConfig struct {
	PostgresDSNs       map[string]string
	ClickHouseDSNs     map[string]string
	ClickHouseHTTPDSNs map[string]string
	RedisAddrs         map[string]string
	RedisURLs          map[string]string
	KafkaBrokers       map[string][]string

	HTTPMockBaseURLs map[string]string

	LocalStackEndpoints map[string]string
	LocalStackRegions   map[string]string
	LocalStackAccessKey map[string]string
	LocalStackSecretKey map[string]string
}

// PostgresDSN returns the DSN for the named PostgreSQL instance, or an empty
// string if no such instance exists.
func (c AppTestConfig) PostgresDSN(name string) string { return c.PostgresDSNs[name] }

// ClickHouseDSN returns the native DSN for the named ClickHouse instance.
func (c AppTestConfig) ClickHouseDSN(name string) string { return c.ClickHouseDSNs[name] }

// ClickHouseHTTPDSN returns the HTTP DSN for the named ClickHouse instance.
func (c AppTestConfig) ClickHouseHTTPDSN(name string) string { return c.ClickHouseHTTPDSNs[name] }

// RedisAddr returns the host:port address for the named Redis instance.
func (c AppTestConfig) RedisAddr(name string) string { return c.RedisAddrs[name] }

// RedisURL returns the redis:// URL for the named Redis instance.
func (c AppTestConfig) RedisURL(name string) string { return c.RedisURLs[name] }

// KafkaBrokerList returns the broker list for the named Kafka instance, or
// nil if no such instance exists.
func (c AppTestConfig) KafkaBrokerList(name string) []string { return c.KafkaBrokers[name] }

// HTTPMockBaseURL returns the base URL for the named HTTP mock instance.
func (c AppTestConfig) HTTPMockBaseURL(name string) string { return c.HTTPMockBaseURLs[name] }

// LocalStackEndpoint returns the AWS-compatible endpoint URL for the named
// LocalStack instance.
func (c AppTestConfig) LocalStackEndpoint(name string) string { return c.LocalStackEndpoints[name] }

// LocalStackRegion returns the AWS region configured for the named
// LocalStack instance.
func (c AppTestConfig) LocalStackRegion(name string) string { return c.LocalStackRegions[name] }

// LocalStackAccessKeyID returns the test access key ID for the named
// LocalStack instance.
func (c AppTestConfig) LocalStackAccessKeyID(name string) string { return c.LocalStackAccessKey[name] }

// LocalStackSecretAccessKey returns the test secret access key for the
// named LocalStack instance.
func (c AppTestConfig) LocalStackSecretAccessKey(name string) string {
	return c.LocalStackSecretKey[name]
}
