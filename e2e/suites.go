package e2e

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/les"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/geth/api"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/signal"
	"github.com/stretchr/testify/suite"
)

// NodeManagerTestSuite defines a test suit with NodeManager.
type NodeManagerTestSuite struct {
	suite.Suite
	NodeManager       common.NodeManager
	nodeSyncCompleted bool
}

// EnsureNodeSync ensures that synchronization of the node is done once and that it
// is done properly else, the call will fail.
// FIXME(tiabc): BackendTestSuite contains the same method, let's sort it out?
func (s *NodeManagerTestSuite) EnsureNodeSync(forceResync ...bool) {
	if len(forceResync) > 0 && forceResync[0] {
		s.nodeSyncCompleted = false
	}

	if s.nodeSyncCompleted {
		return
	}

	require := s.Require()

	ethClient, err := s.NodeManager.LightEthereumService()
	require.NoError(err)
	require.NotNil(ethClient)

	sync := node.NewSyncPoll(ethClient)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	// Validate that synchronization failed because of time.
	syncError := sync.Poll(ctx)
	require.NoError(syncError)

	s.nodeSyncCompleted = true
}

// StartTestNode initiazes a NodeManager instances with configuration retrieved
// from the test config.
func (s *NodeManagerTestSuite) StartTestNode(networkID int, opts ...TestNodeOption) {
	nodeConfig, err := MakeTestNodeConfig(networkID)
	s.NoError(err)

	// Apply any options altering node config.
	for i := range opts {
		opts[i](nodeConfig)
	}

	// import account keys
	s.NoError(importTestAccouns(nodeConfig.KeyStoreDir))

	s.False(s.NodeManager.IsNodeRunning())
	nodeStarted, err := s.NodeManager.StartNode(nodeConfig)
	s.NoError(err)
	s.NotNil(nodeStarted)
	<-nodeStarted
	s.True(s.NodeManager.IsNodeRunning())
}

// StopTestNode attempts to stop initialized NodeManager.
func (s *NodeManagerTestSuite) StopTestNode() {
	s.NotNil(s.NodeManager)
	s.True(s.NodeManager.IsNodeRunning())
	nodeStopped, err := s.NodeManager.StopNode()
	s.NoError(err)
	<-nodeStopped
	s.False(s.NodeManager.IsNodeRunning())
}

// BackendTestSuite is a test suite with api.StatusBackend initialized
// and a few utility methods to start and stop node or get various services.
type BackendTestSuite struct {
	suite.Suite
	Backend           *api.StatusBackend
	nodeSyncCompleted bool
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
func (s *BackendTestSuite) StartTestBackend(networkID int, opts ...TestNodeOption) {
	nodeConfig, err := MakeTestNodeConfig(networkID)
	s.NoError(err)

	// Apply any options altering node config.
	for i := range opts {
		opts[i](nodeConfig)
	}

	// import account keys
	s.NoError(importTestAccouns(nodeConfig.KeyStoreDir))

	// start node
	s.False(s.Backend.IsNodeRunning())
	nodeStarted, err := s.Backend.StartNode(nodeConfig)
	s.NoError(err)
	<-nodeStarted
	s.True(s.Backend.IsNodeRunning())
}

// StopTestBackend stops the node.
func (s *BackendTestSuite) StopTestBackend() {
	s.True(s.Backend.IsNodeRunning())
	backendStopped, err := s.Backend.StopNode()
	s.NoError(err)
	<-backendStopped
	s.False(s.Backend.IsNodeRunning())
}

// RestartTestNode restarts a currently running node.
func (s *BackendTestSuite) RestartTestNode() {
	s.True(s.Backend.IsNodeRunning())
	nodeRestarted, err := s.Backend.RestartNode()
	s.NoError(err)
	<-nodeRestarted
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
func (s *BackendTestSuite) TxQueueManager() common.TxQueueManager {
	return s.Backend.TxQueueManager()
}

// EnsureNodeSync ensures that synchronization of the node is done once and that it
// is done properly else, the call will fail.
// FIXME(tiabc): NodeManagerTestSuite contains the same method, let's sort it out?
func (s *BackendTestSuite) EnsureNodeSync(forceResync ...bool) {
	if len(forceResync) > 0 && forceResync[0] {
		s.nodeSyncCompleted = false
	}

	if s.nodeSyncCompleted {
		return
	}

	require := s.Require()

	ethClient := s.LightEthereumService()
	sync := node.NewSyncPoll(ethClient)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	// Validate that synchronization failed because of time.
	syncError := sync.Poll(ctx)
	require.NoError(syncError)

	s.nodeSyncCompleted = true
}

func importTestAccouns(keyStoreDir string) (err error) {
	err = common.ImportTestAccount(keyStoreDir, "test-account1.pk")
	if err != nil {
		return
	}

	return common.ImportTestAccount(keyStoreDir, "test-account2.pk")
}
