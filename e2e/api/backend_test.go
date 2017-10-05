package api_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/status-im/status-go/e2e"
	"github.com/status-im/status-go/geth/account"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/jail"
	"github.com/status-im/status-go/geth/log"
	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
	"github.com/stretchr/testify/suite"
)

func TestAPIBackendTestSuite(t *testing.T) {
	suite.Run(t, new(APIBackendTestSuite))
}

type APIBackendTestSuite struct {
	e2e.BackendTestSuite
}

// FIXME(tiabc): There's also a test with the same name in geth/node/manager_test.go
// so this test should only check StatusBackend logic with a mocked version of the underlying NodeManager.
func (s *APIBackendTestSuite) TestRaceConditions() {
	require := s.Require()
	require.NotNil(s.Backend)

	cnt := 25
	progress := make(chan struct{}, cnt)
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	nodeConfig1, err := e2e.MakeTestNodeConfig(params.RopstenNetworkID)
	require.NoError(err)

	nodeConfig2, err := e2e.MakeTestNodeConfig(params.RinkebyNetworkID)
	require.NoError(err)

	nodeConfigs := []*params.NodeConfig{nodeConfig1, nodeConfig2}

	var funcsToTest = []func(*params.NodeConfig){
		func(config *params.NodeConfig) {
			log.Info("StartNode()")
			_, err := s.Backend.StartNode(config)
			s.T().Logf("StartNode() for network: %d, error: %v", config.NetworkID, err)
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("StopNode()")
			_, err := s.Backend.StopNode()
			s.T().Logf("StopNode() for network: %d, error: %v", config.NetworkID, err)
			progress <- struct{}{}
		},
		// func(config *params.NodeConfig) {
		// 	log.Info("ResetChainData()")
		// 	_, err := s.Backend.ResetChainData()
		// 	s.T().Logf("ResetChainData(), error: %v", err)
		// 	progress <- struct{}{}
		// },
		func(config *params.NodeConfig) {
			log.Info("RestartNode()")
			_, err := s.Backend.RestartNode()
			s.T().Logf("RestartNode(), error: %v", err)
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("NodeManager()")
			instance := s.Backend.NodeManager()
			s.NotNil(instance)
			s.IsType(&node.NodeManager{}, instance)
			s.T().Logf("NodeManager(), result: %v", instance)
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("AccountManager()")
			instance := s.Backend.AccountManager()
			s.NotNil(instance)
			s.IsType(&account.Manager{}, instance)
			s.T().Logf("Manager(), result: %v", instance)
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("JailManager()")
			instance := s.Backend.JailManager()
			s.NotNil(instance)
			s.IsType(&jail.Jail{}, instance)
			s.T().Logf("JailManager(), result: %v", instance)
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("CreateAccount()")
			address, pubKey, mnemonic, err := s.Backend.AccountManager().CreateAccount("password")
			s.T().Logf("CreateAccount(), error: %v (address: %v, pubKey: %v, mnemonic: %v)", err, address, pubKey, mnemonic)
			if err == nil {
				// SelectAccount
				log.Info("CreateAccount()")
				err = s.Backend.AccountManager().SelectAccount(address, "password")
				s.T().Logf("SelectAccount(%v, %v), error: %v", address, "password", err)

				// CreateChildAccount
				log.Info("CreateChildAccount()")
				address, pubKey, err := s.Backend.AccountManager().CreateChildAccount(address, "password")
				s.T().Logf("CreateAccount(), error: %v (address: %v, pubKey: %v)", err, address, pubKey)

				// RecoverAccount
				log.Info("RecoverAccount()")
				address, pubKey, err = s.Backend.AccountManager().RecoverAccount("password", mnemonic)
				s.T().Logf("RecoverAccount(), error: %v (address: %v, pubKey: %v)", err, address, pubKey)
			}
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("VerifyAccountPassword()")
			_, err := s.Backend.AccountManager().VerifyAccountPassword(config.KeyStoreDir, "0x0", "bar")
			s.T().Logf("VerifyAccountPassword(), err: %v", err)
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("Logout()")
			s.T().Logf("Logout(), result: %v", s.Backend.AccountManager().Logout())
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("IsNodeRunning()")
			s.T().Logf("IsNodeRunning(), result: %v", s.Backend.IsNodeRunning())
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("CompleteTransaction()")
			_, err := s.Backend.CompleteTransaction("id", "password")
			s.T().Logf("CompleteTransaction(), error: %v", err)
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("DiscardTransaction()")
			s.T().Logf("DiscardTransaction(), error: %v", s.Backend.DiscardTransaction("id"))
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("CompleteTransactions()")
			ids := []common.QueuedTxID{"id1", "id2"}
			s.T().Logf("CompleteTransactions(), result: %v", s.Backend.CompleteTransactions(ids, "password"))
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("DiscardTransactions()")
			ids := []common.QueuedTxID{"id1", "id2"}
			s.T().Logf("DiscardTransactions(), result: %v", s.Backend.DiscardTransactions(ids))
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

	time.Sleep(2 * time.Second)            // so that we see some logs
	nodeStopped, _ := s.Backend.StopNode() // just in case we have a node running
	if nodeStopped != nil {
		<-nodeStopped
	}
}

// FIXME(tiabc): There's also a test with the same name in geth/node/manager_test.go
// so this test should only check StatusBackend logic with a mocked version of the underlying NodeManager.
func (s *APIBackendTestSuite) TestNetworkSwitching() {
	// get Ropsten config
	nodeConfig, err := e2e.MakeTestNodeConfig(params.RopstenNetworkID)
	s.NoError(err)

	s.False(s.Backend.IsNodeRunning())
	nodeStarted, err := s.Backend.StartNode(nodeConfig)
	s.NoError(err)

	<-nodeStarted // wait till node is started
	s.True(s.Backend.IsNodeRunning())

	firstHash, err := e2e.FirstBlockHash(s.Backend.NodeManager())
	s.NoError(err)
	s.Equal("0x41941023680923e0fe4d74a34bdac8141f2540e3ae90623718e47d66d1ca4a2d", firstHash)

	// now stop node, and make sure that a new node, on different network can be started
	nodeStopped, err := s.Backend.StopNode()
	s.NoError(err)
	<-nodeStopped

	// start new node with completely different config
	nodeConfig, err = e2e.MakeTestNodeConfig(params.RinkebyNetworkID)
	s.NoError(err)

	s.False(s.Backend.IsNodeRunning())
	nodeStarted, err = s.Backend.StartNode(nodeConfig)
	s.NoError(err)

	<-nodeStarted
	s.True(s.Backend.IsNodeRunning())

	// make sure we are on another network indeed
	firstHash, err = e2e.FirstBlockHash(s.Backend.NodeManager())
	s.NoError(err)
	s.Equal("0x6341fd3daf94b748c72ced5a5b26028f2474f5f00d824504e4fa37a75767e177", firstHash)

	nodeStopped, err = s.Backend.StopNode()
	s.NoError(err)
	<-nodeStopped
}

// FIXME(tiabc): There's also a test with the same name in geth/node/manager_test.go
// so this test should only check StatusBackend logic with a mocked version of the underlying NodeManager.
func (s *APIBackendTestSuite) TestResetChainData() {
	s.T().Skip()

	require := s.Require()
	require.NotNil(s.Backend)

	s.StartTestBackend(params.RinkebyNetworkID)
	defer s.StopTestBackend()

	time.Sleep(2 * time.Second) // allow to sync for some time

	s.True(s.Backend.IsNodeRunning())
	nodeReady, err := s.Backend.ResetChainData()
	require.NoError(err)
	<-nodeReady
	s.True(s.Backend.IsNodeRunning()) // new node, with previous config should be running

	// make sure we can read the first byte, and it is valid (for Rinkeby)
	firstHash, err := e2e.FirstBlockHash(s.Backend.NodeManager())
	s.NoError(err)
	s.Equal("0x6341fd3daf94b748c72ced5a5b26028f2474f5f00d824504e4fa37a75767e177", firstHash)
}

// FIXME(tiabc): There's also a test with the same name in geth/node/manager_test.go
// so this test should only check StatusBackend logic with a mocked version of the underlying NodeManager.
func (s *APIBackendTestSuite) TestRestartNode() {
	require := s.Require()
	require.NotNil(s.Backend)

	s.StartTestBackend(params.RinkebyNetworkID)
	defer s.StopTestBackend()

	firstHash, err := e2e.FirstBlockHash(s.Backend.NodeManager())
	s.NoError(err)
	s.Equal("0x6341fd3daf94b748c72ced5a5b26028f2474f5f00d824504e4fa37a75767e177", firstHash)

	s.True(s.Backend.IsNodeRunning())
	nodeRestarted, err := s.Backend.RestartNode()
	require.NoError(err)
	<-nodeRestarted
	s.True(s.Backend.IsNodeRunning()) // new node, with previous config should be running

	// make sure we can read the first byte, and it is valid (for Rinkeby)
	firstHash, err = e2e.FirstBlockHash(s.Backend.NodeManager())
	s.NoError(err)
	s.Equal("0x6341fd3daf94b748c72ced5a5b26028f2474f5f00d824504e4fa37a75767e177", firstHash)
}
