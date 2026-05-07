package itestkit

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/clickhouse"
	tcnetwork "github.com/testcontainers/testcontainers-go/network"
)

const (
	defaultClickHouseImage    = "clickhouse/clickhouse-server:24.8-alpine"
	defaultClickHouseUsername = "default"
	defaultClickHousePassword = "test"
	clickHouseNativePort      = "9000/tcp"
	clickHouseHTTPPort        = "8123/tcp"
)

// startClickHouse launches one ClickHouse container per entry in instances.
// On any error every previously started container is terminated.
func startClickHouse(
	ctx context.Context,
	opts StackOptions,
	net *testcontainers.DockerNetwork,
) (map[string]*ClickHouseResource, error) {
	if len(opts.ClickHouse) == 0 {
		return nil, nil
	}
	out := make(map[string]*ClickHouseResource, len(opts.ClickHouse))
	for name, cfg := range opts.ClickHouse {
		res, err := startOneClickHouse(ctx, name, cfg, opts, net)
		if err != nil {
			terminateClickHouse(out, opts)
			return nil, fmt.Errorf("start clickhouse[%s]: %w", name, err)
		}
		out[name] = res
	}
	return out, nil
}

func startOneClickHouse(
	ctx context.Context,
	name string,
	cfg ClickHouseOptions,
	stackOpts StackOptions,
	net *testcontainers.DockerNetwork,
) (*ClickHouseResource, error) {
	image := cfg.Image
	if image == "" {
		image = defaultClickHouseImage
	}
	dbName := cfg.Database
	if dbName == "" {
		dbName = defaultDatabaseName(stackOpts.ProjectName, name)
	}
	username := cfg.Username
	if username == "" {
		username = defaultClickHouseUsername
	}
	password := cfg.Password
	if password == "" {
		password = defaultClickHousePassword
	}

	containerOpts := []testcontainers.ContainerCustomizer{
		clickhouse.WithDatabase(dbName),
		clickhouse.WithUsername(username),
		clickhouse.WithPassword(password),
	}

	if len(cfg.InitScripts) > 0 {
		paths := make([]string, 0, len(cfg.InitScripts))
		for _, script := range cfg.InitScripts {
			abs, err := filepath.Abs(script)
			if err != nil {
				return nil, fmt.Errorf("init script %s: %w", script, err)
			}
			paths = append(paths, abs)
		}
		containerOpts = append(containerOpts, clickhouse.WithInitScripts(paths...))
	}
	if cfg.ConfigFile != "" {
		abs, err := filepath.Abs(cfg.ConfigFile)
		if err != nil {
			return nil, fmt.Errorf("config file %s: %w", cfg.ConfigFile, err)
		}
		containerOpts = append(containerOpts, clickhouse.WithConfigFile(abs))
	}
	if net != nil {
		containerOpts = append(containerOpts, tcnetwork.WithNetwork([]string{"clickhouse-" + name}, net))
	}
	if len(cfg.ExtraEnv) > 0 {
		containerOpts = append(containerOpts, testcontainers.WithEnv(cfg.ExtraEnv))
	}

	container, err := clickhouse.Run(ctx, image, containerOpts...)
	if err != nil {
		return nil, err
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = testcontainers.TerminateContainer(container)
		return nil, fmt.Errorf("host: %w", err)
	}
	nativePort, err := container.MappedPort(ctx, clickHouseNativePort)
	if err != nil {
		_ = testcontainers.TerminateContainer(container)
		return nil, fmt.Errorf("native port: %w", err)
	}
	httpPort, err := container.MappedPort(ctx, clickHouseHTTPPort)
	if err != nil {
		_ = testcontainers.TerminateContainer(container)
		return nil, fmt.Errorf("http port: %w", err)
	}

	dsn := fmt.Sprintf(
		"clickhouse://%s:%s@%s:%s/%s",
		url.QueryEscape(username),
		url.QueryEscape(password),
		host,
		nativePort.Port(),
		dbName,
	)
	httpDSN := fmt.Sprintf(
		"http://%s:%s@%s:%s/?database=%s",
		url.QueryEscape(username),
		url.QueryEscape(password),
		host,
		httpPort.Port(),
		dbName,
	)

	return &ClickHouseResource{
		Name:       name,
		DSN:        dsn,
		HTTPDSN:    httpDSN,
		Host:       host,
		NativePort: nativePort.Port(),
		HTTPPort:   httpPort.Port(),
		Database:   dbName,
		Username:   username,
		Password:   password,
		Container:  container,
	}, nil
}

func terminateClickHouse(res map[string]*ClickHouseResource, opts StackOptions) {
	if opts.Debug.KeepContainersOnFailure {
		return
	}
	for _, r := range res {
		if c, ok := r.Container.(testcontainers.Container); ok && c != nil {
			_ = testcontainers.TerminateContainer(c)
		}
	}
}
