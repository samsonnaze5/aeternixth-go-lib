package itestkit

import "testing"

func TestExportEnv(t *testing.T) {
	s := &Stack{
		Postgres: map[string]*PostgresResource{
			"core": {Name: "core", DSN: "postgres://test:test@localhost:5432/core_test"},
		},
		ClickHouse: map[string]*ClickHouseResource{
			"events": {Name: "events", DSN: "clickhouse://default:test@localhost:9000/e", HTTPDSN: "http://default:test@localhost:8123/?database=e"},
		},
		Redis: map[string]*RedisResource{
			"cache": {Name: "cache", Addr: "localhost:6379", URL: "redis://localhost:6379"},
		},
		Kafka: map[string]*KafkaResource{
			"main": {Name: "main", Brokers: []string{"localhost:19092", "localhost:19093"}},
		},
	}

	env := s.ExportEnv()

	wantKeys := []string{
		"POSTGRES_CORE_DSN",
		"CLICKHOUSE_EVENTS_DSN",
		"CLICKHOUSE_EVENTS_HTTP_DSN",
		"REDIS_CACHE_ADDR",
		"REDIS_CACHE_URL",
		"KAFKA_MAIN_BROKERS",
	}
	for _, k := range wantKeys {
		if _, ok := env[k]; !ok {
			t.Errorf("missing env var %q", k)
		}
	}
	if got, want := env["KAFKA_MAIN_BROKERS"], "localhost:19092,localhost:19093"; got != want {
		t.Errorf("KAFKA_MAIN_BROKERS = %q, want %q", got, want)
	}
}

func TestExportEnv_DashInInstance(t *testing.T) {
	s := &Stack{
		Postgres: map[string]*PostgresResource{
			"main-db": {Name: "main-db", DSN: "postgres://x"},
		},
	}
	env := s.ExportEnv()
	if _, ok := env["POSTGRES_MAIN_DB_DSN"]; !ok {
		t.Errorf("expected POSTGRES_MAIN_DB_DSN, got %v", env)
	}
}

func TestAppTestConfig_HelpersReturnEmpty(t *testing.T) {
	var c AppTestConfig
	if c.PostgresDSN("missing") != "" {
		t.Errorf("expected empty string for missing key")
	}
	if c.KafkaBrokerList("missing") != nil {
		t.Errorf("expected nil for missing key")
	}
	if c.HTTPMockBaseURL("missing") != "" {
		t.Errorf("expected empty string for missing httpmock key")
	}
	if c.LocalStackEndpoint("missing") != "" {
		t.Errorf("expected empty string for missing localstack key")
	}
	if c.LocalStackRegion("missing") != "" {
		t.Errorf("expected empty string for missing localstack region")
	}
	if c.LocalStackAccessKeyID("missing") != "" {
		t.Errorf("expected empty string for missing localstack access key")
	}
	if c.LocalStackSecretAccessKey("missing") != "" {
		t.Errorf("expected empty string for missing localstack secret key")
	}
}

func TestExportEnv_HTTPMocksAndLocalStack(t *testing.T) {
	s := &Stack{
		HTTPMocks: map[string]*HTTPMockResource{
			"payment_gateway": {
				Name:     "payment_gateway",
				Provider: HTTPMockProviderWireMock,
				BaseURL:  "http://localhost:23456",
			},
			"exchange_rate": {
				Name:     "exchange_rate",
				Provider: HTTPMockProviderMockServer,
				BaseURL:  "http://localhost:12345",
			},
		},
		LocalStack: map[string]*LocalStackResource{
			"aws": {
				Name:            "aws",
				Endpoint:        "http://localhost:4566",
				Region:          "ap-southeast-1",
				AccessKeyID:     "test",
				SecretAccessKey: "test",
			},
		},
	}

	env := s.ExportEnv()

	wantKeys := []string{
		"HTTPMOCK_PAYMENT_GATEWAY_BASE_URL",
		"HTTPMOCK_EXCHANGE_RATE_BASE_URL",
		"LOCALSTACK_AWS_ENDPOINT",
		"LOCALSTACK_AWS_REGION",
		"LOCALSTACK_AWS_ACCESS_KEY_ID",
		"LOCALSTACK_AWS_SECRET_ACCESS_KEY",
	}
	for _, k := range wantKeys {
		if _, ok := env[k]; !ok {
			t.Errorf("missing env var %q (got keys: %v)", k, mapKeys(env))
		}
	}
	if got := env["HTTPMOCK_PAYMENT_GATEWAY_BASE_URL"]; got != "http://localhost:23456" {
		t.Errorf("HTTPMOCK_PAYMENT_GATEWAY_BASE_URL = %q", got)
	}
	if got := env["LOCALSTACK_AWS_REGION"]; got != "ap-southeast-1" {
		t.Errorf("LOCALSTACK_AWS_REGION = %q", got)
	}
}

func mapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
