package api_test

import (
	"io/ioutil"
	"os"
	"testing"

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
