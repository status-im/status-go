package node_test

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/les"
	gethnode "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
	. "github.com/status-im/status-go/geth/testing"
	"github.com/stretchr/testify/suite"
)

func TestManagerTestSuite(t *testing.T) {
	suite.Run(t, new(ManagerTestSuite))
}

type ManagerTestSuite struct {
	BaseTestSuite
}

func (s *ManagerTestSuite) SetupTest() {
	s.NodeManager = node.NewNodeManager()
	s.Require().NotNil(s.NodeManager)
	s.Require().IsType(&node.NodeManager{}, s.NodeManager)
}

func (s *ManagerTestSuite) TestGettingReferencedServices() {
	s.Require().NotNil(s.NodeManager)

	var nilNodeManager *node.NodeManager

	// test for nil values of nodeManager
	var noNodeTests = []struct {
		name        string
		initFn      func() (interface{}, error)
		expectedErr error
	}{
		{
			"null manager, get NodeConfig",
			func() (interface{}, error) {
				return nilNodeManager.NodeConfig()
			},
			node.ErrInvalidNodeManager,
		},
		{
			"null manager, get Node",
			func() (interface{}, error) {
				return nilNodeManager.Node()
			},
			node.ErrInvalidNodeManager,
		},
		{
			"null manager, get LES",
			func() (interface{}, error) {
				return nilNodeManager.LightEthereumService()
			},
			node.ErrInvalidNodeManager,
		},
		{
			"null manager, get Whisper",
			func() (interface{}, error) {
				return nilNodeManager.WhisperService()
			},
			node.ErrInvalidNodeManager,
		},
		{
			"null manager, get AccountManager",
			func() (interface{}, error) {
				return nilNodeManager.AccountManager()
			},
			node.ErrInvalidNodeManager,
		},
		{
			"null manager, get AccountKeyStore",
			func() (interface{}, error) {
				return nilNodeManager.AccountKeyStore()
			},
			node.ErrInvalidNodeManager,
		},
		{
			"null manager, get RPC Client",
			func() (interface{}, error) {
				return nilNodeManager.RPCClient()
			},
			node.ErrInvalidNodeManager,
		},
		{
			"non-null manager, no running node, get NodeConfig",
			func() (interface{}, error) {
				return s.NodeManager.NodeConfig()
			},
			node.ErrNoRunningNode,
		},
		{
			"non-null manager, no running node, get Node",
			func() (interface{}, error) {
				return s.NodeManager.Node()
			},
			node.ErrNoRunningNode,
		},
		{
			"non-null manager, no running node, get LES",
			func() (interface{}, error) {
				return s.NodeManager.LightEthereumService()
			},
			node.ErrNoRunningNode,
		},
		{
			"non-null manager, no running node, get Whisper",
			func() (interface{}, error) {
				return s.NodeManager.WhisperService()
			},
			node.ErrNoRunningNode,
		},
		{
			"non-null manager, no running node, get AccountManager",
			func() (interface{}, error) {
				return s.NodeManager.AccountManager()
			},
			node.ErrNoRunningNode,
		},
		{
			"non-null manager, no running node, get AccountKeyStore",
			func() (interface{}, error) {
				return s.NodeManager.AccountKeyStore()
			},
			node.ErrNoRunningNode,
		},
		{
			"non-null manager, no running node, get RPC Client",
			func() (interface{}, error) {
				return s.NodeManager.RPCClient()
			},
			node.ErrNoRunningNode,
		},
	}
	for _, testCase := range noNodeTests {
		s.T().Log(testCase.name)
		obj, err := testCase.initFn()
		s.Nil(obj)
		s.EqualError(err, testCase.expectedErr.Error())
	}

	// test with node fully started
	s.StartTestNode(params.RinkebyNetworkID)
	defer s.StopTestNode()
	var nodeReadyTestCases = []struct {
		name         string
		initFn       func() (interface{}, error)
		expectedType interface{}
	}{
		{
			"node is running, get NodeConfig",
			func() (interface{}, error) {
				return s.NodeManager.NodeConfig()
			},
			&params.NodeConfig{},
		},
		{
			"node is running, get Node",
			func() (interface{}, error) {
				return s.NodeManager.Node()
			},
			&gethnode.Node{},
		},
		{
			"node is running, get LES",
			func() (interface{}, error) {
				return s.NodeManager.LightEthereumService()
			},
			&les.LightEthereum{},
		},
		{
			"node is running, get Whisper",
			func() (interface{}, error) {
				return s.NodeManager.WhisperService()
			},
			&whisper.Whisper{},
		},
		{
			"node is running, get AccountManager",
			func() (interface{}, error) {
				return s.NodeManager.AccountManager()
			},
			&accounts.Manager{},
		},
		{
			"node is running, get AccountKeyStore",
			func() (interface{}, error) {
				return s.NodeManager.AccountKeyStore()
			},
			&keystore.KeyStore{},
		},
		{
			"node is running, get RPC Client",
			func() (interface{}, error) {
				return s.NodeManager.RPCClient()
			},
			&rpc.Client{},
		},
	}
	for _, testCase := range nodeReadyTestCases {
		obj, err := testCase.initFn()
		s.T().Log(testCase.name)
		s.NoError(err)
		s.NotNil(obj)
		s.IsType(testCase.expectedType, obj)
	}
}

func (s *ManagerTestSuite) TestNodeStartStop() {
	require := s.Require()
	require.NotNil(s.NodeManager)

	nodeConfig, err := MakeTestNodeConfig(params.RopstenNetworkID)
	require.NoError(err)

	// try stopping non-started node
	require.False(s.NodeManager.IsNodeRunning())
	err = s.NodeManager.StopNode()
	if s.Error(err) {
		require.IsType(node.ErrNoRunningNode, err)
	}

	require.False(s.NodeManager.IsNodeRunning())
	nodeStarted, err := s.NodeManager.StartNode(nodeConfig)
	require.NoError(err)

	<-nodeStarted // wait till node is started
	require.True(s.NodeManager.IsNodeRunning())

	// try starting another node (w/o stopping the previously started node)
	_, err = s.NodeManager.StartNode(nodeConfig)
	if s.Error(err) {
		require.IsType(node.ErrNodeAlreadyExists, err)
	}

	// now stop node, and make sure that a new node, on different network can be started
	err = s.NodeManager.StopNode()
	require.NoError(err)

	// start new node with exactly the same config
	require.False(s.NodeManager.IsNodeRunning())
	nodeStarted, err = s.NodeManager.StartNode(nodeConfig)
	require.NoError(err)

	<-nodeStarted
	require.True(s.NodeManager.IsNodeRunning())

	s.StopTestNode()
}

func (s *ManagerTestSuite) TestNetworkSwitching() {
	require := s.Require()
	require.NotNil(s.NodeManager)

	// get Ropsten config
	nodeConfig, err := MakeTestNodeConfig(params.RopstenNetworkID)
	require.NoError(err)

	require.False(s.NodeManager.IsNodeRunning())
	nodeStarted, err := s.NodeManager.StartNode(nodeConfig)
	require.NoError(err)

	<-nodeStarted // wait till node is started
	require.True(s.NodeManager.IsNodeRunning())

	s.FirstBlockHash("0x41941023680923e0fe4d74a34bdac8141f2540e3ae90623718e47d66d1ca4a2d")

	// now stop node, and make sure that a new node, on different network can be started
	err = s.NodeManager.StopNode()
	require.NoError(err)

	// start new node with completely different config
	nodeConfig, err = MakeTestNodeConfig(params.RinkebyNetworkID)
	require.NoError(err)

	require.False(s.NodeManager.IsNodeRunning())
	nodeStarted, err = s.NodeManager.StartNode(nodeConfig)
	require.NoError(err)

	<-nodeStarted
	require.True(s.NodeManager.IsNodeRunning())

	// make sure we are on another network indeed
	s.FirstBlockHash("0x6341fd3daf94b748c72ced5a5b26028f2474f5f00d824504e4fa37a75767e177")

	s.StopTestNode()
}

func (s *ManagerTestSuite) TestResetChainData() {
	require := s.Require()
	require.NotNil(s.NodeManager)

	s.StartTestNode(params.RinkebyNetworkID)
	defer s.StopTestNode()

	time.Sleep(2 * time.Second) // allow to sync for some time

	s.True(s.NodeManager.IsNodeRunning())
	nodeReady, err := s.NodeManager.ResetChainData()
	require.NoError(err)
	<-nodeReady
	s.True(s.NodeManager.IsNodeRunning()) // new node, with previous config should be running

	// make sure we can read the first byte, and it is valid (for Rinkeby)
	s.FirstBlockHash("0x6341fd3daf94b748c72ced5a5b26028f2474f5f00d824504e4fa37a75767e177")
}
