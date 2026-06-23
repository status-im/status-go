package backend

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/requests"
)

func setupConfigs() (*params.NodeConfig, *requests.APIConfig) {
	newNodeConfig := &params.NodeConfig{
		APIModules:       "test, eth, wakuv2",
		ConnectorConfig:  params.ConnectorConfig{Enabled: true},
		HTTPEnabled:      true,
		HTTPHost:         "0.0.0.0",
		HTTPPort:         8545,
		HTTPVirtualHosts: []string{"status-go"},
		WSEnabled:        false,
		WSHost:           "127.0.0.1",
		WSPort:           8586,
	}

	apiConfig := &requests.APIConfig{
		APIModules:       "connector",
		ConnectorEnabled: false,
		HTTPEnabled:      false,
		HTTPHost:         "127.0.0.1",
		HTTPPort:         8080,
		HTTPVirtualHosts: []string{"*"},
		WSEnabled:        true,
		WSHost:           "192.168.0.1",
		WSPort:           7777,
	}

	return newNodeConfig, apiConfig
}

func TestOverrideApiConfig(t *testing.T) {
	newNodeConfig, apiConfig := setupConfigs()
	overrideApiConfig(newNodeConfig, apiConfig)

	require.Equal(t, apiConfig.APIModules, newNodeConfig.APIModules)
	require.Equal(t, apiConfig.ConnectorEnabled, newNodeConfig.ConnectorConfig.Enabled)
	require.Equal(t, apiConfig.HTTPEnabled, newNodeConfig.HTTPEnabled)
	require.Equal(t, apiConfig.HTTPHost, newNodeConfig.HTTPHost)
	require.Equal(t, apiConfig.HTTPPort, newNodeConfig.HTTPPort)
	require.Equal(t, apiConfig.HTTPVirtualHosts, newNodeConfig.HTTPVirtualHosts)
	require.Equal(t, apiConfig.WSEnabled, newNodeConfig.WSEnabled)
	require.Equal(t, apiConfig.WSHost, newNodeConfig.WSHost)
	require.Equal(t, apiConfig.WSPort, newNodeConfig.WSPort)
}

func TestDefaultNodeConfig_AllowForceCommunityMembersReevaluation(t *testing.T) {
	const installationID = "test-installation"
	const keyUID = "test-key-uid"

	t.Run("enabled when env is set", func(t *testing.T) {
		t.Setenv("STATUS_ALLOW_FORCE_REEVAL", "1")

		cfg, err := DefaultNodeConfig(installationID, keyUID, &requests.CreateAccount{
			RootDataDir: t.TempDir(),
		})
		require.NoError(t, err)
		require.True(t, cfg.ShhextConfig.AllowForceCommunityMembersReevaluation)
	})

	t.Run("disabled when env is unset", func(t *testing.T) {
		t.Setenv("STATUS_ALLOW_FORCE_REEVAL", "")

		cfg, err := DefaultNodeConfig(installationID, keyUID, &requests.CreateAccount{
			RootDataDir: t.TempDir(),
		})
		require.NoError(t, err)
		require.False(t, cfg.ShhextConfig.AllowForceCommunityMembersReevaluation)
	})
}

func TestDefaultNodeConfigSetsLogosStorageDefaults(t *testing.T) {
	bootstrapNode := "test-bootstrap-node"
	request := &requests.CreateAccount{
		RootDataDir:                     t.TempDir(),
		LogosStorageConfigBootstrapNode: &bootstrapNode,
	}

	nodeConfig, err := DefaultNodeConfig("installation-id", "key-uid", request)
	require.NoError(t, err)

	require.False(t, nodeConfig.LogosStorageConfig.Enabled)
	require.Equal(t, filepath.Join(request.RootDataDir, "logos-storage", "data"), nodeConfig.LogosStorageConfig.NodeConfig.DataDir)
	require.Equal(t, params.DefaultLogosStorageBlockRetries, nodeConfig.LogosStorageConfig.NodeConfig.BlockRetries)
	require.False(t, nodeConfig.LogosStorageConfig.NodeConfig.MetricsEnabled)
	require.Equal(t, "nocolors", nodeConfig.LogosStorageConfig.NodeConfig.LogFormat)
	require.Equal(t, []string{bootstrapNode}, nodeConfig.LogosStorageConfig.NodeConfig.BootstrapNodes)
}
