package services

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/params"
	"github.com/stretchr/testify/suite"

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

func (s *DebugAPISuite) TestAccessibleDebugAPIs() {
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
	// Debug APIs should be unavailable
	s.AssertAPIMethodUnexported("debug_postSync")

	// Debug  APIs should be available only for IPC
	s.AssertAPIMethodExportedPrivately("debug_postSync")
}

func (s *DebugAPISuite) TestDebugPostconfirmSuccess() {
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

	s.addPeerToCurrentNode()
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

	resp := s.Backend.CallRPC(`{"jsonrpc":"2.0","method":"shh_addSymKey",
			"params":["` + symID + `"],
			"id":1}`)
	type returnedIDResponse struct {
		Result string
		Error  interface{}
	}
	symkeyAddResp := returnedIDResponse{}
	err = json.Unmarshal([]byte(resp), &symkeyAddResp)
	s.NoError(err)

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
	body, _ := json.Marshal(req)

	basicCall := fmt.Sprintf(
		`{"jsonrpc":"2.0","method":"debug_postSync","params":[%s],"id":67}`,
		body)

	return s.Backend.CallPrivateRPC(basicCall)
}

// addPeers adds a peer to the running node
func (s *DebugAPISuite) addPeerToCurrentNode() {
	s.Require().NotNil(s.Backend)
	node1 := s.Backend.StatusNode().GethNode()
	s.NotNil(node1)
	node2 := s.newPeer("test2").GethNode()
	s.NotNil(node2)

	node1.Server().AddPeer(node2.Server().Self())
}

// newNode creates, configures and starts a new peer.
func (s *DebugAPISuite) newPeer(name string) *node.StatusNode {
	dir, err := ioutil.TempDir("", "test-shhext-")
	s.NoError(err)
	// network id is irrelevant
	cfg, err := params.NewNodeConfig(dir, "", 777)
	cfg.LightEthConfig.Enabled = false
	cfg.Name = name
	cfg.NetworkID = uint64(GetNetworkID())
	s.Require().NoError(err)
	n := node.New()
	s.Require().NoError(n.Start(cfg))

	return n
}
