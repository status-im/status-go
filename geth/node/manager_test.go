package node_test

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/log"
	gethnode "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
	. "github.com/status-im/status-go/geth/testing"
	"github.com/stretchr/testify/suite"
)

type ManagerTestSuite struct {
	BaseTestSuite
}

func TestNodeManager(t *testing.T) {
	suite.Run(t, new(ManagerTestSuite))
}

func (s *ManagerTestSuite) SetupTest() {
	s.NodeManager = node.NewNodeManager()
	s.Require().NotNil(s.NodeManager)
	s.Require().IsType(&node.NodeManager{}, s.NodeManager)
}

func (s *ManagerTestSuite) TestReferences() {
	s.Require().NotNil(s.NodeManager)

	var nilNodeManager *node.NodeManager

	// test for nil values of nodeManager
	var noNodeTests = []struct {
		name        string
		initFn      func() (interface{}, error)
		expectedErr error
	}{
		{
			"null manager, StartNode()",
			func() (interface{}, error) {
				return nilNodeManager.StartNode(nil)
			},
			node.ErrInvalidNodeManager,
		},
		{
			"null manager, StopNode()",
			func() (interface{}, error) {
				return nilNodeManager.StopNode()
			},
			node.ErrInvalidNodeManager,
		},
		{
			"null manager, RestartNode()",
			func() (interface{}, error) {
				return nilNodeManager.RestartNode()
			},
			node.ErrInvalidNodeManager,
		},
		{
			"null manager, ResetChainData()",
			func() (interface{}, error) {
				return nilNodeManager.ResetChainData()
			},
			node.ErrInvalidNodeManager,
		},
		{
			"null manager, IsNodeRunning()",
			func() (interface{}, error) {
				result := nilNodeManager.IsNodeRunning()
				var err error
				if !result {
					err = node.ErrInvalidNodeManager
				}
				return nil, err
			},
			node.ErrInvalidNodeManager,
		},
		{
			"null manager, PopulateStaticPeers()",
			func() (interface{}, error) {
				return nil, nilNodeManager.PopulateStaticPeers()
			},
			node.ErrInvalidNodeManager,
		},
		{
			"null manager, AddPeer()",
			func() (interface{}, error) {
				return nil, nilNodeManager.AddPeer("enode://da3bf389a031f33fb55c9f5f54fde8473912402d27fffaa50efd74c0d0515f3a61daf6d52151f2876b19c15828e6f670352bff432b5ec457652e74755e8c864f@51.15.62.116:30303")
			},
			node.ErrInvalidNodeManager,
		},
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
			"null manager, get RPC Server",
			func() (interface{}, error) {
				return nilNodeManager.RPCServer()
			},
			node.ErrInvalidNodeManager,
		},
		{
			"non-null manager, no running node, RestartNode()",
			func() (interface{}, error) {
				return s.NodeManager.RestartNode()
			},
			node.ErrNoRunningNode,
		},
		{
			"non-null manager, no running node, ResetChainData()",
			func() (interface{}, error) {
				return s.NodeManager.ResetChainData()
			},
			node.ErrNoRunningNode,
		},
		{
			"non-null manager, no running node, PopulateStaticPeers()",
			func() (interface{}, error) {
				return nil, s.NodeManager.PopulateStaticPeers()
			},
			node.ErrNoRunningNode,
		},
		{
			"non-null manager, no running node, AddPeer()",
			func() (interface{}, error) {
				return nil, s.NodeManager.AddPeer("enode://da3bf389a031f33fb55c9f5f54fde8473912402d27fffaa50efd74c0d0515f3a61daf6d52151f2876b19c15828e6f670352bff432b5ec457652e74755e8c864f@51.15.62.116:30303")
			},
			node.ErrNoRunningNode,
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
		{
			"non-null manager, no running node, get RPC Server",
			func() (interface{}, error) {
				return s.NodeManager.RPCServer()
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
		{
			"node is running, get RPC Server",
			func() (interface{}, error) {
				return s.NodeManager.RPCServer()
			},
			&rpc.Server{},
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
	_, err = s.NodeManager.StopNode()
	require.EqualError(err, node.ErrNoRunningNode.Error())

	require.False(s.NodeManager.IsNodeRunning())
	nodeStarted, err := s.NodeManager.StartNode(nodeConfig)
	require.NoError(err)

	<-nodeStarted // wait till node is started
	require.True(s.NodeManager.IsNodeRunning())

	// try starting another node (w/o stopping the previously started node)
	_, err = s.NodeManager.StartNode(nodeConfig)
	require.EqualError(err, node.ErrNodeExists.Error())

	// now stop node, and make sure that a new node, on different network can be started
	nodeStopped, err := s.NodeManager.StopNode()
	require.NoError(err)
	<-nodeStopped

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

	FirstBlockHash(require, s.NodeManager, "0x41941023680923e0fe4d74a34bdac8141f2540e3ae90623718e47d66d1ca4a2d")

	// now stop node, and make sure that a new node, on different network can be started
	nodeStopped, err := s.NodeManager.StopNode()
	require.NoError(err)
	<-nodeStopped

	// start new node with completely different config
	nodeConfig, err = MakeTestNodeConfig(params.RinkebyNetworkID)
	require.NoError(err)

	require.False(s.NodeManager.IsNodeRunning())
	nodeStarted, err = s.NodeManager.StartNode(nodeConfig)
	require.NoError(err)

	<-nodeStarted
	require.True(s.NodeManager.IsNodeRunning())

	// make sure we are on another network indeed
	FirstBlockHash(require, s.NodeManager, "0x6341fd3daf94b748c72ced5a5b26028f2474f5f00d824504e4fa37a75767e177")

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
	FirstBlockHash(require, s.NodeManager, "0x6341fd3daf94b748c72ced5a5b26028f2474f5f00d824504e4fa37a75767e177")
}

func (s *ManagerTestSuite) TestRestartNode() {
	require := s.Require()
	require.NotNil(s.NodeManager)

	s.StartTestNode(params.RinkebyNetworkID)
	defer s.StopTestNode()

	s.True(s.NodeManager.IsNodeRunning())
	nodeReady, err := s.NodeManager.RestartNode()
	require.NoError(err)
	<-nodeReady
	s.True(s.NodeManager.IsNodeRunning()) // new node, with previous config should be running

	// make sure we can read the first byte, and it is valid (for Rinkeby)
	FirstBlockHash(require, s.NodeManager, "0x6341fd3daf94b748c72ced5a5b26028f2474f5f00d824504e4fa37a75767e177")
}

func (s *ManagerTestSuite) TestRaceConditions() {
	require := s.Require()
	require.NotNil(s.NodeManager)

	cnt := 25
	progress := make(chan struct{}, cnt)
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	nodeConfig1, err := MakeTestNodeConfig(params.RopstenNetworkID)
	require.NoError(err)

	nodeConfig2, err := MakeTestNodeConfig(params.RinkebyNetworkID)
	require.NoError(err)

	nodeConfigs := []*params.NodeConfig{nodeConfig1, nodeConfig2}

	var funcsToTest = []func(*params.NodeConfig){
		func(config *params.NodeConfig) {
			log.Info("StartNode()")
			_, err := s.NodeManager.StartNode(config)
			s.T().Logf("StartNode() for network: %d, error: %v", config.NetworkID, err)
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("StopNode()")
			_, err := s.NodeManager.StopNode()
			s.T().Logf("StopNode() for network: %d, error: %v", config.NetworkID, err)
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("Node()")
			_, err := s.NodeManager.Node()
			s.T().Logf("Node(), error: %v", err)
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("IsNodeRunning()")
			s.T().Logf("IsNodeRunning(), result: %v", s.NodeManager.IsNodeRunning())
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("PopulateStaticPeers()")
			s.T().Logf("PopulateStaticPeers(), error: %v", s.NodeManager.PopulateStaticPeers())
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("ResetChainData()")
			_, err := s.NodeManager.ResetChainData()
			s.T().Logf("ResetChainData(), error: %v", err)
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("RestartNode()")
			_, err := s.NodeManager.RestartNode()
			s.T().Logf("RestartNode(), error: %v", err)
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("NodeConfig()")
			_, err := s.NodeManager.NodeConfig()
			s.T().Logf("NodeConfig(), error: %v", err)
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("LightEthereumService()")
			_, err := s.NodeManager.LightEthereumService()
			s.T().Logf("LightEthereumService(), error: %v", err)
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("WhisperService()")
			_, err := s.NodeManager.WhisperService()
			s.T().Logf("WhisperService(), error: %v", err)
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("AccountManager()")
			_, err := s.NodeManager.AccountManager()
			s.T().Logf("AccountManager(), error: %v", err)
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("AccountKeyStore()")
			_, err := s.NodeManager.AccountKeyStore()
			s.T().Logf("AccountKeyStore(), error: %v", err)
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("RPCClient()")
			_, err := s.NodeManager.RPCClient()
			s.T().Logf("RPCClient(), error: %v", err)
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("RPCServer()")
			_, err := s.NodeManager.RPCServer()
			s.T().Logf("RPCServer(), error: %v", err)
			progress <- struct{}{}
		},
	}

	// increase StartNode()/StopNode() population
	for i := 0; i < 5; i++ {
		funcsToTest = append(funcsToTest, funcsToTest[0], funcsToTest[1])
	}

	for i := 0; i < cnt; i++ {
		randConfig := nodeConfigs[rnd.Intn(len(nodeConfigs))]
		randFunc := funcsToTest[rnd.Intn(len(funcsToTest))]

		if rnd.Intn(100) > 75 { // introduce random delays
			time.Sleep(500 * time.Millisecond)
		}
		go randFunc(randConfig)
	}

	for range progress {
		cnt -= 1
		if cnt <= 0 {
			break
		}
	}

	time.Sleep(2 * time.Second)                // so that we see some logs
	nodeStopped, _ := s.NodeManager.StopNode() // just in case we have a node running
	if nodeStopped != nil {
		<-nodeStopped
	}
}

func (s *ManagerTestSuite) TestNodeStartCrash() {
	require := s.Require()
	require.NotNil(s.NodeManager)

	nodeConfig, err := MakeTestNodeConfig(params.RinkebyNetworkID)
	require.NoError(err)

	// start node outside the manager (on the same port), so that manager node.Start() method fails
	outsideNode, err := node.MakeNode(nodeConfig)
	require.NoError(outsideNode.Start())

	// let's listen for node.crashed signal
	signalReceived := false
	node.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		log.Info("Notification Received", "event", jsonEvent)
		var envelope node.SignalEnvelope
		err := json.Unmarshal([]byte(jsonEvent), &envelope)
		s.NoError(err, fmt.Sprintf("cannot unmarshal JSON: %s", jsonEvent))

		if envelope.Type == node.EventNodeCrashed {
			signalReceived = true
		}
	})

	// now try starting using node manager
	nodeStarted, err := s.NodeManager.StartNode(nodeConfig)
	require.NoError(err) // no error is thrown, as node is started in separate routine
	<-nodeStarted        // no deadlock either, as manager should close the channel on error
	require.False(s.NodeManager.IsNodeRunning())

	time.Sleep(2 * time.Second) // allow signal to propagate
	require.True(signalReceived, "node crash signal is expected")

	// stop outside node, and re-try
	require.NoError(outsideNode.Stop())
	signalReceived = false
	nodeStarted, err = s.NodeManager.StartNode(nodeConfig)
	require.NoError(err) // again, no error
	<-nodeStarted        // no deadlock, and no signal this time, manager should be able to start node
	require.True(s.NodeManager.IsNodeRunning())

	time.Sleep(2 * time.Second) // allow signal to propagate
	require.False(signalReceived, "node should start w/o crash signal")

	// cleanup
	s.NodeManager.StopNode()
	node.ResetDefaultNodeNotificationHandler()
}
