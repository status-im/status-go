package api

import (
	"testing"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/requests"

	"github.com/stretchr/testify/require"
)

func TestOverrideApiConfig(t *testing.T) {
	newNodeConfig := &params.NodeConfig{
		APIModules:      "test, eth, wakuv2",
		ConnectorConfig: params.ConnectorConfig{Enabled: true},
		HTTPEnabled:     true,
		HTTPHost:        "0.0.0.0",
		HTTPPort:        8545,
		WSEnabled:       false,
		WSHost:          "127.0.0.1",
		WSPort:          8586,
	}

	apiConfig := &requests.APIConfig{
		APIModules:       "connector",
		ConnectorEnabled: false,
		HTTPEnabled:      false,
		HTTPHost:         "127.0.0.1",
		HTTPPort:         8080,
		WSEnabled:        true,
		WSHost:           "192.168.0.1",
		WSPort:           7777,
	}

	overrideApiConfig(newNodeConfig, apiConfig)

	require.Equal(t, apiConfig.APIModules, newNodeConfig.APIModules)
	require.Equal(t, apiConfig.ConnectorEnabled, newNodeConfig.ConnectorConfig.Enabled)
	require.Equal(t, apiConfig.HTTPEnabled, newNodeConfig.HTTPEnabled)
	require.Equal(t, apiConfig.HTTPHost, newNodeConfig.HTTPHost)
	require.Equal(t, apiConfig.HTTPPort, newNodeConfig.HTTPPort)
	require.Equal(t, apiConfig.WSEnabled, newNodeConfig.WSEnabled)
	require.Equal(t, apiConfig.WSHost, newNodeConfig.WSHost)
	require.Equal(t, apiConfig.WSPort, newNodeConfig.WSPort)
}
