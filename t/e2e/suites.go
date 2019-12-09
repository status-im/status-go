package e2e

import (
	"github.com/ethereum/go-ethereum/log"

	"github.com/status-im/status-go/whisper"

	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/t/utils"
	"github.com/status-im/status-go/transactions"
	"github.com/stretchr/testify/suite"
)

// StatusNodeTestSuite defines a test suite with StatusNode.
type StatusNodeTestSuite struct {
	suite.Suite
	StatusNode *node.StatusNode
}

// All general log messages in this package should be routed through this logger.
var logger = log.New("package", "status-go/t/e2e")

func Init() {
	utils.Init()
	for id := range utils.TestNetworkNames {
		nodeConfig, err := utils.MakeTestNodeConfig(id)
		if err != nil {
			panic(err)
		}

		err = importTestAccounts(nodeConfig.KeyStoreDir)
		if err != nil {
			panic(err)
		}
	}
}

// StartTestNode initiazes a StatusNode instances with configuration retrieved
// from the test config.
func (s *StatusNodeTestSuite) StartTestNode(opts ...TestNodeOption) {
	nodeConfig, err := utils.MakeTestNodeConfig(utils.GetNetworkID())
	s.NoError(err)

	// Apply any options altering node config.
	for i := range opts {
		opts[i](nodeConfig)
	}

	// import account keys
	s.NoError(importTestAccounts(nodeConfig.KeyStoreDir))

	s.False(s.StatusNode.IsRunning())
	s.NoError(s.StatusNode.Start(nodeConfig, nil))
	s.True(s.StatusNode.IsRunning())
}

// StopTestNode attempts to stop initialized StatusNode.
func (s *StatusNodeTestSuite) StopTestNode() {
	s.NotNil(s.StatusNode)
	s.True(s.StatusNode.IsRunning())
	s.NoError(s.StatusNode.Stop())
	s.False(s.StatusNode.IsRunning())
}

// BackendTestSuite is a test suite with api.GethStatusBackend initialized
// and a few utility methods to start and stop node or get various services.
type BackendTestSuite struct {
	suite.Suite
	Backend *api.GethStatusBackend
}

// SetupTest initializes Backend.
func (s *BackendTestSuite) SetupTest() {
	s.Backend = api.NewGethStatusBackend()
	s.NotNil(s.Backend)
}

// TearDownTest cleans up the packages state.
func (s *BackendTestSuite) TearDownTest() {
	signal.ResetDefaultNodeNotificationHandler()
}

// StartTestBackend imports some keys and starts a node.
func (s *BackendTestSuite) StartTestBackend(opts ...TestNodeOption) {
	nodeConfig, err := utils.MakeTestNodeConfig(utils.GetNetworkID())
	s.Require().NoError(err)

	// Apply any options altering node config.
	for i := range opts {
		opts[i](nodeConfig)
	}
	s.NoError(s.Backend.AccountManager().InitKeystore(nodeConfig.KeyStoreDir))
	// import account keys
	s.NoError(importTestAccounts(nodeConfig.KeyStoreDir))

	// start node
	s.False(s.Backend.IsNodeRunning())
	s.Require().NoError(s.Backend.StartNode(nodeConfig))
	s.True(s.Backend.IsNodeRunning())
}

func (s *BackendTestSuite) StartTestBackendWithAccount(account multiaccounts.Account, password string, subaccs []accounts.Account, opts ...TestNodeOption) {
	nodeConfig, err := utils.MakeTestNodeConfig(utils.GetNetworkID())
	s.Require().NoError(err)

	// Apply any options altering node config.
	for i := range opts {
		opts[i](nodeConfig)
	}
	// accounts must be imported before keystore is initialized
	s.NoError(importTestAccounts(nodeConfig.KeyStoreDir))
	s.Backend.UpdateRootDataDir(nodeConfig.DataDir)
	s.NoError(s.Backend.OpenAccounts())
	s.NoError(s.Backend.AccountManager().InitKeystore(nodeConfig.KeyStoreDir))

	s.Require().NoError(s.Backend.StartNodeWithAccountAndConfig(account, password, nodeConfig, subaccs))
}

func (s *BackendTestSuite) LogoutAndStop() {
	s.NoError(s.Backend.Logout())
	s.StopTestBackend()
}

// StopTestBackend stops the node.
func (s *BackendTestSuite) StopTestBackend() {
	s.True(s.Backend.IsNodeRunning())
	s.NoError(s.Backend.StopNode())
	s.False(s.Backend.IsNodeRunning())
}

// RestartTestNode restarts a currently running node.
func (s *BackendTestSuite) RestartTestNode() {
	s.True(s.Backend.IsNodeRunning())
	s.Require().NoError(s.Backend.RestartNode())
	s.True(s.Backend.IsNodeRunning())
}

// WhisperService returns a reference to the Whisper service.
func (s *BackendTestSuite) WhisperService() *whisper.Whisper {
	whisperService, err := s.Backend.StatusNode().WhisperService()
	s.NoError(err)
	s.NotNil(whisperService)

	return whisperService
}

// Transactor returns a reference to the Transactor.
func (s *BackendTestSuite) Transactor() *transactions.Transactor {
	return s.Backend.Transactor()
}

func importTestAccounts(keyStoreDir string) (err error) {
	logger.Debug("Import accounts to", "dir", keyStoreDir)

	err = utils.ImportTestAccount(keyStoreDir, utils.GetAccount1PKFile())
	if err != nil {
		return
	}

	return utils.ImportTestAccount(keyStoreDir, utils.GetAccount2PKFile())
}
