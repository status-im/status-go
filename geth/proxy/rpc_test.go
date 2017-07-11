package proxy_test

import (
	"testing"

	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/proxy"
	. "github.com/status-im/status-go/geth/testing"
	"github.com/stretchr/testify/suite"
)

func TestRPCRouterTestSuite(t *testing.T) {
	suite.Run(t, new(RPCRouterTestSuite))
}

type RPCRouterTestSuite struct {
	BaseTestSuite
}

func (s *RPCRouterTestSuite) SetupTest() {
	s.NodeManager = proxy.NewRPCRouter(node.NewNodeManager())

	s.Require().NotNil(s.NodeManager)
	s.Require().IsType(&proxy.RPCRouter{}, s.NodeManager)
}

func (s *RPCRouterTestSuite) TestRPCClientConnection() {
	require := s.Require()
	require.NotNil(s.NodeManager)

	//TODO(alex): How do we validate whether the client we
	// receive is actually from a upstrem or is from the internally
	// started server.
	// For now validate config state.

	nodeConfig, err := MakeTestNodeConfig(params.RopstenNetworkID)
	require.NoError(err)

	// validate default state of UpstreamConfig.Enable.
	require.NotEqual(nodeConfig.UpstreamConfig.Enabled, true)
	require.NotEmpty(nodeConfig.UpstreamConfig.URL)
	require.Equal(nodeConfig.UpstreamConfig.URL, params.UpstreamRopstenEthereumNetworkURL)
}
