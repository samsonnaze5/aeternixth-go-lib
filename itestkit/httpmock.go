package itestkit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mockserver"
	tcnetwork "github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	defaultMockServerImage = "mockserver/mockserver:5.15.0"
	defaultWireMockImage   = "wiremock/wiremock:3.9.1"
	wireMockMappingsTarget = "/home/wiremock/mappings"
	wireMockFilesTarget    = "/home/wiremock/__files"
)

// startHTTPMocks launches one HTTP mock container per entry in opts.HTTPMocks.
// MockServer instances do not get expectations applied here — that is a
// post-startup step performed by applyMockServerExpectations once every
// container is healthy, so the orchestrator can attribute setup failures
// to the specific instance.
func startHTTPMocks(
	ctx context.Context,
	opts StackOptions,
	net *testcontainers.DockerNetwork,
	log Logger,
) (map[string]*HTTPMockResource, error) {
	if len(opts.HTTPMocks) == 0 {
		return nil, nil
	}
	out := make(map[string]*HTTPMockResource, len(opts.HTTPMocks))
	for name, cfg := range opts.HTTPMocks {
		provider := cfg.Provider
		if provider == "" {
			provider = HTTPMockProviderMockServer
		}
		var (
			res *HTTPMockResource
			err error
		)
		switch provider {
		case HTTPMockProviderMockServer:
			res, err = startOneMockServer(ctx, name, cfg, net)
		case HTTPMockProviderWireMock:
			res, err = startOneWireMock(ctx, name, cfg, net, log)
		default:
			err = fmt.Errorf("%w: %q", ErrInvalidHTTPMockProvider, provider)
		}
		if err != nil {
			terminateHTTPMocks(out, opts)
			return nil, fmt.Errorf("start httpmock[%s]: %w", name, err)
		}
		out[name] = res
	}
	return out, nil
}

func startOneMockServer(
	ctx context.Context,
	name string,
	cfg HTTPMockOptions,
	net *testcontainers.DockerNetwork,
) (*HTTPMockResource, error) {
	image := cfg.Image
	if image == "" {
		image = defaultMockServerImage
	}

	containerOpts := []testcontainers.ContainerCustomizer{}
	if net != nil {
		containerOpts = append(containerOpts, tcnetwork.WithNetwork([]string{"mockserver-" + name}, net))
	}
	if len(cfg.ExtraEnv) > 0 {
		containerOpts = append(containerOpts, testcontainers.WithEnv(cfg.ExtraEnv))
	}

	container, err := mockserver.Run(ctx, image, containerOpts...)
	if err != nil {
		return nil, err
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = testcontainers.TerminateContainer(container)
		return nil, fmt.Errorf("host: %w", err)
	}
	port, err := container.MappedPort(ctx, "1080/tcp")
	if err != nil {
		_ = testcontainers.TerminateContainer(container)
		return nil, fmt.Errorf("mapped port: %w", err)
	}

	baseURL := fmt.Sprintf("http://%s:%s", host, port.Port())
	return &HTTPMockResource{
		Name:      name,
		Provider:  HTTPMockProviderMockServer,
		BaseURL:   baseURL,
		Host:      host,
		Port:      port.Port(),
		Container: container,
	}, nil
}

// startOneWireMock starts a WireMock container with mappings and __files
// directories populated from MappingPaths and FilePaths. Multi-path support
// is achieved by copying each file individually via testcontainers.ContainerFile,
// avoiding the bind-mount limitation of a single source per target.
func startOneWireMock(
	ctx context.Context,
	name string,
	cfg HTTPMockOptions,
	net *testcontainers.DockerNetwork,
	log Logger,
) (*HTTPMockResource, error) {
	image := cfg.Image
	if image == "" {
		image = defaultWireMockImage
	}

	mappings, err := collectFilesForContainer(cfg.MappingPaths, wireMockMappingsTarget, cfg.StrictPath, fmt.Sprintf("httpmock[%s] mapping", name), log)
	if err != nil {
		return nil, err
	}
	files, err := collectFilesForContainer(cfg.FilePaths, wireMockFilesTarget, cfg.StrictPath, fmt.Sprintf("httpmock[%s] file", name), log)
	if err != nil {
		return nil, err
	}
	all := append(mappings, files...)

	env := map[string]string{}
	for k, v := range cfg.ExtraEnv {
		env[k] = v
	}

	req := testcontainers.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{"8080/tcp"},
		Env:          env,
		Files:        all,
		WaitingFor:   wait.ForHTTP("/__admin/health").WithPort("8080/tcp"),
	}
	if net != nil {
		req.Networks = []string{net.Name}
		req.NetworkAliases = map[string][]string{net.Name: {"wiremock-" + name}}
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = testcontainers.TerminateContainer(container)
		return nil, fmt.Errorf("host: %w", err)
	}
	port, err := container.MappedPort(ctx, "8080/tcp")
	if err != nil {
		_ = testcontainers.TerminateContainer(container)
		return nil, fmt.Errorf("mapped port: %w", err)
	}

	baseURL := fmt.Sprintf("http://%s:%s", host, port.Port())
	return &HTTPMockResource{
		Name:      name,
		Provider:  HTTPMockProviderWireMock,
		BaseURL:   baseURL,
		Host:      host,
		Port:      port.Port(),
		Container: container,
	}, nil
}

// collectFilesForContainer walks every host path in paths and produces
// testcontainers.ContainerFile entries that mirror the directory structure
// under containerTarget. Hidden files are skipped. If a path is missing,
// strict=true returns an error; strict=false logs a warning and skips.
func collectFilesForContainer(paths []string, containerTarget string, strict bool, label string, log Logger) ([]testcontainers.ContainerFile, error) {
	var out []testcontainers.ContainerFile
	for _, p := range paths {
		info, err := os.Stat(p)
		if os.IsNotExist(err) {
			if strict {
				return nil, fmt.Errorf("validate %s path %s: does not exist", label, p)
			}
			log.Printf("itestkit: %s path missing, skipping: %s", label, p)
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("validate %s path %s: %w", label, p, err)
		}
		if !info.IsDir() {
			out = append(out, testcontainers.ContainerFile{
				HostFilePath:      p,
				ContainerFilePath: filepath.Join(containerTarget, filepath.Base(p)),
				FileMode:          0o644,
			})
			continue
		}
		walkErr := filepath.WalkDir(p, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			if strings.HasPrefix(d.Name(), ".") {
				return nil
			}
			rel, err := filepath.Rel(p, path)
			if err != nil {
				return err
			}
			out = append(out, testcontainers.ContainerFile{
				HostFilePath:      path,
				ContainerFilePath: filepath.ToSlash(filepath.Join(containerTarget, rel)),
				FileMode:          0o644,
			})
			return nil
		})
		if walkErr != nil {
			return nil, fmt.Errorf("walk %s: %w", p, walkErr)
		}
	}
	return out, nil
}

// applyMockServerExpectations posts every configured HTTPExpectation to the
// running MockServer instance via /mockserver/expectation. Errors include
// the instance name and "<method> <path>" so callers can quickly identify
// the offending expectation.
func applyMockServerExpectations(ctx context.Context, opts StackOptions, res map[string]*HTTPMockResource) error {
	for name, cfg := range opts.HTTPMocks {
		r, ok := res[name]
		if !ok {
			continue
		}
		if r.Provider != HTTPMockProviderMockServer || len(cfg.Expectations) == 0 {
			continue
		}
		for _, e := range cfg.Expectations {
			if err := postMockServerExpectation(ctx, r.BaseURL, e); err != nil {
				method := e.Method
				if method == "" {
					method = "GET"
				}
				return fmt.Errorf("apply httpmock[%s] expectation %s %s: %w", name, method, e.Path, err)
			}
		}
	}
	return nil
}

// postMockServerExpectation builds the JSON body MockServer expects on
// PUT /mockserver/expectation and submits it. The function deliberately
// avoids depending on a MockServer Go client to keep the dependency
// footprint small.
func postMockServerExpectation(ctx context.Context, baseURL string, e HTTPExpectation) error {
	method := e.Method
	if method == "" {
		method = "GET"
	}
	status := e.ResponseStatus
	if status == 0 {
		status = 200
	}

	httpRequest := map[string]any{
		"method": method,
		"path":   e.Path,
	}
	if len(e.QueryParams) > 0 {
		qp := map[string][]string{}
		for k, v := range e.QueryParams {
			qp[k] = []string{v}
		}
		httpRequest["queryStringParameters"] = qp
	}
	if len(e.RequestHeaders) > 0 {
		h := map[string][]string{}
		for k, v := range e.RequestHeaders {
			h[k] = []string{v}
		}
		httpRequest["headers"] = h
	}
	if e.RequestBody != "" {
		httpRequest["body"] = e.RequestBody
	}

	httpResponse := map[string]any{
		"statusCode": status,
	}
	if len(e.ResponseHeaders) > 0 {
		h := map[string][]string{}
		for k, v := range e.ResponseHeaders {
			h[k] = []string{v}
		}
		httpResponse["headers"] = h
	}
	if e.ResponseBody != "" {
		httpResponse["body"] = e.ResponseBody
	}
	if e.Delay > 0 {
		httpResponse["delay"] = map[string]any{
			"timeUnit": "MILLISECONDS",
			"value":    e.Delay.Milliseconds(),
		}
	}

	body := map[string]any{
		"httpRequest":  httpRequest,
		"httpResponse": httpResponse,
	}
	if e.Times > 0 {
		body["times"] = map[string]any{
			"remainingTimes": e.Times,
			"unlimited":      false,
		}
	} else {
		body["times"] = map[string]any{
			"unlimited": true,
		}
	}

	buf, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal expectation: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, baseURL+"/mockserver/expectation", bytes.NewReader(buf))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("post expectation: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("mockserver returned %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return nil
}

func terminateHTTPMocks(res map[string]*HTTPMockResource, opts StackOptions) {
	if opts.Debug.KeepContainersOnFailure {
		return
	}
	for _, r := range res {
		if c, ok := r.Container.(testcontainers.Container); ok && c != nil {
			_ = testcontainers.TerminateContainer(c)
		}
	}
}
