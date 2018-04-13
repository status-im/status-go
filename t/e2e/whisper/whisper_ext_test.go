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
	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/signal"
	"github.com/status-im/status-go/shhext"
	"github.com/stretchr/testify/suite"
)

func TestWhisperExtentionSuite(t *testing.T) {
	suite.Run(t, new(WhisperExtentionSuite))
}

type WhisperExtentionSuite struct {
	suite.Suite

	nodes []*node.StatusNode
}

func (s *WhisperExtentionSuite) SetupTest() {
	s.nodes = make([]*node.StatusNode, 2)
	for i := range s.nodes {
		dir, err := ioutil.TempDir("", "test-shhext-")
		s.NoError(err)
		// network id is irrelevant
		cfg, err := params.NewNodeConfig(dir, "", 777, true)
		cfg.LightEthConfig.Enabled = false
		cfg.Name = fmt.Sprintf("test-shhext-%d", i)
		s.Require().NoError(err)
		s.nodes[i] = node.New()
		s.Require().NoError(s.nodes[i].Start(cfg))
	}
	s.True(s.nodes[0].IsRunning())
	s.True(s.nodes[1].IsRunning())
	s.nodes[0].GethNode().Server().AddPeer(
		s.nodes[1].GethNode().Server().Self(),
	)
}

func (s *WhisperExtentionSuite) TestRecievedSignal() {
	confirmed := make(chan common.Hash, 1)
	signal.SetDefaultNodeNotificationHandler(func(rawSignal string) {
		var sg struct {
			Type  string
			Event json.RawMessage
		}
		s.NoError(json.Unmarshal([]byte(rawSignal), &sg))

		if sg.Type == signal.EventEnvelopeSent {
			var event shhext.EnvelopeSentSignal
			s.NoError(json.Unmarshal(sg.Event, &event))
			confirmed <- event.Hash
		}
	})
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
	}
	var hash common.Hash
	s.NoError(client.Call(&hash, "shhext_post", msg))
	s.NotEqual(common.Hash{}, hash)
	select {
	case conf := <-confirmed:
		s.Equal(hash, conf)
	case <-time.After(time.Second):
		s.Fail("timed out while waiting for confirmation")
	}
}

func (s *WhisperExtentionSuite) TearDown() {
	for _, n := range s.nodes {
		cfg := n.Config()
		s.NotNil(cfg)
		s.NoError(n.Stop())
		s.NoError(os.Remove(cfg.DataDir))
	}
}
