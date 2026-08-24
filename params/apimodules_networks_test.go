package params_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/params"
)

func TestNewNodeConfigAPIModulesIncludesNetworks(t *testing.T) {
	cfg, err := params.NewNodeConfig(t.TempDir(), 1)
	require.NoError(t, err)
	require.Contains(t, cfg.FormatAPIModules(), "networks")
}
