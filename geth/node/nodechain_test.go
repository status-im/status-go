package node_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
	. "github.com/status-im/status-go/geth/testing"
	"github.com/stretchr/testify/suite"
)

// TestNodeChainTestSuite runs tests associated with the NodeChainTestSuite.
func TestNodeChainTestSuite(t *testing.T) {
	suite.Run(t, new(NodeChainTestSuite))
}

// NodeChainTestSuite defines a struct which holds tests related to Node synchronization
// with it's chain data directory.
type NodeChainTestSuite struct {
	BaseTestSuite
	chainDir string
}

// Setup sets up the related entities for running this test suite.
func (s *NodeChainTestSuite) SetupTest() {
	require := s.Require()

	s.NodeManager = node.NewNodeManager()

	chainDir, err := ioutil.TempDir("", "chainDir")
	require.NoError(err)
	s.chainDir = chainDir
}

func (s *NodeChainTestSuite) TearDownTest() {
	require := s.Require()
	err := os.RemoveAll(s.chainDir)
	require.NoError(err)
}

func (s *NodeChainTestSuite) TestResetChainData() {
	require := s.Require()
	require.NotNil(s.NodeManager)

	nodeConfig, err := MakeTestNodeConfig(params.RopstenNetworkID)
	require.NoError(err)

	nodeConfig.DataDir = s.chainDir
	require.False(s.NodeManager.IsNodeRunning())

	nodeStarted, err := s.NodeManager.StartNode(nodeConfig)
	require.NoError(err)
	<-nodeStarted
	require.True(s.NodeManager.IsNodeRunning())

	s.EnsureNodeSync()

	ready, resetErr := s.NodeManager.ResetChainData()
	require.NoError(resetErr)
	require.NotNil(ready)
	<-ready
	require.True(s.NodeManager.IsNodeRunning())

	s.EnsureNodeSync(true)
}
