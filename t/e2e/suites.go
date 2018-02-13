package e2e

import (
	"github.com/ethereum/go-ethereum/les"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
	"github.com/status-im/status-go/geth/api"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/log"
	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/signal"
	"github.com/status-im/status-go/geth/transactions"
	. "github.com/status-im/status-go/t/utils" //nolint: golint
	"github.com/stretchr/testify/suite"
)

// NodeManagerTestSuite defines a test suit with NodeManager.
type NodeManagerTestSuite struct {
	suite.Suite
	NodeManager *node.NodeManager
}

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

// StartTestNode initiazes a NodeManager instances with configuration retrieved
// from the test config.
func (s *NodeManagerTestSuite) StartTestNode(opts ...TestNodeOption) {
	nodeConfig, err := MakeTestNodeConfig(GetNetworkID())
	s.NoError(err)

	// Apply any options altering node config.
	for i := range opts {
		opts[i](nodeConfig)
	}

	// import account keys
	s.NoError(importTestAccounts(nodeConfig.KeyStoreDir))

	s.False(s.NodeManager.IsNodeRunning())
	s.NoError(s.NodeManager.StartNode(nodeConfig))
	s.True(s.NodeManager.IsNodeRunning())
}

// StopTestNode attempts to stop initialized NodeManager.
func (s *NodeManagerTestSuite) StopTestNode() {
	s.NotNil(s.NodeManager)
	s.True(s.NodeManager.IsNodeRunning())
	s.NoError(s.NodeManager.StopNode())
	s.False(s.NodeManager.IsNodeRunning())
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
	whisperService, err := s.Backend.NodeManager().WhisperService()
	s.NoError(err)
	s.NotNil(whisperService)

	return whisperService
}

// LightEthereumService returns a reference to the LES service.
func (s *BackendTestSuite) LightEthereumService() *les.LightEthereum {
	lightEthereum, err := s.Backend.NodeManager().LightEthereumService()
	s.NoError(err)
	s.NotNil(lightEthereum)

	return lightEthereum
}

// TxQueueManager returns a reference to the TxQueueManager.
func (s *BackendTestSuite) TxQueueManager() *transactions.Manager {
	return s.Backend.TxQueueManager()
}

func importTestAccounts(keyStoreDir string) (err error) {
	log.Debug("Import accounts to", keyStoreDir)

	err = common.ImportTestAccount(keyStoreDir, GetAccount1PKFile())
	if err != nil {
		return
	}

	return common.ImportTestAccount(keyStoreDir, GetAccount2PKFile())
}
