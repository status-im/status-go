//go:build codex_integration
// +build codex_integration

package communities

import "os"

// GetEnvOrDefault returns the value of the environment variable k,
// or def if the variable is not set or is empty.
// This helper is shared across all integration test files.
// It's exported so it can be used by tests in the communities_test package.
func GetEnvOrDefault(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
