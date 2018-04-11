package node_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	gethnode "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
	. "github.com/status-im/status-go/t/utils"
)

type TestServiceAPI struct{}

func (api *TestServiceAPI) SomeMethod(_ context.Context) (string, error) {
	return "some method result", nil
}

type testService struct{}

func (s *testService) Protocols() []p2p.Protocol {
	return []p2p.Protocol{}
}

func (s *testService) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "pri",
			Version:   "1.0",
			Service:   &TestServiceAPI{},
			Public:    false,
		},
		{
			Namespace: "pub",
			Version:   "1.0",
			Service:   &TestServiceAPI{},
			Public:    true,
		},
	}
}

func (s *testService) Start(server *p2p.Server) error {
	return nil
}

func (s *testService) Stop() error {
	return nil
}

func createStatusNode(config *params.NodeConfig) (*node.StatusNode, error) {
	services := []gethnode.ServiceConstructor{
		func(_ *gethnode.ServiceContext) (gethnode.Service, error) {
			return &testService{}, nil
		},
	}
	statusNode := node.New()
	return statusNode, statusNode.Start(config, services...)
}

func TestNodeRPCClientCallOnlyPublicAPIs(t *testing.T) {
	var err error

	config, err := MakeTestNodeConfig(GetNetworkID())
	require.NoError(t, err)
	config.APIModules = "" // no whitelisted API modules; use only public APIs

	statusNode, err := createStatusNode(config)
	require.NoError(t, err)
	defer func() {
		err := statusNode.Stop()
		require.NoError(t, err)
	}()

	client := statusNode.RPCClient()
	require.NotNil(t, client)

	var result string

	// call public API
	err = client.Call(&result, "pub_someMethod")
	require.NoError(t, err)
	require.Equal(t, "some method result", result)

	// call private API with public RPC client
	err = client.Call(&result, "pri_someMethod")
	require.EqualError(t, err, "The method pri_someMethod does not exist/is not available")
}

func TestNodeRPCClientCallWhitelistedPrivateService(t *testing.T) {
	var err error

	config, err := MakeTestNodeConfig(GetNetworkID())
	require.NoError(t, err)
	config.APIModules = "pri"

	statusNode, err := createStatusNode(config)
	require.NoError(t, err)
	defer func() {
		err := statusNode.Stop()
		require.NoError(t, err)
	}()

	client := statusNode.RPCClient()
	require.NotNil(t, client)

	// call private API
	var result string
	err = client.Call(&result, "pri_someMethod")
	require.NoError(t, err)
	require.Equal(t, "some method result", result)
}

func TestNodeRPCPrivateClientCallPrivateService(t *testing.T) {
	var err error

	config, err := MakeTestNodeConfig(GetNetworkID())
	require.NoError(t, err)

	statusNode, err := createStatusNode(config)
	require.NoError(t, err)
	defer func() {
		err := statusNode.Stop()
		require.NoError(t, err)
	}()

	client := statusNode.RPCPrivateClient()
	require.NotNil(t, client)

	// call private API with private RPC client
	var result string
	err = client.Call(&result, "pri_someMethod")
	require.NoError(t, err)
	require.Equal(t, "some method result", result)
}
