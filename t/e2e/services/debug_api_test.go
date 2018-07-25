package services

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/p2p"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/params"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/t/helpers"
	. "github.com/status-im/status-go/t/utils"
)

func TestDebugAPISuite(t *testing.T) {
	s := new(DebugAPISuite)
	s.upstream = false
	suite.Run(t, s)
}

func TestDebugAPISuiteUpstream(t *testing.T) {
	s := new(DebugAPISuite)
	s.upstream = true
	suite.Run(t, s)
}

type DebugAPISuite struct {
	BaseJSONRPCSuite
	upstream bool
}

func (s *DebugAPISuite) TestAccessibleDebugAPIsUnexported() {
	if s.upstream && GetNetworkID() == params.StatusChainNetworkID {
		s.T().Skip()
		return
	}

	err := s.SetupTest(s.upstream, false, false)
	s.NoError(err)
	// Debug APIs should be unavailable
	s.AssertAPIMethodUnexported("debug_postSync")
	err = s.Backend.StopNode()
	s.NoError(err)

	err = s.SetupTest(s.upstream, false, true)
	s.NoError(err)
	defer func() {
		err := s.Backend.StopNode()
		s.NoError(err)
	}()
	// Debug APIs should be available
	s.AssertAPIMethodExported("debug_postSync")
}

func (s *DebugAPISuite) TestDebugPostSyncSuccess() {
	// Test upstream if that's not StatusChain
	if s.upstream && GetNetworkID() == params.StatusChainNetworkID {
		s.T().Skip()
		return
	}

	err := s.SetupTest(s.upstream, false, true)
	s.NoError(err)
	defer func() {
		err := s.Backend.StopNode()
		s.NoError(err)
	}()

	dir, err := ioutil.TempDir("", "test-debug")
	s.NoError(err)
	defer os.RemoveAll(dir) //nolint: errcheck
	s.addPeerToCurrentNode(dir)

	symID := s.generateSymKey()
	result := s.sendPostConfirmMessage(symID)

	var r struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
		Result hexutil.Bytes `json:"result"`
	}
	s.NoError(json.Unmarshal([]byte(result), &r))
	s.Empty(r.Error.Message)
	s.NotEmpty(r.Result)
}

// generateSymKey generates and stores a symetric key.
func (s *DebugAPISuite) generateSymKey() string {
	w, err := s.Backend.StatusNode().WhisperService()
	s.Require().NoError(err)
	symID, err := w.GenerateSymKey()
	s.Require().NoError(err)

	return symID
}

// sendPostConfirmMessage calls debug_postSync endpoint with valid
// parameters.
func (s *DebugAPISuite) sendPostConfirmMessage(symID string) string {
	req := whisper.NewMessage{
		SymKeyID:  symID,
		PowTarget: whisper.DefaultMinimumPoW,
		PowTime:   200,
		Topic:     whisper.TopicType{0x01, 0x01, 0x01, 0x01},
		Payload:   []byte("hello"),
	}
	body, err := json.Marshal(req)
	s.NoError(err)

	basicCall := fmt.Sprintf(
		`{"jsonrpc":"2.0","method":"debug_postSync","params":[%s],"id":67}`,
		body)

	return s.Backend.CallPrivateRPC(basicCall)
}

// addPeers adds a peer to the running node
func (s *DebugAPISuite) addPeerToCurrentNode(dir string) {
	s.Require().NotNil(s.Backend)
	node1 := s.Backend.StatusNode().GethNode()
	s.NotNil(node1)
	node2 := s.newPeer("test2", dir).GethNode()
	s.NotNil(node2)

	errCh := helpers.WaitForPeerAsync(s.Backend.StatusNode().Server(),
		node2.Server().Self().String(),
		p2p.PeerEventTypeAdd,
		time.Second*5)

	node1.Server().AddPeer(node2.Server().Self())
	require.NoError(s.T(), <-errCh)
}

// newNode creates, configures and starts a new peer.
func (s *DebugAPISuite) newPeer(name, dir string) *node.StatusNode {
	// network id is irrelevant
	cfg, err := params.NewNodeConfig(dir, "", params.FleetBeta, 777)
	s.Require().NoError(err)
	cfg.LightEthConfig.Enabled = false
	cfg.Name = name
	cfg.NetworkID = uint64(GetNetworkID())
	n := node.New()
	s.Require().NoError(n.Start(cfg))

	return n
}
