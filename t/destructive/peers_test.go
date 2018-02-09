package destructive

import (
	"testing"

	"github.com/status-im/status-go/geth/api"
	"github.com/status-im/status-go/geth/log"
	"github.com/status-im/status-go/t/e2e"
	. "github.com/status-im/status-go/t/utils"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/stretchr/testify/suite"
)

func TestPeersSuiteLinkUpDown(t *testing.T) {
	suite.Run(t, &PeersTestSuite{tester: new(LinkUpDownTester)})
}

type PeersTestSuite struct {
	suite.Suite

	backend *api.StatusBackend
	tester  *LinkUpDownTester
}

func (s *PeersTestSuite) SetupTest() {
	netid := GetNetworkID()
	s.Require().NotEqual(0, netid, "test will work only on public network")
	s.backend = api.NewStatusBackend()
	config, err := e2e.MakeTestNodeConfig(GetNetworkID())
	// we need to enable atleast 1 protocol, otherwise peers won't connect
	config.LightEthConfig.Enabled = false
	config.WhisperConfig.Enabled = true
	s.Require().NoError(err)
	done, err := s.backend.StartNode(config)
	s.Require().NoError(err)
	<-done
}

func (s *PeersTestSuite) TearDownTest() {
	done, err := s.backend.StopNode()
	s.Require().NoError(err)
	<-done
}

func (s *PeersTestSuite) TestStaticPeersReconnect() {
	events := make(chan *p2p.PeerEvent, 10)
	node, err := s.backend.NodeManager().Node()
	s.Require().NoError(err)

	node.Server().SubscribeEvents(events)
	peers := map[discover.NodeID]struct{}{}
	for ev := range events {
		if ev.Type == p2p.PeerEventTypeAdd {
			log.Info("tests", "event", ev)
			peers[ev.Peer] = struct{}{}
		}
		// rewrite it with timeout, and wait till peers number won't be changing for some time
		if len(peers) == 2 {
			break
		}
	}
	s.Require().NoError(s.tester.Setup())
	for ev := range events {
		if ev.Type == p2p.PeerEventTypeDrop {
			log.Info("tests", "event", ev)
			delete(peers, ev.Peer)
		}
		if len(peers) == 0 {
			break
		}
	}
	s.Require().NoError(s.tester.TearDown())
	// disconnects would be due to network err
	for ev := range events {
		if ev.Type == p2p.PeerEventTypeAdd {
			log.Info("tests", "event", ev)
			peers[ev.Peer] = struct{}{}
		}
		if len(peers) == 2 {
			break
		}
	}
}
