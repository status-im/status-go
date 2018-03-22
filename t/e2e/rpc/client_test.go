package rpc

import (
	"testing"

	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/rpc"
	e2e "github.com/status-im/status-go/t/e2e"
	. "github.com/status-im/status-go/t/utils" //nolint: golint
	"github.com/stretchr/testify/suite"
)

type RPCClientTestSuite struct {
	e2e.NodeManagerTestSuite
}

func TestRPCClientTestSuite(t *testing.T) {
	suite.Run(t, new(RPCClientTestSuite))
}

func (s *RPCClientTestSuite) SetupTest() {
	s.NodeManager = node.NewNodeManager()
	s.NotNil(s.NodeManager)
}

func (s *RPCClientTestSuite) TestNewClient() {
	config, err := MakeTestNodeConfig(GetNetworkID())
	s.NoError(err)

	// upstream disabled
	s.False(config.UpstreamConfig.Enabled)
	_, err = rpc.NewClient(nil, config.UpstreamConfig)
	s.NoError(err)

	// upstream enabled with correct URL
	upstreamGood := config.UpstreamConfig
	upstreamGood.Enabled = true
	upstreamGood.URL = "http://example.com/rpc"
	_, err = rpc.NewClient(nil, upstreamGood)
	s.NoError(err)

	// upstream enabled with incorrect URL
	upstreamBad := config.UpstreamConfig
	upstreamBad.Enabled = true
	upstreamBad.URL = "///__httphh://///incorrect_urlxxx"
	_, err = rpc.NewClient(nil, upstreamBad)
	s.Error(err)
}
