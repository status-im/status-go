//go:build !race

package filter

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/suite"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"go.uber.org/zap"
	"golang.org/x/exp/maps"
)

func TestFilterApiSuite(t *testing.T) {
	suite.Run(t, new(FilterApiTestSuite))
}

type FilterApiTestSuite struct {
	filter.FilterTestSuite
	msgRcvd chan bool
}

func (s *FilterApiTestSuite) SetupTest() {
	s.FilterTestSuite.SetupTest()
	s.Log.Info("SetupTest()")
}

func (s *FilterApiTestSuite) TearDownTest() {
	s.FilterTestSuite.TearDownTest()
}

func (s *FilterApiTestSuite) TestSubscribe() {
	contentFilter := protocol.ContentFilter{PubsubTopic: s.TestTopic, ContentTopics: protocol.NewContentTopicSet(s.TestContentTopic)}

	// We have one full node already created in SetupTest(),
	// create another one
	fullNodeData2 := s.GetWakuFilterFullNode(s.TestTopic, true)
	s.ConnectToFullNode(s.LightNode, fullNodeData2.FullNode)
	//s.ConnectHosts(s.FullNodeHost, fullNodeData2.FullNodeHost)
	peers := []peer.ID{s.FullNodeHost.ID(), fullNodeData2.FullNodeHost.ID()}
	s.Log.Info("FullNodeHost IDs:", zap.Any("peers", peers))
	// Make sure IDs are different
	//s.Require().True(peers[0] != peers[1])
	apiConfig := FilterConfig{MaxPeers: 2}

	s.Require().Equal(apiConfig.MaxPeers, 2)
	s.Require().Equal(contentFilter.PubsubTopic, s.TestTopic)
	ctx, cancel := context.WithCancel(context.Background())
	s.Log.Info("About to perform API Subscribe()")
	params := subscribeParameters{300 * time.Second, 1024}
	apiSub, err := Subscribe(ctx, s.LightNode, contentFilter, apiConfig, s.Log, &params)
	s.Require().NoError(err)
	s.Require().Equal(apiSub.ContentFilter, contentFilter)
	s.Log.Info("Subscribed")

	s.Require().Len(apiSub.subs, 2)
	for sub := range apiSub.subs {
		s.Log.Info("SubDetails:", zap.String("id", sub))
	}
	subsArray := maps.Keys(apiSub.subs)
	s.Require().True(subsArray[0] != subsArray[1])
	// Publish msg and confirm it's received twice because of multiplexing
	s.PublishMsg(&filter.WakuMsg{PubSubTopic: s.TestTopic, ContentTopic: s.TestContentTopic, Payload: "Test msg"})
	cnt := 0
	for msg := range apiSub.DataCh {
		s.Log.Info("Received msg:", zap.Int("cnt", cnt), zap.String("payload", string(msg.Message().Payload)))
		cnt++
		break
	}
	s.Require().Equal(cnt, 1)

	//Verify HealthCheck
	subs := s.LightNode.Subscriptions()
	s.Require().Equal(2, len(subs))

	s.Log.Info("stopping full node", zap.Stringer("id", fullNodeData2.FullNodeHost.ID()))
	fullNodeData3 := s.GetWakuFilterFullNode(s.TestTopic, true)

	s.ConnectToFullNode(s.LightNode, fullNodeData3.FullNode)

	fullNodeData2.FullNode.Stop()
	fullNodeData2.FullNodeHost.Close()
	time.Sleep(2 * time.Second)
	subs = s.LightNode.Subscriptions()

	s.Require().Equal(2, len(subs))

	for _, sub := range subs {
		s.Require().NotEqual(fullNodeData2.FullNodeHost.ID(), sub.PeerID)
	}

	apiSub.Unsubscribe(contentFilter)
	cancel()
	for range apiSub.DataCh {
	}
	s.Log.Info("DataCh is closed")

}

func (s *FilterApiTestSuite) OnNewEnvelope(env *protocol.Envelope) error {
	if env.Message().ContentTopic == s.ContentFilter.ContentTopicsList()[0] {
		s.Log.Info("received message via filter")
		s.msgRcvd <- true
	} else {
		s.Log.Info("received message via filter but doesn't match contentTopic")
	}
	return nil
}

func (s *FilterApiTestSuite) TestFilterManager() {
	ctx, cancel := context.WithCancel(context.Background())

	testPubsubTopic := s.TestTopic
	contentTopicBytes := make([]byte, 4)
	_, err := rand.Read(contentTopicBytes)

	s.Require().NoError(err)

	s.ContentFilter = protocol.ContentFilter{
		PubsubTopic:   testPubsubTopic,
		ContentTopics: protocol.NewContentTopicSet("/test/filtermgr" + hex.EncodeToString(contentTopicBytes) + "/topic/proto"),
	}

	s.msgRcvd = make(chan bool, 1)

	s.Log.Info("creating filterManager")
	fm := NewFilterManager(ctx, s.Log, 2, s, s.LightNode)
	fm.filterSubBatchDuration = 1 * time.Second
	fm.onlineChecker.SetOnline(true)
	fID := uuid.NewString()
	fm.SubscribeFilter(fID, s.ContentFilter)
	time.Sleep(2 * time.Second)

	// Ensure there is at least 1 active filter subscription
	subscriptions := s.LightNode.Subscriptions()
	s.Require().Greater(len(subscriptions), 0)

	s.Log.Info("publishing msg")

	s.PublishMsg(&filter.WakuMsg{
		Payload:      "filtermgr testMsg",
		ContentTopic: s.ContentFilter.ContentTopicsList()[0],
		PubSubTopic:  testPubsubTopic,
	})
	t := time.NewTicker(2 * time.Second)
	select {
	case received := <-s.msgRcvd:
		s.Require().True(received)
		s.Log.Info("unsubscribe 1")
	case <-t.C:
		s.Log.Error("timed out waiting for message")
		s.Fail("timed out waiting for message")
	}
	// Mock peers going down
	s.LightNodeHost.Peerstore().RemovePeer(s.FullNodeHost.ID())

	fm.OnConnectionStatusChange("", false, 0)
	time.Sleep(2 * time.Second)
	fm.OnConnectionStatusChange("", true, 0)
	s.ConnectToFullNode(s.LightNode, s.FullNode)
	time.Sleep(3 * time.Second)

	// Ensure there is at least 1 active filter subscription
	subscriptions = s.LightNode.Subscriptions()
	s.Require().Greater(len(subscriptions), 0)
	s.Log.Info("publish message 2")

	// Ensure that messages are retrieved with a fresh sub
	s.PublishMsg(&filter.WakuMsg{
		Payload:      "filtermgr testMsg2",
		ContentTopic: s.ContentFilter.ContentTopicsList()[0],
		PubSubTopic:  testPubsubTopic,
	})
	t = time.NewTicker(2 * time.Second)

	select {
	case received := <-s.msgRcvd:
		s.Require().True(received)
		s.Log.Info("received message 2")
	case <-t.C:
		s.Log.Error("timed out waiting for message 2")
		s.Fail("timed out waiting for message 2")
	}

	fm.UnsubscribeFilter(fID)
	cancel()
}
