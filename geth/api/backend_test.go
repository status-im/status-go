package api_test

import (
	"encoding/json"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/les"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/geth/account"
	"github.com/status-im/status-go/geth/api"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/jail"
	"github.com/status-im/status-go/geth/log"
	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/signal"
	. "github.com/status-im/status-go/geth/testing"
	"github.com/status-im/status-go/geth/txqueue"
	"github.com/stretchr/testify/suite"
)

func TestBackendTestSuite(t *testing.T) {
	suite.Run(t, new(BackendTestSuite))
}

type BackendTestSuite struct {
	suite.Suite
	backend *api.StatusBackend
}

func (s *BackendTestSuite) SetupTest() {
	require := s.Require()
	backend := api.NewStatusBackend()
	require.NotNil(backend)
	require.IsType(&api.StatusBackend{}, backend)
	s.backend = backend
}

func (s *BackendTestSuite) StartTestBackend(networkID int, opts ...TestNodeOption) {
	require := s.Require()
	require.NotNil(s.backend)

	nodeConfig, err := MakeTestNodeConfig(networkID)
	require.NoError(err)

	// Apply any options altering node config.
	for i := range opts {
		opts[i](nodeConfig)
	}

	// import account keys
	require.NoError(common.ImportTestAccount(nodeConfig.KeyStoreDir, "test-account1.pk"))
	require.NoError(common.ImportTestAccount(nodeConfig.KeyStoreDir, "test-account2.pk"))

	require.False(s.backend.IsNodeRunning())
	nodeStarted, err := s.backend.StartNode(nodeConfig)
	require.NoError(err)
	<-nodeStarted
	require.True(s.backend.IsNodeRunning())
}

func (s *BackendTestSuite) StopTestBackend() {
	require := s.Require()
	require.NotNil(s.backend)
	require.True(s.backend.IsNodeRunning())
	backendStopped, err := s.backend.StopNode()
	require.NoError(err)
	<-backendStopped
	require.False(s.backend.IsNodeRunning())
}

func (s *BackendTestSuite) WhisperService() *whisper.Whisper {
	require := s.Require()
	require.NotNil(s.backend)

	whisperService, err := s.backend.NodeManager().WhisperService()
	require.NoError(err)
	require.NotNil(whisperService)

	return whisperService
}

func (s *BackendTestSuite) LightEthereumService() *les.LightEthereum {
	require := s.Require()
	require.NotNil(s.backend)

	lightEthereum, err := s.backend.NodeManager().LightEthereumService()
	require.NoError(err)
	require.NotNil(lightEthereum)

	return lightEthereum
}

func (s *BackendTestSuite) TxQueueManager() common.TxQueueManager {
	return s.backend.TxQueueManager()
}

func (s *BackendTestSuite) RestartTestNode() {
	require := s.Require()
	require.NotNil(s.backend)
	require.True(s.backend.IsNodeRunning())

	require.True(s.backend.IsNodeRunning())
	nodeRestarted, err := s.backend.RestartNode()
	require.NoError(err)
	require.NotNil(nodeRestarted)
	<-nodeRestarted
	require.True(s.backend.IsNodeRunning())
}

func (s *BackendTestSuite) TestNewBackend() {
	require := s.Require()
	require.NotNil(s.backend)

	s.StartTestBackend(params.RinkebyNetworkID)
	defer s.StopTestBackend()
}

func (s *BackendTestSuite) TestNodeStartStop() {
	require := s.Require()
	require.NotNil(s.backend)

	nodeConfig, err := MakeTestNodeConfig(params.RopstenNetworkID)
	require.NoError(err)

	// try stopping non-started node
	require.False(s.backend.IsNodeRunning())
	nodeStopped, err := s.backend.StopNode()
	require.EqualError(err, node.ErrNoRunningNode.Error())
	require.Nil(nodeStopped)

	require.False(s.backend.IsNodeRunning())
	nodeStarted, err := s.backend.StartNode(nodeConfig)
	require.NoError(err)
	require.NotNil(nodeStarted)

	<-nodeStarted // wait till node is started
	require.True(s.backend.IsNodeRunning())

	// try starting another node (w/o stopping the previously started node)
	nodeStarted, err = s.backend.StartNode(nodeConfig)
	require.EqualError(err, node.ErrNodeExists.Error())
	require.Nil(nodeStarted)

	// now stop node, and make sure that a new node, on different network can be started
	nodeStopped, err = s.backend.StopNode()
	require.NoError(err)
	require.NotNil(nodeStopped)
	<-nodeStopped

	// start new node with exactly the same config
	require.False(s.backend.IsNodeRunning())
	nodeStarted, err = s.backend.StartNode(nodeConfig)
	require.NoError(err)
	require.NotNil(nodeStarted)
	defer s.backend.StopNode() //nolint: errcheck

	<-nodeStarted
	require.True(s.backend.IsNodeRunning())
}

func (s *BackendTestSuite) TestStartNodeWithUpstreamEnabled() {
	require := s.Require()

	backend := api.NewStatusBackend()

	nodeConfig, err := MakeTestNodeConfig(params.RopstenNetworkID)
	require.NoError(err)

	nodeConfig.UpstreamConfig.Enabled = true

	nodeStarted, err := backend.StartNode(nodeConfig)
	require.NoError(err)
	defer backend.StopNode() //nolint: errcheck

	<-nodeStarted
	require.True(backend.IsNodeRunning())
}

// FIXME(tiabc): There's also a test with the same name in geth/node/rpc_test.go
// so this test should only check StatusBackend logic with a mocked version of the underlying NodeManager.
func (s *BackendTestSuite) TestCallRPC() {
	require := s.Require()
	require.NotNil(s.backend)

	nodeConfig, err := MakeTestNodeConfig(params.RinkebyNetworkID)
	require.NoError(err)

	nodeStarted, err := s.backend.StartNode(nodeConfig)
	require.NoError(err)
	require.NotNil(nodeStarted)
	defer s.backend.StopNode() //nolint: errcheck
	<-nodeStarted

	progress := make(chan struct{}, 25)
	type rpcCall struct {
		inputJSON string
		validator func(resultJSON string)
	}
	var rpcCalls = []rpcCall{
		{
			`{"jsonrpc":"2.0","method":"eth_sendTransaction","params":[{
				"from": "0xb60e8dd61c5d32be8058bb8eb970870f07233155",
				"to": "0xd46e8dd67c5d32be8058bb8eb970870f07244567",
				"gas": "0x76c0",
				"gasPrice": "0x9184e72a000",
				"value": "0x9184e72a",
				"data": "0xd46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675"}],"id":1}`,
			func(resultJSON string) {
				log.Info("eth_sendTransaction")
				s.NotContains(resultJSON, "error")
				progress <- struct{}{}
			},
		},
		{
			`{"jsonrpc":"2.0","method":"shh_version","params":[],"id":67}`,
			func(resultJSON string) {
				expected := `{"jsonrpc":"2.0","id":67,"result":"5.0"}`
				s.Equal(expected, resultJSON)
				s.T().Log("shh_version: ", resultJSON)
				progress <- struct{}{}
			},
		},
		{
			`{"jsonrpc":"2.0","method":"web3_sha3","params":["0x68656c6c6f20776f726c64"],"id":64}`,
			func(resultJSON string) {
				expected := `{"jsonrpc":"2.0","id":64,"result":"0x47173285a8d7341e5e972fc677286384f802f8ef42a5ec5f03bbfa254cb01fad"}`
				s.Equal(expected, resultJSON)
				s.T().Log("web3_sha3: ", resultJSON)
				progress <- struct{}{}
			},
		},
		{
			`{"jsonrpc":"2.0","method":"net_version","params":[],"id":67}`,
			func(resultJSON string) {
				expected := `{"jsonrpc":"2.0","id":67,"result":"4"}`
				s.Equal(expected, resultJSON)
				s.T().Log("net_version: ", resultJSON)
				progress <- struct{}{}
			},
		},
	}

	cnt := len(rpcCalls) - 1 // send transaction blocks up until complete/discarded/times out
	for _, r := range rpcCalls {
		go func(r rpcCall) {
			s.T().Logf("Run test: %v", r.inputJSON)
			resultJSON := s.backend.CallRPC(r.inputJSON)
			r.validator(resultJSON)
		}(r)
	}

	for range progress {
		cnt -= 1
		if cnt <= 0 {
			break
		}
	}
}

func (s *BackendTestSuite) TestCallRPCSendTransaction() {
	nodeConfig, err := MakeTestNodeConfig(params.RopstenNetworkID)
	s.NoError(err)

	nodeStarted, err := s.backend.StartNode(nodeConfig)
	s.NoError(err)
	defer s.backend.StopNode() //nolint: errcheck

	<-nodeStarted

	// Allow to sync the blockchain.
	time.Sleep(TestConfig.Node.SyncSeconds * time.Second)

	err = s.backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password)
	s.NoError(err)

	transactionCompleted := make(chan struct{})

	var txHash gethcommon.Hash
	signal.SetDefaultNodeNotificationHandler(func(rawSignal string) {
		var signal signal.Envelope
		err := json.Unmarshal([]byte(rawSignal), &signal)
		s.NoError(err)

		if signal.Type == txqueue.EventTransactionQueued {
			event := signal.Event.(map[string]interface{})
			txID := event["id"].(string)
			txHash, err = s.backend.CompleteTransaction(common.QueuedTxID(txID), TestConfig.Account1.Password)
			s.NoError(err, "cannot complete queued transaction %s", txID)

			close(transactionCompleted)
		}
	})

	result := s.backend.CallRPC(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "eth_sendTransaction",
		"params": [{
			"from": "` + TestConfig.Account1.Address + `",
			"to": "0xd46e8dd67c5d32be8058bb8eb970870f07244567",
			"value": "0x9184e72a"
		}]
	}`)
	s.NotContains(result, "error")

	select {
	case <-transactionCompleted:
	case <-time.After(time.Minute):
		s.FailNow("sending transaction timed out")
	}

	s.Equal(`{"jsonrpc":"2.0","id":1,"result":"`+txHash.String()+`"}`, result)
}

func (s *BackendTestSuite) TestCallRPCSendTransactionUpstream() {
	nodeConfig, err := MakeTestNodeConfig(params.RopstenNetworkID)
	s.NoError(err)

	nodeConfig.UpstreamConfig.Enabled = true
	nodeConfig.UpstreamConfig.URL = "https://ropsten.infura.io/nKmXgiFgc2KqtoQ8BCGJ"

	nodeStarted, err := s.backend.StartNode(nodeConfig)
	s.NoError(err)
	defer s.backend.StopNode() //nolint: errcheck

	<-nodeStarted

	// Allow to sync the blockchain.
	time.Sleep(TestConfig.Node.SyncSeconds * time.Second)

	err = s.backend.AccountManager().SelectAccount(TestConfig.Account2.Address, TestConfig.Account2.Password)
	s.NoError(err)

	transactionCompleted := make(chan struct{})

	var txHash gethcommon.Hash
	signal.SetDefaultNodeNotificationHandler(func(rawSignal string) {
		var signal signal.Envelope
		err := json.Unmarshal([]byte(rawSignal), &signal)
		s.NoError(err)

		if signal.Type == txqueue.EventTransactionQueued {
			event := signal.Event.(map[string]interface{})
			txID := event["id"].(string)

			// Complete with a wrong passphrase.
			txHash, err = s.backend.CompleteTransaction(common.QueuedTxID(txID), "some-invalid-passphrase")
			s.EqualError(err, keystore.ErrDecrypt.Error(), "should return an error as the passphrase was invalid")

			// Complete with a correct passphrase.
			txHash, err = s.backend.CompleteTransaction(common.QueuedTxID(txID), TestConfig.Account2.Password)
			s.NoError(err, "cannot complete queued transaction %s", txID)

			close(transactionCompleted)
		}
	})

	result := s.backend.CallRPC(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "eth_sendTransaction",
		"params": [{
			"from": "` + TestConfig.Account2.Address + `",
			"to": "` + TestConfig.Account1.Address + `",
			"value": "0x9184e72a"
		}]
	}`)
	s.NotContains(result, "error")

	select {
	case <-transactionCompleted:
	case <-time.After(time.Minute):
		s.FailNow("sending transaction timed out")
	}

	s.Equal(`{"jsonrpc":"2.0","id":1,"result":"`+txHash.String()+`"}`, result)
}

// FIXME(tiabc): There's also a test with the same name in geth/node/manager_test.go
// so this test should only check StatusBackend logic with a mocked version of the underlying NodeManager.
func (s *BackendTestSuite) TestRaceConditions() {
	require := s.Require()
	require.NotNil(s.backend)

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
			_, err := s.backend.StartNode(config)
			s.T().Logf("StartNode() for network: %d, error: %v", config.NetworkID, err)
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("StopNode()")
			_, err := s.backend.StopNode()
			s.T().Logf("StopNode() for network: %d, error: %v", config.NetworkID, err)
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("ResetChainData()")
			_, err := s.backend.ResetChainData()
			s.T().Logf("ResetChainData(), error: %v", err)
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("RestartNode()")
			_, err := s.backend.RestartNode()
			s.T().Logf("RestartNode(), error: %v", err)
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("NodeManager()")
			instance := s.backend.NodeManager()
			s.NotNil(instance)
			s.IsType(&node.NodeManager{}, instance)
			s.T().Logf("NodeManager(), result: %v", instance)
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("AccountManager()")
			instance := s.backend.AccountManager()
			s.NotNil(instance)
			s.IsType(&account.Manager{}, instance)
			s.T().Logf("Manager(), result: %v", instance)
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("JailManager()")
			instance := s.backend.JailManager()
			s.NotNil(instance)
			s.IsType(&jail.Jail{}, instance)
			s.T().Logf("JailManager(), result: %v", instance)
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("CreateAccount()")
			address, pubKey, mnemonic, err := s.backend.AccountManager().CreateAccount("password")
			s.T().Logf("CreateAccount(), error: %v (address: %v, pubKey: %v, mnemonic: %v)", err, address, pubKey, mnemonic)
			if err == nil {
				// SelectAccount
				log.Info("CreateAccount()")
				err = s.backend.AccountManager().SelectAccount(address, "password")
				s.T().Logf("SelectAccount(%v, %v), error: %v", address, "password", err)

				// CreateChildAccount
				log.Info("CreateChildAccount()")
				address, pubKey, err := s.backend.AccountManager().CreateChildAccount(address, "password")
				s.T().Logf("CreateAccount(), error: %v (address: %v, pubKey: %v)", err, address, pubKey)

				// RecoverAccount
				log.Info("RecoverAccount()")
				address, pubKey, err = s.backend.AccountManager().RecoverAccount("password", mnemonic)
				s.T().Logf("RecoverAccount(), error: %v (address: %v, pubKey: %v)", err, address, pubKey)
			}
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("VerifyAccountPassword()")
			_, err := s.backend.AccountManager().VerifyAccountPassword(config.KeyStoreDir, "0x0", "bar")
			s.T().Logf("VerifyAccountPassword(), err: %v", err)
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("Logout()")
			s.T().Logf("Logout(), result: %v", s.backend.AccountManager().Logout())
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("IsNodeRunning()")
			s.T().Logf("IsNodeRunning(), result: %v", s.backend.IsNodeRunning())
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("CompleteTransaction()")
			_, err := s.backend.CompleteTransaction("id", "password")
			s.T().Logf("CompleteTransaction(), error: %v", err)
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("DiscardTransaction()")
			s.T().Logf("DiscardTransaction(), error: %v", s.backend.DiscardTransaction("id"))
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("CompleteTransactions()")
			ids := []common.QueuedTxID{"id1", "id2"}
			s.T().Logf("CompleteTransactions(), result: %v", s.backend.CompleteTransactions(ids, "password"))
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("DiscardTransactions()")
			ids := []common.QueuedTxID{"id1", "id2"}
			s.T().Logf("DiscardTransactions(), result: %v", s.backend.DiscardTransactions(ids))
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
	nodeStopped, _ := s.backend.StopNode() // just in case we have a node running
	if nodeStopped != nil {
		<-nodeStopped
	}
}

// FIXME(tiabc): There's also a test with the same name in geth/node/manager_test.go
// so this test should only check StatusBackend logic with a mocked version of the underlying NodeManager.
func (s *BackendTestSuite) TestNetworkSwitching() {
	require := s.Require()
	require.NotNil(s.backend)

	// get Ropsten config
	nodeConfig, err := MakeTestNodeConfig(params.RopstenNetworkID)
	require.NoError(err)

	require.False(s.backend.IsNodeRunning())
	nodeStarted, err := s.backend.StartNode(nodeConfig)
	require.NoError(err)

	<-nodeStarted // wait till node is started
	require.True(s.backend.IsNodeRunning())

	FirstBlockHash(require, s.backend.NodeManager(), "0x41941023680923e0fe4d74a34bdac8141f2540e3ae90623718e47d66d1ca4a2d")

	// now stop node, and make sure that a new node, on different network can be started
	nodeStopped, err := s.backend.StopNode()
	require.NoError(err)
	<-nodeStopped

	// start new node with completely different config
	nodeConfig, err = MakeTestNodeConfig(params.RinkebyNetworkID)
	require.NoError(err)

	require.False(s.backend.IsNodeRunning())
	nodeStarted, err = s.backend.StartNode(nodeConfig)
	require.NoError(err)

	<-nodeStarted
	require.True(s.backend.IsNodeRunning())

	// make sure we are on another network indeed
	FirstBlockHash(require, s.backend.NodeManager(), "0x6341fd3daf94b748c72ced5a5b26028f2474f5f00d824504e4fa37a75767e177")

	nodeStopped, err = s.backend.StopNode()
	require.NoError(err)
	<-nodeStopped
}

// FIXME(tiabc): There's also a test with the same name in geth/node/manager_test.go
// so this test should only check StatusBackend logic with a mocked version of the underlying NodeManager.
func (s *BackendTestSuite) TestResetChainData() {
	require := s.Require()
	require.NotNil(s.backend)

	s.StartTestBackend(params.RinkebyNetworkID)
	defer s.StopTestBackend()

	time.Sleep(2 * time.Second) // allow to sync for some time

	s.True(s.backend.IsNodeRunning())
	nodeReady, err := s.backend.ResetChainData()
	require.NoError(err)
	<-nodeReady
	s.True(s.backend.IsNodeRunning()) // new node, with previous config should be running

	// make sure we can read the first byte, and it is valid (for Rinkeby)
	FirstBlockHash(require, s.backend.NodeManager(), "0x6341fd3daf94b748c72ced5a5b26028f2474f5f00d824504e4fa37a75767e177")
}

// FIXME(tiabc): There's also a test with the same name in geth/node/manager_test.go
// so this test should only check StatusBackend logic with a mocked version of the underlying NodeManager.
func (s *BackendTestSuite) TestRestartNode() {
	require := s.Require()
	require.NotNil(s.backend)

	s.StartTestBackend(params.RinkebyNetworkID)
	defer s.StopTestBackend()

	FirstBlockHash(require, s.backend.NodeManager(), "0x6341fd3daf94b748c72ced5a5b26028f2474f5f00d824504e4fa37a75767e177")

	s.True(s.backend.IsNodeRunning())
	nodeRestarted, err := s.backend.RestartNode()
	require.NoError(err)
	<-nodeRestarted
	s.True(s.backend.IsNodeRunning()) // new node, with previous config should be running

	// make sure we can read the first byte, and it is valid (for Rinkeby)
	FirstBlockHash(require, s.backend.NodeManager(), "0x6341fd3daf94b748c72ced5a5b26028f2474f5f00d824504e4fa37a75767e177")
}
