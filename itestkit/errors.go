package itestkit

import "errors"

// Sentinel errors returned by validation. Callers can match against them
// using errors.Is for programmatic handling in self-tests.
var (
	// ErrNoServiceConfigured is returned when StackOptions contains no
	// service maps at all (or all are empty).
	ErrNoServiceConfigured = errors.New("itestkit: no service configured")

	// ErrInvalidInstanceName is returned when an instance name fails the
	// regex check in instance-name validation.
	ErrInvalidInstanceName = errors.New("itestkit: invalid instance name")

	// ErrDuplicateInstanceName is returned when the same instance name
	// appears twice within one service map. Map keys are unique by Go's
	// language guarantees, but this error is returned for clarity if a
	// future API supports lists.
	ErrDuplicateInstanceName = errors.New("itestkit: duplicate instance name")

	// ErrInvalidHTTPMockProvider is returned when an HTTPMock instance
	// is configured with a provider other than mockserver or wiremock.
	ErrInvalidHTTPMockProvider = errors.New("itestkit: invalid http mock provider")

	// ErrWireMockExpectations is returned when a WireMock instance is
	// configured with HTTPExpectations. v1.1 supports expectations only
	// for MockServer; WireMock consumes file-based mappings instead.
	ErrWireMockExpectations = errors.New("itestkit: wiremock provider does not accept expectations in v1.1")
)
