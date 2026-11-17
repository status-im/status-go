package params

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFallbackProxyStages(t *testing.T) {
	require.Equal(t, []string{"test"}, FallbackProxyStages("prod", ""))

	require.Empty(t, FallbackProxyStages("prod", "https://proxy.example.org"))
	for _, stageName := range []string{"test", "staging", "production", ""} {
		require.Empty(t, FallbackProxyStages(stageName, ""), "stage %q", stageName)
	}
}
