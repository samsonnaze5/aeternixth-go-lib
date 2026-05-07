package itestkit

import (
	"fmt"
	"regexp"
	"time"
)

// instanceNameRegex enforces the spec's instance-name rule: lowercase ASCII
// letter first, then any of [a-z0-9_-].
var instanceNameRegex = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

// validateOptions checks the public StackOptions for problems before any
// container is started. All errors are returned eagerly with the prefix
// "validate stack options: ..." so callers can grep for them in test logs.
func validateOptions(opts StackOptions) error {
	total := len(opts.Postgres) + len(opts.ClickHouse) + len(opts.Redis) + len(opts.Kafka) +
		len(opts.HTTPMocks) + len(opts.LocalStack)
	if total == 0 {
		return fmt.Errorf("validate stack options: %w", ErrNoServiceConfigured)
	}

	for name := range opts.Postgres {
		if !instanceNameRegex.MatchString(name) {
			return fmt.Errorf("validate stack options: postgres[%s]: %w", name, ErrInvalidInstanceName)
		}
	}
	for name := range opts.ClickHouse {
		if !instanceNameRegex.MatchString(name) {
			return fmt.Errorf("validate stack options: clickhouse[%s]: %w", name, ErrInvalidInstanceName)
		}
	}
	for name := range opts.Redis {
		if !instanceNameRegex.MatchString(name) {
			return fmt.Errorf("validate stack options: redis[%s]: %w", name, ErrInvalidInstanceName)
		}
	}
	for name := range opts.Kafka {
		if !instanceNameRegex.MatchString(name) {
			return fmt.Errorf("validate stack options: kafka[%s]: %w", name, ErrInvalidInstanceName)
		}
	}
	for name, cfg := range opts.HTTPMocks {
		if !instanceNameRegex.MatchString(name) {
			return fmt.Errorf("validate stack options: httpmock[%s]: %w", name, ErrInvalidInstanceName)
		}
		provider := cfg.Provider
		if provider == "" {
			provider = HTTPMockProviderMockServer
		}
		switch provider {
		case HTTPMockProviderMockServer, HTTPMockProviderWireMock:
		default:
			return fmt.Errorf("validate stack options: httpmock[%s]: %w: %q", name, ErrInvalidHTTPMockProvider, provider)
		}
		if provider == HTTPMockProviderWireMock && len(cfg.Expectations) > 0 {
			return fmt.Errorf("validate stack options: httpmock[%s]: %w", name, ErrWireMockExpectations)
		}
	}
	for name := range opts.LocalStack {
		if !instanceNameRegex.MatchString(name) {
			return fmt.Errorf("validate stack options: localstack[%s]: %w", name, ErrInvalidInstanceName)
		}
	}
	return nil
}

// applyDefaults returns a StackOptions with zero-valued fields replaced by
// the defaults defined in the spec. The input is not modified.
func applyDefaults(opts StackOptions) StackOptions {
	if opts.ProjectName == "" {
		opts.ProjectName = "itest"
	}
	if !opts.Network.Enabled {
		// Default to enabled — Network.Enabled defaults to false in Go's zero
		// value, but the spec says default = true. We honour it here.
		opts.Network.Enabled = true
	}
	if opts.Timeouts.StartupTimeout == 0 {
		opts.Timeouts.StartupTimeout = 120 * time.Second
	}
	if opts.Timeouts.MigrationTimeout == 0 {
		opts.Timeouts.MigrationTimeout = 60 * time.Second
	}
	if opts.Timeouts.SeedTimeout == 0 {
		opts.Timeouts.SeedTimeout = 60 * time.Second
	}
	if opts.Timeouts.TopicTimeout == 0 {
		opts.Timeouts.TopicTimeout = 30 * time.Second
	}
	if opts.Timeouts.CleanupTimeout == 0 {
		opts.Timeouts.CleanupTimeout = 30 * time.Second
	}
	if opts.Timeouts.HTTPMockSetupTimeout == 0 {
		opts.Timeouts.HTTPMockSetupTimeout = 30 * time.Second
	}
	if opts.Timeouts.LocalStackInitTimeout == 0 {
		opts.Timeouts.LocalStackInitTimeout = 60 * time.Second
	}
	return opts
}
