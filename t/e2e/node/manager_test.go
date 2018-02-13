package node_test

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/les"
	gethnode "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"

	e2e "github.com/status-im/status-go/t/e2e"
	. "github.com/status-im/status-go/t/utils"
	"github.com/stretchr/testify/suite"
)

func TestManagerTestSuite(t *testing.T) {
	suite.Run(t, new(ManagerTestSuite))
}

type ManagerTestSuite struct {
	e2e.NodeManagerTestSuite
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
	s.StartTestNode()
	defer func() {
		time.Sleep(100 * time.Millisecond)
		s.StopTestNode()
	}()

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
	nodeConfig, err := e2e.MakeTestNodeConfig(GetNetworkID())
	s.NoError(err)

	// try stopping non-started node
	s.False(s.NodeManager.IsNodeRunning())
	time.Sleep(100 * time.Millisecond) //https://github.com/status-im/status-go/issues/429#issuecomment-339663163
	s.Equal(node.ErrNoRunningNode, s.NodeManager.StopNode())

	// start node
	s.False(s.NodeManager.IsNodeRunning())
	s.NoError(s.NodeManager.StartNode(nodeConfig))
	// wait till node is started
	s.True(s.NodeManager.IsNodeRunning())

	// try starting another node (w/o stopping the previously started node)
	s.Equal(node.ErrNodeExists, s.NodeManager.StartNode(nodeConfig))

	// now stop node
	time.Sleep(100 * time.Millisecond) //https://github.com/status-im/status-go/issues/429#issuecomment-339663163
	s.NoError(s.NodeManager.StopNode())
	s.False(s.NodeManager.IsNodeRunning())

	// start new node with exactly the same config
	s.NoError(s.NodeManager.StartNode(nodeConfig))
	s.True(s.NodeManager.IsNodeRunning())

	// finally stop the node
	time.Sleep(100 * time.Millisecond) //https://github.com/status-im/status-go/issues/429#issuecomment-339663163
	s.NoError(s.NodeManager.StopNode())
}

func (s *ManagerTestSuite) TestNetworkSwitching() {
	// get Ropsten config
	nodeConfig, err := e2e.MakeTestNodeConfig(GetNetworkID())
	s.NoError(err)
	s.False(s.NodeManager.IsNodeRunning())
	s.NoError(s.NodeManager.StartNode(nodeConfig))
	// wait till node is started
	s.Require().True(s.NodeManager.IsNodeRunning())

	firstHash, err := e2e.FirstBlockHash(s.NodeManager)
	s.NoError(err)
	s.Equal(GetHeadHash(), firstHash)

	// now stop node, and make sure that a new node, on different network can be started
	time.Sleep(100 * time.Millisecond) //https://github.com/status-im/status-go/issues/429#issuecomment-339663163
	s.NoError(s.NodeManager.StopNode())
	s.False(s.NodeManager.IsNodeRunning())

	// start new node with completely different config
	nodeConfig, err = e2e.MakeTestNodeConfig(params.RinkebyNetworkID)
	s.NoError(err)
	s.NoError(s.NodeManager.StartNode(nodeConfig))
	s.True(s.NodeManager.IsNodeRunning())

	// make sure we are on another network indeed
	firstHash, err = e2e.FirstBlockHash(s.NodeManager)
	s.NoError(err)
	s.Equal(GetHeadHashFromNetworkID(params.RinkebyNetworkID), firstHash)

	time.Sleep(100 * time.Millisecond) //https://github.com/status-im/status-go/issues/429#issuecomment-339663163
	s.NoError(s.NodeManager.StopNode())
}

func (s *ManagerTestSuite) TestStartNodeWithUpstreamEnabled() {
	if GetNetworkID() == params.StatusChainNetworkID {
		s.T().Skip()
	}

	nodeConfig, err := e2e.MakeTestNodeConfig(GetNetworkID())
	s.NoError(err)

	networkURL, err := GetRemoteURL()
	s.NoError(err)

	nodeConfig.UpstreamConfig.Enabled = true
	nodeConfig.UpstreamConfig.URL = networkURL

	s.NoError(s.NodeManager.StartNode(nodeConfig))
	s.True(s.NodeManager.IsNodeRunning())

	time.Sleep(100 * time.Millisecond) //https://github.com/status-im/status-go/issues/429#issuecomment-339663163
	s.NoError(s.NodeManager.StopNode())
}

// TODO(adam): race conditions should be tested with -race flag and unit tests, if possible.
// TODO(boris): going via https://github.com/status-im/status-go/pull/433#issuecomment-342232645 . Testing should be with -race flag
// Research if it's possible to do the same with unit tests.
//func (s *ManagerTestSuite) TestRaceConditions() {
//	cnt := 25
//	progress := make(chan struct{}, cnt)
//	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
//
//	nodeConfig1, e := e2e.MakeTestNodeConfig(GetNetworkID())
//	s.NoError(e)
//
//	nodeConfig2, e := e2e.MakeTestNodeConfig(GetNetworkID())
//	s.NoError(e)
//
//	nodeConfigs := []*params.NodeConfig{nodeConfig1, nodeConfig2}
//
//	var funcsToTest = []func(*params.NodeConfig){
//		func(config *params.NodeConfig) {
//			log.Info("StartNode()")
//			_, err := s.NodeManager.StartNode(config)
//			s.T().Logf("StartNode() for network: %d, error: %v", config.NetworkID, err)
//			progress <- struct{}{}
//		},
//		func(config *params.NodeConfig) {
//			_, err := s.NodeManager.StopNode()
//			s.T().Logf("StopNode() for network: %d, error: %v", config.NetworkID, err)
//			progress <- struct{}{}
//		},
//		func(config *params.NodeConfig) {
//			log.Info("Node()")
//			_, err := s.NodeManager.Node()
//			s.T().Logf("Node(), error: %v", err)
//			progress <- struct{}{}
//		},
//		func(config *params.NodeConfig) {
//			log.Info("IsNodeRunning()")
//			s.T().Logf("IsNodeRunning(), result: %v", s.NodeManager.IsNodeRunning())
//			progress <- struct{}{}
//		},
//		func(config *params.NodeConfig) {
//			log.Info("PopulateStaticPeers()")
//			s.T().Logf("PopulateStaticPeers(), error: %v", s.NodeManager.PopulateStaticPeers())
//			progress <- struct{}{}
//		},
//		// TODO(adam): quarantined until it uses a different datadir
//		// as otherwise it wipes out cached blockchain data.
//		// func(config *params.NodeConfig) {
//		//	log.Info("ResetChainData()")
//		//	_, err := s.NodeManager.ResetChainData()
//		//	s.T().Logf("ResetChainData(), error: %v", err)
//		//	progress <- struct{}{}
//		// },
//		func(config *params.NodeConfig) {
//			log.Info("RestartNode()")
//			_, err := s.NodeManager.RestartNode()
//			s.T().Logf("RestartNode(), error: %v", err)
//			progress <- struct{}{}
//		},
//		func(config *params.NodeConfig) {
//			log.Info("NodeConfig()")
//			_, err := s.NodeManager.NodeConfig()
//			s.T().Logf("NodeConfig(), error: %v", err)
//			progress <- struct{}{}
//		},
//		func(config *params.NodeConfig) {
//			log.Info("LightEthereumService()")
//			_, err := s.NodeManager.LightEthereumService()
//			s.T().Logf("LightEthereumService(), error: %v", err)
//			progress <- struct{}{}
//		},
//		func(config *params.NodeConfig) {
//			log.Info("WhisperService()")
//			_, err := s.NodeManager.WhisperService()
//			s.T().Logf("WhisperService(), error: %v", err)
//			progress <- struct{}{}
//		},
//		func(config *params.NodeConfig) {
//			log.Info("AccountManager()")
//			_, err := s.NodeManager.AccountManager()
//			s.T().Logf("AccountManager(), error: %v", err)
//			progress <- struct{}{}
//		},
//		func(config *params.NodeConfig) {
//			log.Info("AccountKeyStore()")
//			_, err := s.NodeManager.AccountKeyStore()
//			s.T().Logf("AccountKeyStore(), error: %v", err)
//			progress <- struct{}{}
//		},
//		func(config *params.NodeConfig) {
//			log.Info("RPCClient()")
//			s.NodeManager.RPCClient()
//			progress <- struct{}{}
//		},
//	}
//
//	// increase StartNode()/StopNode() population
//	for i := 0; i < 5; i++ {
//		funcsToTest = append(funcsToTest, funcsToTest[0], funcsToTest[1])
//	}
//
//	for i := 0; i < cnt; i++ {
//		randConfig := nodeConfigs[rnd.Intn(len(nodeConfigs))]
//		randFunc := funcsToTest[rnd.Intn(len(funcsToTest))]
//
//		if rnd.Intn(100) > 75 { // introduce random delays
//			time.Sleep(500 * time.Millisecond)
//		}
//		go randFunc(randConfig)
//	}
//
//	for range progress {
//		cnt -= 1
//		if cnt <= 0 {
//			break
//		}
//	}
//
//	time.Sleep(2 * time.Second)                // so that we see some logs
//	nodeStopped, _ := s.NodeManager.StopNode() // just in case we have a node running
//
//	if nodeStopped != nil {
//		<-nodeStopped
//	}
//}
