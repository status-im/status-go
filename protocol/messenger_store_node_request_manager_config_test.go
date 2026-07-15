package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultStoreNodeRequestConfig(t *testing.T) {
	cfg := defaultStoreNodeRequestConfig()
	require.Greater(t, cfg.MaxPageCount, 1, "cap must be enabled by default")
	require.GreaterOrEqual(t, cfg.MaxPageCount, 20, "cap must be generous vs the 1-2 page normal fetch")
	require.True(t, cfg.StopWhenDataFound)
	require.True(t, cfg.WaitForResponse)
}
