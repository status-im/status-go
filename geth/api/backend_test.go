package api_test

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/les"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/geth/api"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
	. "github.com/status-im/status-go/geth/testing"
	"github.com/stretchr/testify/suite"
)

func TestBackendTestSuite(t *testing.T) {
	suite.Run(t, new(BackendTestSuite))
}

type BackendTestSuite struct {
	BaseTestSuite
	backend *api.StatusBackend
}

func (s *BackendTestSuite) SetupTest() {
	require := s.Require()
	backend := api.NewStatusBackend()
	require.NotNil(backend)
	require.IsType(&api.StatusBackend{}, backend)
	s.backend = backend
	s.NodeManager = backend.NodeManager()
}

func (s *BackendTestSuite) StartTestBackend(networkID int) {
	require := s.Require()
	require.NotNil(s.backend)

	nodeConfig, err := MakeTestNodeConfig(networkID)
	require.NoError(err)

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
	require.NoError(s.backend.StopNode())
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
	err = s.backend.StopNode()
	if s.Error(err) {
		require.IsType(node.ErrNoRunningNode, err)
	}

	require.False(s.backend.IsNodeRunning())
	nodeStarted, err := s.backend.StartNode(nodeConfig)
	require.NoError(err)

	<-nodeStarted // wait till node is started
	require.True(s.backend.IsNodeRunning())

	// try starting another node (w/o stopping the previously started node)
	_, err = s.backend.StartNode(nodeConfig)
	if s.Error(err) {
		require.IsType(node.ErrNodeAlreadyExists, err)
	}

	// now stop node, and make sure that a new node, on different network can be started
	err = s.backend.StopNode()
	require.NoError(err)

	// start new node with exactly the same config
	require.False(s.backend.IsNodeRunning())
	nodeStarted, err = s.backend.StartNode(nodeConfig)
	require.NoError(err)
	defer s.StopTestNode()

	<-nodeStarted
	require.True(s.backend.IsNodeRunning())
}

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

	s.FirstBlockHash("0x41941023680923e0fe4d74a34bdac8141f2540e3ae90623718e47d66d1ca4a2d")

	// now stop node, and make sure that a new node, on different network can be started
	err = s.backend.StopNode()
	require.NoError(err)

	// start new node with completely different config
	nodeConfig, err = MakeTestNodeConfig(params.RinkebyNetworkID)
	require.NoError(err)

	require.False(s.backend.IsNodeRunning())
	nodeStarted, err = s.backend.StartNode(nodeConfig)
	require.NoError(err)

	<-nodeStarted
	require.True(s.backend.IsNodeRunning())

	// make sure we are on another network indeed
	s.FirstBlockHash("0x6341fd3daf94b748c72ced5a5b26028f2474f5f00d824504e4fa37a75767e177")

	require.NoError(s.backend.StopNode())
}

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
	s.FirstBlockHash("0x6341fd3daf94b748c72ced5a5b26028f2474f5f00d824504e4fa37a75767e177")
}

func (s *BackendTestSuite) TestRestartNode() {
	require := s.Require()
	require.NotNil(s.backend)

	s.StartTestBackend(params.RinkebyNetworkID)
	defer s.StopTestBackend()

	s.FirstBlockHash("0x6341fd3daf94b748c72ced5a5b26028f2474f5f00d824504e4fa37a75767e177")

	s.True(s.backend.IsNodeRunning())
	nodeRestarted, err := s.backend.RestartNode()
	require.NoError(err)
	<-nodeRestarted
	s.True(s.backend.IsNodeRunning()) // new node, with previous config should be running

	// make sure we can read the first byte, and it is valid (for Rinkeby)
	s.FirstBlockHash("0x6341fd3daf94b748c72ced5a5b26028f2474f5f00d824504e4fa37a75767e177")
}
