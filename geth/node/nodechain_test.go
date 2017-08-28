package node_test

import (
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
	initialChainDir string
}

// Setup sets up the related entities for running this test suite.
func (nc *NodeChainTestSuite) SetupTest() {
	require := nc.Require()

	nc.NodeManager = node.NewNodeManager()
	require.NotNil(nc.NodeManager)
	require.IsType(&node.NodeManager{}, nc.NodeManager)

	initialChainDir := ".nodechain-status"

	nc.initialChainDir = initialChainDir
	require.Equal(nc.initialChainDir, initialChainDir)
}

func (nc *NodeChainTestSuite) TestInitialChainSyncWithResetChainData() {
	require := nc.Require()
	require.NotNil(nc.NodeManager)

	defer os.RemoveAll(nc.initialChainDir)
	os.MkdirAll(nc.initialChainDir, 0777)

	nodeConfig, err := MakeTestNodeConfig(params.RopstenNetworkID)
	require.NoError(err)

	nodeConfig.DataDir = nc.initialChainDir
	require.False(nc.NodeManager.IsNodeRunning())

	nodeStarted, err := nc.NodeManager.StartNode(nodeConfig)
	require.NoError(err)
	require.NotNil(nodeStarted)
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
