package itestkit

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	tcnetwork "github.com/testcontainers/testcontainers-go/network"
)

const (
	defaultRedisImage = "redis:7-alpine"
	redisPort         = "6379/tcp"
)

// startRedis launches one Redis container per entry in instances. If any
// container fails to start, every container started so far is terminated.
//
// FlushBeforeTest is honoured by the orchestrator after every Redis
// instance is up; it is not invoked here.
func startRedis(
	ctx context.Context,
	opts StackOptions,
	net *testcontainers.DockerNetwork,
) (map[string]*RedisResource, error) {
	if len(opts.Redis) == 0 {
		return nil, nil
	}
	out := make(map[string]*RedisResource, len(opts.Redis))
	for name, cfg := range opts.Redis {
		res, err := startOneRedis(ctx, name, cfg, opts, net)
		if err != nil {
			terminateRedis(out, opts)
			return nil, fmt.Errorf("start redis[%s]: %w", name, err)
		}
		out[name] = res
	}
	return out, nil
}

func startOneRedis(
	ctx context.Context,
	name string,
	cfg RedisOptions,
	_ StackOptions,
	net *testcontainers.DockerNetwork,
) (*RedisResource, error) {
	image := cfg.Image
	if image == "" {
		image = defaultRedisImage
	}

	containerOpts := []testcontainers.ContainerCustomizer{}
	if cfg.ConfigFile != "" {
		abs, err := filepath.Abs(cfg.ConfigFile)
		if err != nil {
			return nil, fmt.Errorf("config file %s: %w", cfg.ConfigFile, err)
		}
		containerOpts = append(containerOpts, redis.WithConfigFile(abs))
	}
	if cfg.UseTLS {
		containerOpts = append(containerOpts, redis.WithTLS())
	}
	if net != nil {
		containerOpts = append(containerOpts, tcnetwork.WithNetwork([]string{"redis-" + name}, net))
	}
	if len(cfg.ExtraEnv) > 0 {
		containerOpts = append(containerOpts, testcontainers.WithEnv(cfg.ExtraEnv))
	}

	container, err := redis.Run(ctx, image, containerOpts...)
	if err != nil {
		return nil, err
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = testcontainers.TerminateContainer(container)
		return nil, fmt.Errorf("host: %w", err)
	}
	port, err := container.MappedPort(ctx, redisPort)
	if err != nil {
		_ = testcontainers.TerminateContainer(container)
		return nil, fmt.Errorf("mapped port: %w", err)
	}

	addr := fmt.Sprintf("%s:%s", host, port.Port())
	scheme := "redis"
	if cfg.UseTLS {
		scheme = "rediss"
	}
	urlStr := fmt.Sprintf("%s://%s", scheme, addr)

	return &RedisResource{
		Name:      name,
		Addr:      addr,
		URL:       urlStr,
		Host:      host,
		Port:      port.Port(),
		Container: container,
	}, nil
}

func terminateRedis(res map[string]*RedisResource, opts StackOptions) {
	if opts.Debug.KeepContainersOnFailure {
		return
	}
	for _, r := range res {
		if c, ok := r.Container.(testcontainers.Container); ok && c != nil {
			_ = testcontainers.TerminateContainer(c)
		}
	}
}
