package itestkit_test

import (
	"context"
	"testing"

	"github.com/samsonnaze5/aeternixth-go-lib/itestkit"
)

// ExampleStartStack_twoPostgres shows the minimal configuration for two
// PostgreSQL instances. The example is compile-only because running it
// requires a live Docker runtime.
func ExampleStartStack_twoPostgres() {
	var t *testing.T // supplied by the test framework
	stack, err := itestkit.StartStack(context.Background(), t, itestkit.StackOptions{
		ProjectName: "wallet-service",
		Postgres: map[string]itestkit.PostgresOptions{
			"main": {
				MigrationPaths:  []string{"testdata/postgres/main/migrations"},
				SeedPaths:       []string{"testdata/postgres/main/seeds"},
				ApplyMigrations: true,
				ApplySeeds:      true,
				StrictPath:      true,
			},
			"ledger": {
				MigrationPaths:  []string{"testdata/postgres/ledger/migrations"},
				ApplyMigrations: true,
				StrictPath:      true,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = stack.Config.PostgresDSN("main")
	_ = stack.Config.PostgresDSN("ledger")
}

// ExampleStartStack_httpMockServer shows a MockServer-backed HTTP mock
// configured with one Go-defined expectation.
func ExampleStartStack_httpMockServer() {
	var t *testing.T
	stack, err := itestkit.StartStack(context.Background(), t, itestkit.StackOptions{
		ProjectName: "fx-service",
		HTTPMocks: map[string]itestkit.HTTPMockOptions{
			"exchange_rate": {
				Provider: itestkit.HTTPMockProviderMockServer,
				Expectations: []itestkit.HTTPExpectation{
					{
						Method:         "GET",
						Path:           "/v1/rates/USDTHB",
						ResponseStatus: 200,
						ResponseHeaders: map[string]string{
							"Content-Type": "application/json",
						},
						ResponseBody: `{"base":"USD","quote":"THB","rate":31.46}`,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = stack.Config.HTTPMockBaseURL("exchange_rate")
}

// ExampleStartStack_httpWireMock shows a WireMock-backed HTTP mock
// pointing at file-based JSON mappings.
func ExampleStartStack_httpWireMock() {
	var t *testing.T
	stack, err := itestkit.StartStack(context.Background(), t, itestkit.StackOptions{
		ProjectName: "payment-service",
		HTTPMocks: map[string]itestkit.HTTPMockOptions{
			"payment_gateway": {
				Provider:     itestkit.HTTPMockProviderWireMock,
				MappingPaths: []string{"testdata/wiremock/payment-gateway/mappings"},
				FilePaths:    []string{"testdata/wiremock/payment-gateway/__files"},
				StrictPath:   true,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = stack.Config.HTTPMockBaseURL("payment_gateway")
}

// ExampleStartStack_localStack shows LocalStack with S3 and SQS plus
// downstream-owned init scripts.
func ExampleStartStack_localStack() {
	var t *testing.T
	stack, err := itestkit.StartStack(context.Background(), t, itestkit.StackOptions{
		ProjectName: "document-service",
		LocalStack: map[string]itestkit.LocalStackOptions{
			"aws": {
				Services: []string{"s3", "sqs"},
				Region:   "ap-southeast-1",
				InitScripts: []string{
					"testdata/localstack/init/001_create_bucket.sh",
					"testdata/localstack/init/002_create_queue.sh",
				},
				StrictPath: true,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = stack.Config.LocalStackEndpoint("aws")
	_ = stack.Config.LocalStackRegion("aws")
	_ = stack.Config.LocalStackAccessKeyID("aws")
	_ = stack.Config.LocalStackSecretAccessKey("aws")
}

// ExampleStartStack_fullStack shows a full stack with PostgreSQL,
// ClickHouse, Redis, and Kafka.
func ExampleStartStack_fullStack() {
	var t *testing.T
	stack, err := itestkit.StartStack(context.Background(), t, itestkit.StackOptions{
		ProjectName: "order-service",
		Postgres: map[string]itestkit.PostgresOptions{
			"main": {
				MigrationPaths:  []string{"testdata/postgres/migrations"},
				SeedPaths:       []string{"testdata/postgres/seeds"},
				ApplyMigrations: true,
				ApplySeeds:      true,
				StrictPath:      true,
			},
		},
		ClickHouse: map[string]itestkit.ClickHouseOptions{
			"events": {
				MigrationPaths:  []string{"testdata/clickhouse/migrations"},
				ApplyMigrations: true,
				StrictPath:      true,
			},
		},
		Redis: map[string]itestkit.RedisOptions{
			"cache": {FlushBeforeTest: true},
		},
		Kafka: map[string]itestkit.KafkaOptions{
			"main": {
				CreateTopics: true,
				Topics: []itestkit.KafkaTopic{
					{Name: "order.created", Partitions: 3},
					{Name: "order.cancelled", Partitions: 3},
				},
			},
		},
		Debug: itestkit.DebugOptions{PrintConnectionInfo: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = stack.Config.PostgresDSN("main")
	_ = stack.Config.ClickHouseDSN("events")
	_ = stack.Config.RedisAddr("cache")
	_ = stack.Config.KafkaBrokerList("main")
}
