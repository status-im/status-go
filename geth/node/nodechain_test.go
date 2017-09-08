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
func (nc *NodeChainTestSuite) SetupTest() {
	require := nc.Require()

	nc.NodeManager = node.NewNodeManager()

	chainDir, err := ioutil.TempDir("", "chainDir")
	require.NoError(err)
	nc.chainDir = chainDir
}

func (nc *NodeChainTestSuite) TearDownTest() {
	require := nc.Require()
	err := os.RemoveAll(nc.chainDir)
	require.NoError(err)
}

func (nc *NodeChainTestSuite) TestResetChainData() {
	require := nc.Require()
	require.NotNil(nc.NodeManager)

	nodeConfig, err := MakeTestNodeConfig(params.RopstenNetworkID)
	require.NoError(err)

	nodeConfig.DataDir = nc.chainDir
	require.False(nc.NodeManager.IsNodeRunning())

	nodeStarted, err := nc.NodeManager.StartNode(nodeConfig)
	require.NoError(err)
	<-nodeStarted
	require.True(nc.NodeManager.IsNodeRunning())

	nc.EnsureNodeSync()

	ready, resetErr := nc.NodeManager.ResetChainData()
	require.NoError(resetErr)
	require.NotNil(ready)
	<-ready
	require.True(nc.NodeManager.IsNodeRunning())

	nc.EnsureNodeSync()
}
