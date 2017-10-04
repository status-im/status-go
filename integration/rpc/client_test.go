package rpc

import (
	"testing"

	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/rpc"
	"github.com/status-im/status-go/integration"
	"github.com/stretchr/testify/suite"
)

type RPCClientTestSuite struct {
	integration.NodeManagerTestSuite
}

func TestRPCClientTestSuite(t *testing.T) {
	suite.Run(t, new(RPCClientTestSuite))
}

func (s *RPCClientTestSuite) SetupTest() {
	s.NodeManager = node.NewNodeManager()
	s.NotNil(s.NodeManager)
}

func (s *RPCClientTestSuite) TestNewClient() {
	config, err := integration.MakeTestNodeConfig(params.RinkebyNetworkID)
	s.NoError(err)

	nodeStarted, err := s.NodeManager.StartNode(config)
	s.NoError(err)
	<-nodeStarted

	node, err := s.NodeManager.Node()
	s.NoError(err)

	// upstream disabled, local node ok
	s.False(config.UpstreamConfig.Enabled)
	_, err = rpc.NewClient(node, config.UpstreamConfig)
	s.NoError(err)

	// upstream enabled with incorrect URL, local node ok
	upstreamBad := config.UpstreamConfig
	upstreamBad.Enabled = true
	upstreamBad.URL = "///__httphh://///incorrect_urlxxx"
	_, err = rpc.NewClient(node, upstreamBad)
	s.Error(err)

	// upstream enabled with correct URL, local node ok
	upstreamGood := config.UpstreamConfig
	upstreamGood.Enabled = true
	upstreamGood.URL = "http://example.com/rpc"
	_, err = rpc.NewClient(node, upstreamGood)
	s.NoError(err)

	// upstream disabled, local node failed (stopped)
	nodeStopped, err := s.NodeManager.StopNode()
	s.NoError(err)
	<-nodeStopped

	_, err = rpc.NewClient(node, config.UpstreamConfig)
	s.Error(err)
}
