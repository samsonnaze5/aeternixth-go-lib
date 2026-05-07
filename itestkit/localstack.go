package itestkit

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/localstack"
	tcnetwork "github.com/testcontainers/testcontainers-go/network"
)

const (
	defaultLocalStackImage  = "localstack/localstack:3.8"
	defaultLocalStackRegion = "ap-southeast-1"

	// LocalStack uses fixed test credentials regardless of what the client
	// sends; documented at https://docs.localstack.cloud/.
	localStackAccessKeyID     = "test"
	localStackSecretAccessKey = "test"
)

// startLocalStack launches one LocalStack container per entry in
// opts.LocalStack. Init scripts are deferred to runLocalStackInitScripts
// so a script failure can be attributed to a specific instance and path.
func startLocalStack(
	ctx context.Context,
	opts StackOptions,
	net *testcontainers.DockerNetwork,
) (map[string]*LocalStackResource, error) {
	if len(opts.LocalStack) == 0 {
		return nil, nil
	}
	out := make(map[string]*LocalStackResource, len(opts.LocalStack))
	for name, cfg := range opts.LocalStack {
		res, err := startOneLocalStack(ctx, name, cfg, net)
		if err != nil {
			terminateLocalStack(out, opts)
			return nil, fmt.Errorf("start localstack[%s]: %w", name, err)
		}
		out[name] = res
	}
	return out, nil
}

func startOneLocalStack(
	ctx context.Context,
	name string,
	cfg LocalStackOptions,
	net *testcontainers.DockerNetwork,
) (*LocalStackResource, error) {
	image := cfg.Image
	if image == "" {
		image = defaultLocalStackImage
	}
	region := cfg.Region
	if region == "" {
		region = defaultLocalStackRegion
	}

	env := map[string]string{
		"DEFAULT_REGION":     region,
		"AWS_DEFAULT_REGION": region,
	}
	if len(cfg.Services) > 0 {
		env["SERVICES"] = strings.Join(cfg.Services, ",")
	}
	for k, v := range cfg.ExtraEnv {
		env[k] = v
	}

	containerOpts := []testcontainers.ContainerCustomizer{
		testcontainers.WithEnv(env),
	}
	if net != nil {
		containerOpts = append(containerOpts, tcnetwork.WithNetwork([]string{"localstack-" + name}, net))
	}

	container, err := localstack.Run(ctx, image, containerOpts...)
	if err != nil {
		return nil, err
	}

	endpoint, err := container.PortEndpoint(ctx, "4566/tcp", "http")
	if err != nil {
		_ = testcontainers.TerminateContainer(container)
		return nil, fmt.Errorf("endpoint: %w", err)
	}

	return &LocalStackResource{
		Name:            name,
		Endpoint:        endpoint,
		Region:          region,
		AccessKeyID:     localStackAccessKeyID,
		SecretAccessKey: localStackSecretAccessKey,
		Container:       container,
	}, nil
}

// runLocalStackInitScripts copies each init script into /tmp/itestkit-init/
// inside the container and executes it via /bin/bash. Scripts run in the
// order provided; directories are expanded into their contained scripts in
// lexicographic order. Stdout and stderr are captured and included in the
// returned error so callers can debug awslocal failures inline.
func runLocalStackInitScripts(ctx context.Context, opts StackOptions, res map[string]*LocalStackResource, log Logger) error {
	for name, cfg := range opts.LocalStack {
		if len(cfg.InitScripts) == 0 {
			continue
		}
		r, ok := res[name]
		if !ok {
			continue
		}
		scripts, err := expandLocalStackScripts(cfg.InitScripts, cfg.StrictPath, name, log)
		if err != nil {
			return err
		}
		container, ok := r.Container.(testcontainers.Container)
		if !ok || container == nil {
			continue
		}
		for _, script := range scripts {
			if err := execLocalStackScript(ctx, container, name, script); err != nil {
				return err
			}
		}
	}
	return nil
}

// expandLocalStackScripts resolves directory entries into their contained
// .sh files (lexicographic order). File entries are kept in the caller's
// order. Missing paths follow StrictPath semantics.
func expandLocalStackScripts(paths []string, strict bool, instance string, log Logger) ([]string, error) {
	var out []string
	for _, p := range paths {
		info, err := os.Stat(p)
		if os.IsNotExist(err) {
			if strict {
				return nil, fmt.Errorf("validate localstack[%s] init script path %s: does not exist", instance, p)
			}
			log.Printf("itestkit: localstack[%s] init script path missing, skipping: %s", instance, p)
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", p, err)
		}
		if !info.IsDir() {
			out = append(out, p)
			continue
		}
		entries, err := os.ReadDir(p)
		if err != nil {
			return nil, fmt.Errorf("read dir %s: %w", p, err)
		}
		var local []string
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			if strings.HasPrefix(e.Name(), ".") {
				continue
			}
			if !strings.HasSuffix(e.Name(), ".sh") {
				continue
			}
			local = append(local, filepath.Join(p, e.Name()))
		}
		sort.Strings(local)
		out = append(out, local...)
	}
	return out, nil
}

func execLocalStackScript(ctx context.Context, container testcontainers.Container, instance string, scriptPath string) error {
	data, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("run localstack[%s] init script %s: %w", instance, scriptPath, err)
	}
	target := "/tmp/itestkit-init/" + filepath.Base(scriptPath)
	if err := container.CopyToContainer(ctx, data, target, 0o755); err != nil {
		return fmt.Errorf("run localstack[%s] init script %s: copy: %w", instance, scriptPath, err)
	}
	code, reader, err := container.Exec(ctx, []string{"/bin/bash", target})
	if err != nil {
		return fmt.Errorf("run localstack[%s] init script %s: %w", instance, scriptPath, err)
	}
	var output bytes.Buffer
	if reader != nil {
		_, _ = output.ReadFrom(reader)
	}
	if code != 0 {
		return fmt.Errorf(
			"run localstack[%s] init script %s: exit %d: %s",
			instance, scriptPath, code, strings.TrimSpace(output.String()),
		)
	}
	return nil
}

func terminateLocalStack(res map[string]*LocalStackResource, opts StackOptions) {
	if opts.Debug.KeepContainersOnFailure {
		return
	}
	for _, r := range res {
		if c, ok := r.Container.(testcontainers.Container); ok && c != nil {
			_ = testcontainers.TerminateContainer(c)
		}
	}
}
