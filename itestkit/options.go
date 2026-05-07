// Package itestkit is a reusable Go integration-test infrastructure package.
// It starts real infrastructure dependencies (PostgreSQL, ClickHouse, Redis,
// Kafka) using Testcontainers Go and returns connection information to
// downstream projects.
//
// itestkit owns the infrastructure lifecycle (container startup, cleanup,
// network setup, wait strategies, connection-string generation, migration
// and seed execution). Downstream projects own their schemas, seed data,
// Kafka topic names, business assertions, and test cases.
//
// The package supports multiple named instances of the same service type:
//
//	stack, err := itestkit.StartStack(ctx, t, itestkit.StackOptions{
//	    ProjectName: "wallet-service",
//	    Postgres: map[string]itestkit.PostgresOptions{
//	        "main":   {ApplyMigrations: true, MigrationPaths: []string{"testdata/main"}},
//	        "ledger": {ApplyMigrations: true, MigrationPaths: []string{"testdata/ledger"}},
//	    },
//	})
//
// itestkit must work with any Docker-compatible runtime that Testcontainers
// Go supports (Docker Desktop, Docker Engine, Colima, OrbStack, Podman, etc.).
// It must not depend on Testcontainers Desktop, Cloud, or Docker Compose.
package itestkit

import "time"

// StackOptions configures StartStack. At least one service map must contain
// at least one entry. Empty service maps simply disable that service type.
//
// Map keys are instance names. They must match the regex ^[a-z][a-z0-9_-]*$
// and must be unique within their service map.
type StackOptions struct {
	// ProjectName is used to derive default database names, Kafka cluster IDs,
	// and the shared Docker network name. If empty, "itest" is used.
	ProjectName string

	Postgres   map[string]PostgresOptions
	ClickHouse map[string]ClickHouseOptions
	Redis      map[string]RedisOptions
	Kafka      map[string]KafkaOptions

	// HTTPMocks adds MockServer or WireMock instances for emulating
	// third-party HTTP services in integration tests. Empty map disables.
	HTTPMocks map[string]HTTPMockOptions

	// LocalStack adds LocalStack instances for emulating AWS services
	// (S3, SQS, SNS, DynamoDB, …) in integration tests. Empty map disables.
	LocalStack map[string]LocalStackOptions

	Network  NetworkOptions
	Timeouts TimeoutOptions
	Debug    DebugOptions

	// Logger receives diagnostic output. If nil, StartStack uses the provided
	// testing.TB's Logf, falling back to a no-op logger if both are nil.
	Logger Logger
}

// NetworkOptions controls the shared Docker network created for the stack.
// When Enabled is true (the default), every container is attached to one
// shared user-defined bridge network so they can resolve each other by
// container name. Cleanup removes the network after the test.
type NetworkOptions struct {
	Enabled bool
	Name    string
}

// TimeoutOptions caps each phase of the lifecycle. Zero values fall back to
// the defaults documented in the spec (StartupTimeout=120s, MigrationTimeout=60s,
// SeedTimeout=60s, TopicTimeout=30s, CleanupTimeout=30s,
// HTTPMockSetupTimeout=30s, LocalStackInitTimeout=60s).
type TimeoutOptions struct {
	StartupTimeout   time.Duration
	MigrationTimeout time.Duration
	SeedTimeout      time.Duration
	TopicTimeout     time.Duration
	CleanupTimeout   time.Duration

	HTTPMockSetupTimeout  time.Duration
	LocalStackInitTimeout time.Duration
}

// DebugOptions toggles developer-facing behaviour. None of these flags should
// be enabled in CI by default.
//
//   - KeepContainersOnFailure leaves containers running when StartStack fails
//     so the developer can inspect them. Used together with `docker ps`.
//   - PrintConnectionInfo logs DSNs (with sanitized passwords) after startup.
//   - PrintContainerLogs streams container logs through the Logger.
//   - ReuseContainers is reserved for future use and currently has no effect.
type DebugOptions struct {
	Enabled bool

	KeepContainersOnFailure bool
	PrintConnectionInfo     bool
	PrintContainerLogs      bool
	ReuseContainers         bool
}

// PostgresOptions configures one PostgreSQL instance. All fields are optional —
// the defaults documented in the spec apply when fields are zero.
type PostgresOptions struct {
	Image    string
	Database string
	Username string
	Password string

	MigrationPaths []string
	SeedPaths      []string
	InitScripts    []string
	ConfigFile     string

	ApplyMigrations bool
	ApplySeeds      bool

	// StrictPath controls behaviour when a migration or seed path does not
	// exist. true returns an error; false logs a warning and skips the path.
	StrictPath bool

	// DropPublicSchemaBeforeMigrate runs `DROP SCHEMA public CASCADE; CREATE
	// SCHEMA public;` before applying migrations. This is rarely needed because
	// each instance gets a freshly started container.
	DropPublicSchemaBeforeMigrate bool

	// TruncateTablesBeforeSeed truncates the listed tables before seeds run.
	// Useful when tests run against reused containers (Debug.ReuseContainers).
	TruncateTablesBeforeSeed []string

	ExtraEnv map[string]string
}

// ClickHouseOptions configures one ClickHouse instance. ClickHouse exposes
// both a native (TCP) and HTTP port; itestkit returns DSNs for both.
type ClickHouseOptions struct {
	Image    string
	Database string
	Username string
	Password string

	MigrationPaths []string
	SeedPaths      []string
	InitScripts    []string
	ConfigFile     string

	ApplyMigrations bool
	ApplySeeds      bool

	StrictPath bool

	ExtraEnv map[string]string
}

// RedisOptions configures one Redis instance.
type RedisOptions struct {
	Image string

	ConfigFile string
	UseTLS     bool

	// FlushBeforeTest runs FLUSHALL after the container becomes healthy.
	FlushBeforeTest bool

	ExtraEnv map[string]string
}

// KafkaOptions configures one Kafka (KRaft mode) instance.
type KafkaOptions struct {
	Image     string
	ClusterID string

	Topics []KafkaTopic

	// CreateTopics creates Topics after the broker becomes healthy.
	CreateTopics bool

	ExtraEnv map[string]string
}

// KafkaTopic is a single topic to create on a Kafka instance. Partitions and
// ReplicationFactor default to 1 when zero.
type KafkaTopic struct {
	Name              string
	Partitions        int
	ReplicationFactor int
	Config            map[string]string
}

// HTTPMockProvider selects which HTTP mocking engine backs an HTTPMock
// instance. MockServer is the default because it allows expectations to
// be defined dynamically from Go test code. WireMock is preferred when
// the project ships file-based JSON mappings.
type HTTPMockProvider string

const (
	HTTPMockProviderMockServer HTTPMockProvider = "mockserver"
	HTTPMockProviderWireMock   HTTPMockProvider = "wiremock"
)

// HTTPMockOptions configures one HTTP mock instance.
//
// Provider selects MockServer (default) or WireMock. MockServer consumes
// Expectations programmatically; WireMock consumes MappingPaths and
// FilePaths. Mixing modes is rejected at validation time.
type HTTPMockOptions struct {
	Provider HTTPMockProvider

	Image string

	// MappingPaths is a list of host directories whose contents are loaded
	// into /home/wiremock/mappings inside the container. WireMock only.
	MappingPaths []string

	// FilePaths is a list of host directories whose contents are loaded
	// into /home/wiremock/__files inside the container. WireMock only.
	FilePaths []string

	// Expectations is a list of HTTP request/response pairs configured on
	// the MockServer instance after startup. MockServer only.
	Expectations []HTTPExpectation

	// StrictPath controls behaviour when MappingPaths or FilePaths point to
	// a missing directory. true returns an error; false logs a warning.
	StrictPath bool

	ExtraEnv map[string]string
}

// HTTPExpectation is a single MockServer expectation. Method and Path are
// matched against incoming requests; the matching response is returned.
//
// Defaults: Method = "GET" if empty, ResponseStatus = 200 if zero,
// Times = unlimited if zero.
type HTTPExpectation struct {
	Method string
	Path   string

	QueryParams map[string]string

	RequestHeaders map[string]string
	RequestBody    string

	ResponseStatus  int
	ResponseHeaders map[string]string
	ResponseBody    string

	Delay time.Duration

	Times int
}

// LocalStackOptions configures one LocalStack instance.
//
// Services lists which AWS services to enable inside LocalStack
// (e.g. "s3", "sqs", "sns", "dynamodb"). InitScripts are shell scripts
// (typically `awslocal …` invocations) executed inside the container
// after the AWS services are ready.
type LocalStackOptions struct {
	Image string

	Services []string
	Region   string

	// InitScripts is a list of host paths to shell scripts. Each script is
	// copied into the container and executed in order. Pass directories to
	// run every script inside (lexicographic order).
	InitScripts []string

	StrictPath bool

	ExtraEnv map[string]string
}
