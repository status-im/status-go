package e2e

import (
	"github.com/ethereum/go-ethereum/log"

	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"

	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/sign"
	"github.com/status-im/status-go/signal"
	. "github.com/status-im/status-go/t/utils" //nolint: golint
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

func init() {
	for id := range TestNetworkNames {
		nodeConfig, err := MakeTestNodeConfig(id)
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
	nodeConfig, err := MakeTestNodeConfig(GetNetworkID())
	s.NoError(err)

	// Apply any options altering node config.
	for i := range opts {
		opts[i](nodeConfig)
	}

	// import account keys
	s.NoError(importTestAccounts(nodeConfig.KeyStoreDir))

	s.False(s.StatusNode.IsRunning())
	s.NoError(s.StatusNode.Start(nodeConfig))
	s.True(s.StatusNode.IsRunning())
}

// StopTestNode attempts to stop initialized StatusNode.
func (s *StatusNodeTestSuite) StopTestNode() {
	s.NotNil(s.StatusNode)
	s.True(s.StatusNode.IsRunning())
	s.NoError(s.StatusNode.Stop())
	s.False(s.StatusNode.IsRunning())
}

// BackendTestSuite is a test suite with api.StatusBackend initialized
// and a few utility methods to start and stop node or get various services.
type BackendTestSuite struct {
	suite.Suite
	Backend *api.StatusBackend
}

// SetupTest initializes Backend.
func (s *BackendTestSuite) SetupTest() {
	s.Backend = api.NewStatusBackend()
	s.NotNil(s.Backend)
}

// TearDownTest cleans up the packages state.
func (s *BackendTestSuite) TearDownTest() {
	signal.ResetDefaultNodeNotificationHandler()
}

// StartTestBackend imports some keys and starts a node.
func (s *BackendTestSuite) StartTestBackend(opts ...TestNodeOption) {
	nodeConfig, err := MakeTestNodeConfig(GetNetworkID())
	s.NoError(err)

	// Apply any options altering node config.
	for i := range opts {
		opts[i](nodeConfig)
	}

	// import account keys
	s.NoError(importTestAccounts(nodeConfig.KeyStoreDir))

	// start node
	s.False(s.Backend.IsNodeRunning())
	s.NoError(s.Backend.StartNode(nodeConfig))
	s.True(s.Backend.IsNodeRunning())
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
	s.NoError(s.Backend.RestartNode())
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

// PendingSignRequests returns a reference to PendingSignRequests.
func (s *BackendTestSuite) PendingSignRequests() *sign.PendingRequests {
	return s.Backend.PendingSignRequests()
}

func importTestAccounts(keyStoreDir string) (err error) {
	logger.Debug("Import accounts to", "dir", keyStoreDir)

	err = ImportTestAccount(keyStoreDir, GetAccount1PKFile())
	if err != nil {
		return
	}

	return ImportTestAccount(keyStoreDir, GetAccount2PKFile())
}
