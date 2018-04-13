package api_test

import (
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/geth/account"
	"github.com/status-im/status-go/geth/jail"
	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/t/e2e"
	. "github.com/status-im/status-go/t/utils"
	"github.com/stretchr/testify/suite"
)

func TestAPIBackendTestSuite(t *testing.T) {
	suite.Run(t, new(APIBackendTestSuite))
}

type APIBackendTestSuite struct {
	e2e.BackendTestSuite
}

// FIXME(tiabc): There's also a test with the same name in geth/node/manager_test.go
// so this test should only check StatusBackend logic with a mocked version of the underlying StatusNode.
func (s *APIBackendTestSuite) TestRaceConditions() {
	require := s.Require()
	require.NotNil(s.Backend)

	cnt := 25
	progress := make(chan struct{}, cnt)
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	nodeConfig1, err := MakeTestNodeConfig(GetNetworkID())
	require.NoError(err)

	nodeConfig2, err := MakeTestNodeConfig(GetNetworkID())
	require.NoError(err)

	nodeConfigs := []*params.NodeConfig{nodeConfig1, nodeConfig2}

	var funcsToTest = []func(*params.NodeConfig){
		func(config *params.NodeConfig) {
			log.Info("StartNode()")
			err := s.Backend.StartNode(config)
			s.T().Logf("StartNode() for network: %d, error: %v", config.NetworkID, err)
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("StopNode()")
			err := s.Backend.StopNode()
			s.T().Logf("StopNode() for network: %d, error: %v", config.NetworkID, err)
			progress <- struct{}{}
		},
		// func(config *params.NodeConfig) {
		//	log.Info("ResetChainData()")
		//	_, err := s.Backend.ResetChainData()
		//	s.T().Logf("ResetChainData(), error: %v", err)
		//	progress <- struct{}{}
		// },
		func(config *params.NodeConfig) {
			log.Info("RestartNode()")
			err := s.Backend.RestartNode()
			s.T().Logf("RestartNode(), error: %v", err)
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("StatusNode()")
			instance := s.Backend.StatusNode()
			s.NotNil(instance)
			s.IsType(&node.StatusNode{}, instance)
			s.T().Logf("StatusNode(), result: %v", instance)
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
			if err != nil {
				progress <- struct{}{}
				return
			}

			// SelectAccount
			log.Info("CreateAccount()")
			err = s.Backend.SelectAccount(address, "password")
			s.T().Logf("SelectAccount(%v, %v), error: %v", address, "password", err)

			// CreateChildAccount
			log.Info("CreateChildAccount()")
			address, pubKey, err = s.Backend.AccountManager().CreateChildAccount(address, "password")
			s.T().Logf("CreateAccount(), error: %v (address: %v, pubKey: %v)", err, address, pubKey)

			// RecoverAccount
			log.Info("RecoverAccount()")
			address, pubKey, err = s.Backend.AccountManager().RecoverAccount("password", mnemonic)
			s.T().Logf("RecoverAccount(), error: %v (address: %v, pubKey: %v)", err, address, pubKey)

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
			ids := []string{"id1", "id2"}
			s.T().Logf("CompleteTransactions(), result: %v", s.Backend.CompleteTransactions(ids, "password"))
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("DiscardTransactions()")
			ids := []string{"id1", "id2"}
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
		cnt--
		if cnt <= 0 {
			break
		}
	}

	time.Sleep(2 * time.Second) // so that we see some logs
	if err := s.Backend.StopNode(); err != node.ErrNoRunningNode && err != nil {
		s.NoError(err, "unexpected error")
	}
}

// FIXME(tiabc): There's also a test with the same name in geth/node/manager_test.go
// so this test should only check StatusBackend logic with a mocked version of the underlying StatusNode.
func (s *APIBackendTestSuite) TestNetworkSwitching() {
	// Get test node configuration.
	nodeConfig, err := MakeTestNodeConfig(GetNetworkID())
	s.NoError(err)

	s.False(s.Backend.IsNodeRunning())
	s.NoError(s.Backend.StartNode(nodeConfig))
	s.True(s.Backend.IsNodeRunning())

	firstHash, err := e2e.FirstBlockHash(s.Backend.StatusNode())
	s.NoError(err)
	s.Equal(GetHeadHash(), firstHash)

	// now stop node, and make sure that a new node, on different network can be started
	s.NoError(s.Backend.StopNode())

	// start new node with completely different config
	nodeConfig, err = MakeTestNodeConfig(GetNetworkID())
	s.NoError(err)

	s.False(s.Backend.IsNodeRunning())
	s.NoError(s.Backend.StartNode(nodeConfig))
	s.True(s.Backend.IsNodeRunning())

	// make sure we are on another network indeed
	firstHash, err = e2e.FirstBlockHash(s.Backend.StatusNode())
	s.NoError(err)
	s.Equal(GetHeadHash(), firstHash)

	s.NoError(s.Backend.StopNode())
}

func (s *APIBackendTestSuite) TestResetChainData() {
	if GetNetworkID() != params.StatusChainNetworkID {
		s.T().Skip("test must be running on status network")
	}
	require := s.Require()
	require.NotNil(s.Backend)
	path, err := ioutil.TempDir("/tmp", "status-reset-chain-test")
	require.NoError(err)
	defer func() { s.NoError(os.RemoveAll(path)) }()

	s.StartTestBackend(e2e.WithDataDir(path))
	defer s.StopTestBackend()

	EnsureNodeSync(s.Backend.StatusNode().EnsureSync)

	require.NoError(s.Backend.ResetChainData())

	s.True(s.Backend.IsNodeRunning()) // new node, with previous config should be running

	// make sure we can read the first byte, and it is valid (for Rinkeby)
	firstHash, err := e2e.FirstBlockHash(s.Backend.StatusNode())
	s.NoError(err)
	s.Equal(GetHeadHash(), firstHash)
}

// FIXME(tiabc): There's also a test with the same name in geth/node/manager_test.go
// so this test should only check StatusBackend logic with a mocked version of the underlying StatusNode.
func (s *APIBackendTestSuite) TestRestartNode() {
	require := s.Require()
	require.NotNil(s.Backend)

	// get config
	nodeConfig, err := MakeTestNodeConfig(GetNetworkID())
	s.NoError(err)

	s.False(s.Backend.IsNodeRunning())
	s.NoError(s.Backend.StartNode(nodeConfig))
	s.True(s.Backend.IsNodeRunning())

	firstHash, err := e2e.FirstBlockHash(s.Backend.StatusNode())
	s.NoError(err)
	s.Equal(GetHeadHash(), firstHash)

	s.True(s.Backend.IsNodeRunning())
	require.NoError(s.Backend.RestartNode())
	s.True(s.Backend.IsNodeRunning()) // new node, with previous config should be running

	// make sure we can read the first byte, and it is valid (for Rinkeby)
	firstHash, err = e2e.FirstBlockHash(s.Backend.StatusNode())
	s.NoError(err)
	s.Equal(GetHeadHash(), firstHash)
}
