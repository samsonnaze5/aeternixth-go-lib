package itestkit

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/samsonnaze5/aeternixth-go-lib/itestkit/internal/stringutil"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	tcnetwork "github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	defaultPostgresImage    = "postgres:16-alpine"
	defaultPostgresUsername = "test"
	defaultPostgresPassword = "test"
)

// startPostgres launches one PostgreSQL container per entry in instances.
// On any error, every container that was successfully started is terminated
// before the error is returned (unless Debug.KeepContainersOnFailure is true).
func startPostgres(ctx context.Context, opts StackOptions, net *testcontainers.DockerNetwork) (map[string]*PostgresResource, error) {
	if len(opts.Postgres) == 0 {
		return nil, nil
	}
	out := make(map[string]*PostgresResource, len(opts.Postgres))
	for name, cfg := range opts.Postgres {
		res, err := startOnePostgres(ctx, name, cfg, opts, net)
		if err != nil {
			terminatePostgres(out, opts)
			return nil, fmt.Errorf("start postgres[%s]: %w", name, err)
		}
		out[name] = res
	}
	return out, nil
}

func startOnePostgres(
	ctx context.Context,
	name string,
	cfg PostgresOptions,
	stackOpts StackOptions,
	net *testcontainers.DockerNetwork,
) (*PostgresResource, error) {
	image := cfg.Image
	if image == "" {
		image = defaultPostgresImage
	}
	dbName := cfg.Database
	if dbName == "" {
		dbName = defaultDatabaseName(stackOpts.ProjectName, name)
	}
	username := cfg.Username
	if username == "" {
		username = defaultPostgresUsername
	}
	password := cfg.Password
	if password == "" {
		password = defaultPostgresPassword
	}

	containerOpts := []testcontainers.ContainerCustomizer{
		postgres.WithDatabase(dbName),
		postgres.WithUsername(username),
		postgres.WithPassword(password),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(stackOpts.Timeouts.StartupTimeout),
		),
	}

	for _, script := range cfg.InitScripts {
		abs, err := filepath.Abs(script)
		if err != nil {
			return nil, fmt.Errorf("init script %s: %w", script, err)
		}
		containerOpts = append(containerOpts, postgres.WithInitScripts(abs))
	}
	if cfg.ConfigFile != "" {
		abs, err := filepath.Abs(cfg.ConfigFile)
		if err != nil {
			return nil, fmt.Errorf("config file %s: %w", cfg.ConfigFile, err)
		}
		containerOpts = append(containerOpts, postgres.WithConfigFile(abs))
	}
	if net != nil {
		containerOpts = append(containerOpts, tcnetwork.WithNetwork([]string{"postgres-" + name}, net))
	}
	if len(cfg.ExtraEnv) > 0 {
		containerOpts = append(containerOpts, testcontainers.WithEnv(cfg.ExtraEnv))
	}

	container, err := postgres.Run(ctx, image, containerOpts...)
	if err != nil {
		return nil, err
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = testcontainers.TerminateContainer(container)
		return nil, fmt.Errorf("host: %w", err)
	}
	port, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		_ = testcontainers.TerminateContainer(container)
		return nil, fmt.Errorf("mapped port: %w", err)
	}

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		url.QueryEscape(username),
		url.QueryEscape(password),
		host,
		port.Port(),
		dbName,
	)

	return &PostgresResource{
		Name:      name,
		DSN:       dsn,
		Host:      host,
		Port:      port.Port(),
		Database:  dbName,
		Username:  username,
		Password:  password,
		Container: container,
	}, nil
}

// terminatePostgres terminates every PostgreSQL container in res unless
// debug overrides the behaviour. Errors are intentionally swallowed; the
// caller is already in an error path.
func terminatePostgres(res map[string]*PostgresResource, opts StackOptions) {
	if opts.Debug.KeepContainersOnFailure {
		return
	}
	for _, r := range res {
		if c, ok := r.Container.(testcontainers.Container); ok && c != nil {
			_ = testcontainers.TerminateContainer(c)
		}
	}
}

// defaultDatabaseName builds "<project>_<instance>_test" with normalized
// identifiers. Used as the default for both PostgreSQL and ClickHouse.
func defaultDatabaseName(project, instance string) string {
	return fmt.Sprintf(
		"%s_%s_test",
		stringutil.NormalizeIdentifier(project),
		stringutil.NormalizeIdentifier(instance),
	)
}
