package whisper

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/signal"
	"github.com/stretchr/testify/suite"
)

func TestWhisperExtensionSuite(t *testing.T) {
	suite.Run(t, new(WhisperExtensionSuite))
}

type WhisperExtensionSuite struct {
	suite.Suite

	nodes []*node.StatusNode
}

func (s *WhisperExtensionSuite) SetupTest() {
	s.nodes = make([]*node.StatusNode, 2)
	for i := range s.nodes {
		dir, err := ioutil.TempDir("", "test-shhext-")
		s.NoError(err)
		// network id is irrelevant
		cfg, err := params.NewNodeConfig(dir, "", 777)
		cfg.LightEthConfig.Enabled = false
		cfg.Name = fmt.Sprintf("test-shhext-%d", i)
		s.Require().NoError(err)
		s.nodes[i] = node.New()
		s.Require().NoError(s.nodes[i].Start(cfg))
	}
}

func (s *WhisperExtensionSuite) TestSentSignal() {
	node1 := s.nodes[0].GethNode()
	s.NotNil(node1)
	node2 := s.nodes[1].GethNode()
	s.NotNil(node2)
	node1.Server().AddPeer(node2.Server().Self())
	confirmed := make(chan common.Hash, 1)
	signal.SetDefaultNodeNotificationHandler(func(rawSignal string) {
		var sg struct {
			Type  string
			Event json.RawMessage
		}
		s.NoError(json.Unmarshal([]byte(rawSignal), &sg))

		if sg.Type == signal.EventEnvelopeSent {
			var event signal.EnvelopeSignal
			s.NoError(json.Unmarshal(sg.Event, &event))
			confirmed <- event.Hash
		}
	})
	defer signal.ResetDefaultNodeNotificationHandler()
	client := s.nodes[0].RPCClient()
	s.NotNil(client)
	var symID string
	s.NoError(client.Call(&symID, "shh_newSymKey"))
	msg := whisper.NewMessage{
		SymKeyID:  symID,
		PowTarget: whisper.DefaultMinimumPoW,
		PowTime:   200,
		Topic:     whisper.TopicType{0x01, 0x01, 0x01, 0x01},
		Payload:   []byte("hello"),
		TTL:       5,
	}
	var hash common.Hash
	s.NoError(client.Call(&hash, "shhext_post", msg))
	s.NotEqual(common.Hash{}, hash)
	select {
	case conf := <-confirmed:
		s.Equal(hash, conf)
	case <-time.After(5 * time.Second):
		s.Fail("timed out while waiting for confirmation")
	}
}

func (s *WhisperExtensionSuite) TestExpiredSignal() {
	expired := make(chan common.Hash, 1)
	signal.SetDefaultNodeNotificationHandler(func(rawSignal string) {
		var sg struct {
			Type  string
			Event json.RawMessage
		}
		fmt.Println(rawSignal)
		s.NoError(json.Unmarshal([]byte(rawSignal), &sg))

		if sg.Type == signal.EventEnvelopeExpired {
			var event signal.EnvelopeSignal
			s.NoError(json.Unmarshal(sg.Event, &event))
			expired <- event.Hash
		}
	})
	defer signal.ResetDefaultNodeNotificationHandler()
	client := s.nodes[0].RPCClient()
	s.NotNil(client)
	var symID string
	s.NoError(client.Call(&symID, "shh_newSymKey"))
	msg := whisper.NewMessage{
		SymKeyID:  symID,
		PowTarget: whisper.DefaultMinimumPoW,
		PowTime:   200,
		TTL:       1,
		Topic:     whisper.TopicType{0x01, 0x01, 0x01, 0x01},
		Payload:   []byte("hello"),
	}
	var hash common.Hash
	s.NoError(client.Call(&hash, "shhext_post", msg))
	s.NotEqual(common.Hash{}, hash)
	select {
	case exp := <-expired:
		s.Equal(hash, exp)
	case <-time.After(5 * time.Second):
		s.Fail("timed out while waiting for expiration")
	}
}

func (s *WhisperExtensionSuite) TearDown() {
	for _, n := range s.nodes {
		cfg := n.Config()
		s.NotNil(cfg)
		s.NoError(n.Stop())
		s.NoError(os.Remove(cfg.DataDir))
	}
}
