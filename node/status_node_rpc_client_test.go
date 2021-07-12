package node

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/params"
)

type TestServiceAPI struct{}

func (api *TestServiceAPI) SomeMethod(_ context.Context) (string, error) {
	return "some method result", nil
}

func createAndStartStatusNode(config *params.NodeConfig) (*StatusNode, error) {
	statusNode := New()
	err := statusNode.Start(config, nil)
	if err != nil {
		return nil, err
	}
	return statusNode, nil
}

func TestNodeRPCClientCallOnlyPublicAPIs(t *testing.T) {
	var err error

	statusNode, err := createAndStartStatusNode(&params.NodeConfig{
		APIModules: "", // no whitelisted API modules; use only public APIs
		UpstreamConfig: params.UpstreamRPCConfig{
			URL:     "https://infura.io",
			Enabled: true},
		WakuConfig: params.WakuConfig{
			Enabled: true,
		},
	})
	require.NoError(t, err)
	defer func() {
		err := statusNode.Stop()
		require.NoError(t, err)
	}()

	client := statusNode.RPCClient()
	require.NotNil(t, client)

	// call public API with public RPC Client
	result, err := statusNode.CallRPC(`{"jsonrpc": "2.0", "id": 1, "method": "eth_uninstallFilter", "params": ["id"]}`)
	require.NoError(t, err)

	// the call is successful
	require.False(t, strings.Contains(result, "error"))

	result, err = statusNode.CallRPC(`{"jsonrpc": "2.0", "id": 1, "method": "waku_info"}`)
	require.NoError(t, err)

	// call private API with public RPC client
	require.Equal(t, ErrRPCMethodUnavailable, result)

}

func TestNodeRPCPrivateClientCallPrivateService(t *testing.T) {
	var err error

	statusNode, err := createAndStartStatusNode(&params.NodeConfig{
		WakuConfig: params.WakuConfig{
			Enabled: true,
		},
	})
	require.NoError(t, err)
	defer func() {
		err := statusNode.Stop()
		require.NoError(t, err)
	}()

	result, err := statusNode.CallPrivateRPC(`{"jsonrpc": "2.0", "id": 1, "method": "waku_info"}`)
	require.NoError(t, err)

	// the call is successful
	require.False(t, strings.Contains(result, "error"))
}
