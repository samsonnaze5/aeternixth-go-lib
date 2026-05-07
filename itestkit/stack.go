package itestkit

import (
	"context"
	"fmt"
	"testing"

	"github.com/samsonnaze5/aeternixth-go-lib/itestkit/internal/dockerutil"
	"github.com/testcontainers/testcontainers-go"
	tcnetwork "github.com/testcontainers/testcontainers-go/network"
)

// StartStack starts every infrastructure dependency described by opts and
// returns a Stack with connection information for each. It is the primary
// entry point of the package.
//
// The startup order is fixed: containers come up in the sequence
// PostgreSQL → ClickHouse → Redis → Kafka, then migrations and seeds run,
// then Redis FlushAll, then Kafka topic creation. If any step fails, every
// container started so far is terminated unless DebugOptions.KeepContainersOnFailure
// is set.
//
// When testing.TB is non-nil, StartStack registers the cleanup function
// with t.Cleanup so callers do not need to terminate the stack explicitly.
//
// StartStack does not panic — it returns wrapped errors that include the
// service type, instance name, and (where relevant) file path or topic name.
func StartStack(ctx context.Context, t testing.TB, opts StackOptions) (*Stack, error) {
	if err := validateOptions(opts); err != nil {
		return nil, err
	}
	opts = applyDefaults(opts)
	log := resolveLogger(opts.Logger, t)

	startupCtx, cancelStartup := context.WithTimeout(ctx, opts.Timeouts.StartupTimeout)
	defer cancelStartup()

	var (
		net *testcontainers.DockerNetwork
		err error
	)
	if opts.Network.Enabled {
		netName := opts.Network.Name
		if netName == "" {
			netName = dockerutil.NetworkName(opts.ProjectName)
		}
		net, err = tcnetwork.New(startupCtx, tcnetwork.WithLabels(map[string]string{
			"itestkit.project": opts.ProjectName,
		}))
		if err != nil {
			return nil, fmt.Errorf("create network %s: %w", netName, err)
		}
	}

	stack := &Stack{
		Postgres:   map[string]*PostgresResource{},
		ClickHouse: map[string]*ClickHouseResource{},
		Redis:      map[string]*RedisResource{},
		Kafka:      map[string]*KafkaResource{},
		HTTPMocks:  map[string]*HTTPMockResource{},
		LocalStack: map[string]*LocalStackResource{},
	}

	// terminate is the inverse of every step that succeeded so far. We build
	// it incrementally so a failure at step N undoes steps 1..N-1.
	terminate := func() {
		if opts.Debug.KeepContainersOnFailure {
			return
		}
		cleanupCtx, cancel := context.WithTimeout(context.Background(), opts.Timeouts.CleanupTimeout)
		defer cancel()
		terminatePostgres(stack.Postgres, opts)
		terminateClickHouse(stack.ClickHouse, opts)
		terminateRedis(stack.Redis, opts)
		terminateKafka(stack.Kafka, opts)
		terminateHTTPMocks(stack.HTTPMocks, opts)
		terminateLocalStack(stack.LocalStack, opts)
		if net != nil {
			_ = net.Remove(cleanupCtx)
		}
	}

	if stack.Postgres, err = startPostgres(startupCtx, opts, net); err != nil {
		terminate()
		return nil, err
	}
	if stack.ClickHouse, err = startClickHouse(startupCtx, opts, net); err != nil {
		terminate()
		return nil, err
	}
	if stack.Redis, err = startRedis(startupCtx, opts, net); err != nil {
		terminate()
		return nil, err
	}
	if stack.Kafka, err = startKafka(startupCtx, opts, net); err != nil {
		terminate()
		return nil, err
	}
	if stack.HTTPMocks, err = startHTTPMocks(startupCtx, opts, net, log); err != nil {
		terminate()
		return nil, err
	}
	if stack.LocalStack, err = startLocalStack(startupCtx, opts, net); err != nil {
		terminate()
		return nil, err
	}

	migrationCtx, cancelMig := context.WithTimeout(ctx, opts.Timeouts.MigrationTimeout)
	defer cancelMig()
	if err := applyPostgresMigrations(migrationCtx, opts, stack.Postgres, log); err != nil {
		terminate()
		return nil, err
	}
	if err := applyClickHouseMigrations(migrationCtx, opts, stack.ClickHouse, log); err != nil {
		terminate()
		return nil, err
	}

	seedCtx, cancelSeed := context.WithTimeout(ctx, opts.Timeouts.SeedTimeout)
	defer cancelSeed()
	if err := applyPostgresSeeds(seedCtx, opts, stack.Postgres, log); err != nil {
		terminate()
		return nil, err
	}
	if err := applyClickHouseSeeds(seedCtx, opts, stack.ClickHouse, log); err != nil {
		terminate()
		return nil, err
	}

	for name, cfg := range opts.Redis {
		if !cfg.FlushBeforeTest {
			continue
		}
		r, ok := stack.Redis[name]
		if !ok {
			continue
		}
		if err := RedisFlushAll(ctx, r.URL); err != nil {
			terminate()
			return nil, fmt.Errorf("flush redis[%s]: %w", name, err)
		}
	}

	topicCtx, cancelTopic := context.WithTimeout(ctx, opts.Timeouts.TopicTimeout)
	defer cancelTopic()
	if err := createAllKafkaTopics(topicCtx, opts, stack.Kafka); err != nil {
		terminate()
		return nil, err
	}

	mockCtx, cancelMock := context.WithTimeout(ctx, opts.Timeouts.HTTPMockSetupTimeout)
	defer cancelMock()
	if err := applyMockServerExpectations(mockCtx, opts, stack.HTTPMocks); err != nil {
		terminate()
		return nil, err
	}

	initCtx, cancelInit := context.WithTimeout(ctx, opts.Timeouts.LocalStackInitTimeout)
	defer cancelInit()
	if err := runLocalStackInitScripts(initCtx, opts, stack.LocalStack, log); err != nil {
		terminate()
		return nil, err
	}

	stack.Config = buildAppTestConfig(stack)

	cleanup := func(cleanupCtx context.Context) error {
		terminatePostgres(stack.Postgres, opts)
		terminateClickHouse(stack.ClickHouse, opts)
		terminateRedis(stack.Redis, opts)
		terminateKafka(stack.Kafka, opts)
		terminateHTTPMocks(stack.HTTPMocks, opts)
		terminateLocalStack(stack.LocalStack, opts)
		if net != nil {
			if err := net.Remove(cleanupCtx); err != nil {
				return fmt.Errorf("remove network: %w", err)
			}
		}
		return nil
	}
	stack.Cleanup = cleanup

	if t != nil {
		t.Cleanup(func() {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), opts.Timeouts.CleanupTimeout)
			defer cancel()
			_ = cleanup(cleanupCtx)
		})
	}

	if opts.Debug.PrintConnectionInfo {
		printConnectionInfo(stack, log)
	}
	if opts.Debug.PrintContainerLogs {
		printContainerLogs(ctx, stack, log)
	}

	return stack, nil
}

// buildAppTestConfig flattens the resource maps into the connection-string
// maps documented in the spec.
func buildAppTestConfig(s *Stack) AppTestConfig {
	cfg := AppTestConfig{
		PostgresDSNs:        map[string]string{},
		ClickHouseDSNs:      map[string]string{},
		ClickHouseHTTPDSNs:  map[string]string{},
		RedisAddrs:          map[string]string{},
		RedisURLs:           map[string]string{},
		KafkaBrokers:        map[string][]string{},
		HTTPMockBaseURLs:    map[string]string{},
		LocalStackEndpoints: map[string]string{},
		LocalStackRegions:   map[string]string{},
		LocalStackAccessKey: map[string]string{},
		LocalStackSecretKey: map[string]string{},
	}
	for name, r := range s.Postgres {
		cfg.PostgresDSNs[name] = r.DSN
	}
	for name, r := range s.ClickHouse {
		cfg.ClickHouseDSNs[name] = r.DSN
		cfg.ClickHouseHTTPDSNs[name] = r.HTTPDSN
	}
	for name, r := range s.Redis {
		cfg.RedisAddrs[name] = r.Addr
		cfg.RedisURLs[name] = r.URL
	}
	for name, r := range s.Kafka {
		cfg.KafkaBrokers[name] = r.Brokers
	}
	for name, r := range s.HTTPMocks {
		cfg.HTTPMockBaseURLs[name] = r.BaseURL
	}
	for name, r := range s.LocalStack {
		cfg.LocalStackEndpoints[name] = r.Endpoint
		cfg.LocalStackRegions[name] = r.Region
		cfg.LocalStackAccessKey[name] = r.AccessKeyID
		cfg.LocalStackSecretKey[name] = r.SecretAccessKey
	}
	return cfg
}
