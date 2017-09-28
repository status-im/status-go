package node_test

import (
	"encoding/json"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/les"
	gethnode "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/geth/log"
	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/signal"
	"github.com/status-im/status-go/integration"
	"github.com/stretchr/testify/suite"
)

func TestManagerTestSuite(t *testing.T) {
	suite.Run(t, new(ManagerTestSuite))
}

type ManagerTestSuite struct {
	integration.NodeManagerTestSuite
}

func (s *ManagerTestSuite) SetupTest() {
	s.NodeManager = node.NewNodeManager()
}

func (s *ManagerTestSuite) TestReferencesWithoutStartedNode() {
	var testCases = []struct {
		name        string
		initFn      func() (interface{}, error)
		expectedErr error
	}{
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
				return s.NodeManager.RPCClient(), nil
			},
			nil,
		},
	}
	for _, tc := range testCases {
		s.T().Log(tc.name)
		obj, err := tc.initFn()
		s.Nil(obj)
		s.Equal(tc.expectedErr, err)
	}
}

func (s *ManagerTestSuite) TestReferencesWithStartedNode() {
	s.StartTestNode(params.RinkebyNetworkID)
	defer s.StopTestNode()

	var testCases = []struct {
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
				return s.NodeManager.RPCClient(), nil
			},
			&rpc.Client{},
		},
	}
	for _, tc := range testCases {
		s.T().Log(tc.name)
		obj, err := tc.initFn()
		s.NoError(err)
		s.NotNil(obj)
		s.IsType(tc.expectedType, obj)
	}
}

func (s *ManagerTestSuite) TestNodeStartStop() {
	nodeConfig, err := integration.MakeTestNodeConfig(params.RopstenNetworkID)
	s.NoError(err)

	// try stopping non-started node
	s.False(s.NodeManager.IsNodeRunning())
	_, err = s.NodeManager.StopNode()
	s.Equal(err, node.ErrNoRunningNode)

	// start node
	s.False(s.NodeManager.IsNodeRunning())
	nodeStarted, err := s.NodeManager.StartNode(nodeConfig)
	s.NoError(err)
	// wait till node is started
	<-nodeStarted
	s.True(s.NodeManager.IsNodeRunning())

	// try starting another node (w/o stopping the previously started node)
	_, err = s.NodeManager.StartNode(nodeConfig)
	s.Equal(err, node.ErrNodeExists)

	// now stop node
	nodeStopped, err := s.NodeManager.StopNode()
	s.NoError(err)
	<-nodeStopped
	s.False(s.NodeManager.IsNodeRunning())

	// start new node with exactly the same config
	nodeStarted, err = s.NodeManager.StartNode(nodeConfig)
	s.NoError(err)
	// wait till node is started
	<-nodeStarted
	s.True(s.NodeManager.IsNodeRunning())

	// finally stop the node
	nodeStopped, err = s.NodeManager.StopNode()
	s.NoError(err)
	<-nodeStopped
}

func (s *ManagerTestSuite) TestNetworkSwitching() {
	// get Ropsten config
	nodeConfig, err := integration.MakeTestNodeConfig(params.RopstenNetworkID)
	s.NoError(err)
	s.False(s.NodeManager.IsNodeRunning())
	nodeStarted, err := s.NodeManager.StartNode(nodeConfig)
	s.NoError(err)
	// wait till node is started
	<-nodeStarted
	s.True(s.NodeManager.IsNodeRunning())

	firstHash, err := integration.FirstBlockHash(s.NodeManager)
	s.NoError(err)
	s.Equal("0x41941023680923e0fe4d74a34bdac8141f2540e3ae90623718e47d66d1ca4a2d", firstHash)

	// now stop node, and make sure that a new node, on different network can be started
	nodeStopped, err := s.NodeManager.StopNode()
	s.NoError(err)
	<-nodeStopped
	s.False(s.NodeManager.IsNodeRunning())

	// start new node with completely different config
	nodeConfig, err = integration.MakeTestNodeConfig(params.RinkebyNetworkID)
	s.NoError(err)
	nodeStarted, err = s.NodeManager.StartNode(nodeConfig)
	s.NoError(err)
	// wait till node is started
	<-nodeStarted
	s.True(s.NodeManager.IsNodeRunning())

	// make sure we are on another network indeed
	firstHash, err = integration.FirstBlockHash(s.NodeManager)
	s.NoError(err)
	s.Equal("0x6341fd3daf94b748c72ced5a5b26028f2474f5f00d824504e4fa37a75767e177", firstHash)

	nodeStopped, err = s.NodeManager.StopNode()
	s.NoError(err)
	<-nodeStopped
}

func (s *ManagerTestSuite) TestStartNodeWithUpstreamEnabled() {
	nodeConfig, err := integration.MakeTestNodeConfig(params.RopstenNetworkID)
	s.NoError(err)

	nodeConfig.UpstreamConfig.Enabled = true
	nodeConfig.UpstreamConfig.URL = "https://ropsten.infura.io/nKmXgiFgc2KqtoQ8BCGJ"

	nodeStarted, err := s.NodeManager.StartNode(nodeConfig)
	s.NoError(err)
	<-nodeStarted
	s.True(s.NodeManager.IsNodeRunning())
	nodeStopped, err := s.NodeManager.StopNode()
	s.NoError(err)
	<-nodeStopped
}

// TODO(adam): fix this test to not use a different directory for blockchain data
func (s *ManagerTestSuite) TestResetChainData() {
	s.T().Skip()

	s.StartTestNode(params.RinkebyNetworkID)
	defer s.StopTestNode()

	// allow to sync for some time
	time.Sleep(10 * time.Second)

	// reset chain data
	nodeReady, err := s.NodeManager.ResetChainData()
	s.NoError(err)
	// new node, with previous config should be running
	<-nodeReady
	s.True(s.NodeManager.IsNodeRunning())

	// make sure we can read the first byte, and it is valid (for Rinkeby)
	firstHash, err := integration.FirstBlockHash(s.NodeManager)
	s.NoError(err)
	s.Equal("0x6341fd3daf94b748c72ced5a5b26028f2474f5f00d824504e4fa37a75767e177", firstHash)
}

func (s *ManagerTestSuite) TestRestartNode() {
	s.StartTestNode(params.RinkebyNetworkID)
	defer s.StopTestNode()

	s.True(s.NodeManager.IsNodeRunning())
	nodeReady, err := s.NodeManager.RestartNode()
	s.NoError(err)
	// new node, with previous config should be running
	<-nodeReady
	s.True(s.NodeManager.IsNodeRunning())

	// make sure we can read the first byte, and it is valid (for Rinkeby)
	firstHash, err := integration.FirstBlockHash(s.NodeManager)
	s.NoError(err)
	s.Equal("0x6341fd3daf94b748c72ced5a5b26028f2474f5f00d824504e4fa37a75767e177", firstHash)
}

// TODO(adam): race conditions should be tested with -race flag and unit tests, if possible.
// Research if it's possible to do the same with unit tests.
func (s *ManagerTestSuite) TestRaceConditions() {
	cnt := 25
	progress := make(chan struct{}, cnt)
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	nodeConfig1, err := integration.MakeTestNodeConfig(params.RopstenNetworkID)
	s.NoError(err)

	nodeConfig2, err := integration.MakeTestNodeConfig(params.RinkebyNetworkID)
	s.NoError(err)

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
		// func(config *params.NodeConfig) {
		// 	log.Info("ResetChainData()")
		// 	_, err := s.NodeManager.ResetChainData()
		// 	s.T().Logf("ResetChainData(), error: %v", err)
		// 	progress <- struct{}{}
		// },
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
			s.NodeManager.RPCClient()
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
	// let's listen for node.crashed signal
	signalReceived := make(chan struct{})
	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope signal.Envelope
		err := json.Unmarshal([]byte(jsonEvent), &envelope)
		s.NoError(err)

		if envelope.Type == signal.EventNodeCrashed {
			close(signalReceived)
		}
	})

	nodeConfig, err := integration.MakeTestNodeConfig(params.RinkebyNetworkID)
	s.NoError(err)

	// start node outside the manager (on the same port), so that manager node.Start() method fails
	outsideNode, err := node.MakeNode(nodeConfig)
	s.NoError(err)
	err = outsideNode.Start()
	s.NoError(err)

	// now try starting using node manager
	nodeStarted, err := s.NodeManager.StartNode(nodeConfig)
	s.NoError(err) // no error is thrown, as node is started in separate routine
	<-nodeStarted  // no deadlock either, as manager should close the channel on error
	s.False(s.NodeManager.IsNodeRunning())

	select {
	case <-time.After(5 * time.Second):
		s.FailNow("timed out waiting for signal")
	case <-signalReceived:
	}

	// stop outside node, and re-try
	err = outsideNode.Stop()
	s.NoError(err)
	signalReceived = make(chan struct{})
	nodeStarted, err = s.NodeManager.StartNode(nodeConfig)
	s.NoError(err) // again, no error
	<-nodeStarted  // no deadlock, and no signal this time, manager should be able to start node
	s.True(s.NodeManager.IsNodeRunning())

	select {
	case <-time.After(5 * time.Second):
	case <-signalReceived:
		s.FailNow("signal should not be received")
	}

	// cleanup
	s.NodeManager.StopNode()
	signal.ResetDefaultNodeNotificationHandler()
}
