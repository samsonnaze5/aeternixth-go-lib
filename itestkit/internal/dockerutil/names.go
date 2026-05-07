// Package dockerutil produces deterministic, collision-resistant identifiers
// for Docker resources (networks, container labels) created by itestkit.
// It is internal to itestkit.
package dockerutil

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// NetworkName builds a network name of the form "<project>-net-<suffix>".
// The suffix is a random 8-character hex string so concurrent test runs
// against the same project name do not collide on the shared Docker host.
func NetworkName(project string) string {
	if project == "" {
		project = "itest"
	}
	return fmt.Sprintf("%s-net-%s", project, randomSuffix(4))
}

// randomSuffix returns a hex-encoded random suffix of the requested byte
// length. It falls back to a fixed string if the platform random source is
// unavailable, which is acceptable for ephemeral test resources.
func randomSuffix(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "fallback"
	}
	return hex.EncodeToString(buf)
}
