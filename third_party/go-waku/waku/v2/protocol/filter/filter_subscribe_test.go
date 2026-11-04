package filter

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/waku-org/go-waku/waku/v2/protocol/filter/pb"

	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
)

func (s *FilterTestSuite) TestWakuFilter() {
	// Initial subscribe
	s.subscribe(s.TestTopic, s.TestContentTopic, s.FullNodeHost.ID())

	// Should be received
	s.waitForMsg(&WakuMsg{s.TestTopic, s.TestContentTopic, "first"})

	// Wrong content topic
	s.waitForTimeout(&WakuMsg{s.TestTopic, "TopicB", "second"})

	_, err := s.LightNode.Unsubscribe(s.ctx, s.ContentFilter, WithPeer(s.FullNodeHost.ID()))
	s.Require().NoError(err)

	// Should not receive after unsubscribe
	s.waitForTimeout(&WakuMsg{s.TestTopic, s.TestContentTopic, "third"})

	// Two new subscriptions with same [peer, contentFilter]
	s.subscribe(s.TestTopic, s.TestContentTopic, s.FullNodeHost.ID())

	secondSub := s.getSub(s.TestTopic, s.TestContentTopic, s.FullNodeHost.ID())

	// Assert that we have 2 subscriptions now
	s.Require().Equal(len(s.LightNode.Subscriptions()), 2)

	// Should be received on both subscriptions
	s.waitForMsg(&WakuMsg{s.TestTopic, s.TestContentTopic, "fourth"})

	s.waitForMsgFromChan(&WakuMsg{s.TestTopic, s.TestContentTopic, "fifth"}, secondSub[0].C)

	s.waitForMsg(nil)
	s.waitForMsgFromChan(nil, secondSub[0].C)

	// Unsubscribe from second sub only
	_, err = s.LightNode.UnsubscribeWithSubscription(s.ctx, secondSub[0])
	s.Require().NoError(err)

	// Should still receive
	s.waitForMsg(&WakuMsg{s.TestTopic, s.TestContentTopic, "sixth"})

	// Unsubscribe from first sub only
	_, err = s.LightNode.UnsubscribeWithSubscription(s.ctx, s.subDetails[0])
	s.Require().NoError(err)

	s.Require().Equal(len(s.LightNode.Subscriptions()), 0)

	// Should not receive after unsubscribe
	s.waitForTimeout(&WakuMsg{s.TestTopic, s.TestContentTopic, "seventh"})
}

func (s *FilterTestSuite) TestPubSubSingleContentTopic() {
	// Initial subscribe
	s.subscribe(s.TestTopic, s.TestContentTopic, s.FullNodeHost.ID())

	// Message should be received
	s.waitForMsg(&WakuMsg{s.TestTopic, s.TestContentTopic, "test_msg"})

	_, err := s.LightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestPubSubMultiContentTopic() {

	// Create test context
	s.ctx, s.ctxCancel = context.WithTimeout(context.Background(), 20*time.Second) // Test can't exceed 10 seconds

	messages := s.prepareData(3, false, true, false, nil)

	// Subscribe
	for _, m := range messages {
		s.subscribe(m.PubSubTopic, m.ContentTopic, s.FullNodeHost.ID())
	}

	// All messages should be received
	s.waitForMessages(messages)

	_, err := s.LightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestMultiPubSubMultiContentTopic() {

	// Create test context
	s.ctx, s.ctxCancel = context.WithTimeout(context.Background(), 20*time.Second) // Test can't exceed 20 seconds

	s.MakeWakuFilterLightNode()
	s.StartLightNode()

	s.MakeWakuFilterFullNode(s.TestTopic, true)

	// Connect nodes
	s.LightNodeHost.Peerstore().AddAddr(s.FullNodeHost.ID(), tests.GetHostAddress(s.FullNodeHost), peerstore.PermanentAddrTTL)
	err := s.LightNodeHost.Peerstore().AddProtocols(s.FullNodeHost.ID(), FilterSubscribeID_v20beta1)
	s.Require().NoError(err)

	messages := s.prepareData(2, true, true, false, nil)

	// Subscribe
	for _, m := range messages {
		s.subDetails = append(s.subDetails, s.getSub(m.PubSubTopic, m.ContentTopic, s.FullNodeHost.ID())...)
		s.Log.Info("Subscribing ", zap.String("PubSubTopic", m.PubSubTopic))
		_, err := s.relayNode.Subscribe(context.Background(), protocol.NewContentFilter(m.PubSubTopic))
		s.Require().NoError(err)
	}

	// Debug to see subscriptions in light node
	for _, sub := range s.subDetails {
		s.Log.Info("Light Node subscription ", zap.String("PubSubTopic", sub.ContentFilter.PubsubTopic), zap.String("ContentTopic", sub.ContentFilter.ContentTopicsList()[0]))
	}

	// All messages should be received
	s.waitForMessages(messages)

	_, err = s.LightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestPubSubMultiOverlapContentTopic() {

	// Create test context
	s.ctx, s.ctxCancel = context.WithTimeout(context.Background(), 20*time.Second) // Test can't exceed 20 seconds

	messages := s.prepareData(10, false, true, true, nil)

	// Subscribe
	for _, m := range messages {
		s.subscribe(m.PubSubTopic, m.ContentTopic, s.FullNodeHost.ID())
	}

	// All messages should be received
	s.waitForMessages(messages)

	_, err := s.LightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestSubscriptionRefresh() {

	messages := s.prepareData(2, false, false, true, nil)

	// Initial subscribe
	s.subscribe(s.TestTopic, s.TestContentTopic, s.FullNodeHost.ID())

	// Repeat the same subscribe
	s.subscribe(s.TestTopic, s.TestContentTopic, s.FullNodeHost.ID())

	// Both messages should be received
	s.waitForMessages(messages)

	_, err := s.LightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestContentTopicsLimit() {
	var maxContentTopics = pb.MaxContentTopicsPerRequest

	// Create test context
	s.ctx, s.ctxCancel = context.WithTimeout(context.Background(), 20*time.Second) // Test can't exceed 10 seconds

	// Detect existing content topics from previous test
	if len(s.ContentFilter.PubsubTopic) > 0 {
		existingTopics := len(s.ContentFilter.ContentTopicsList())
		if existingTopics > 0 {
			maxContentTopics = maxContentTopics - existingTopics
		}
	}

	messages := s.prepareData(maxContentTopics+1, false, true, true, nil)

	// Subscribe
	for _, m := range messages[:len(messages)-1] {
		s.subscribe(m.PubSubTopic, m.ContentTopic, s.FullNodeHost.ID())
	}

	// All messages within limit should get received
	s.waitForMessages(messages[:len(messages)-1])

	// Adding over the limit contentTopic should fail
	for _, sub := range s.subDetails {
		if sub.ContentFilter.PubsubTopic == messages[len(messages)-1].PubSubTopic {
			sub.Add(messages[len(messages)-1].ContentTopic)
			_, err := s.LightNode.Subscribe(s.ctx, sub.ContentFilter, WithPeer(s.FullNodeHost.ID()))
			s.Require().Error(err)
		}
	}

	// Unsubscribe for cleanup
	for _, m := range messages {
		_ = s.unsubscribe(m.PubSubTopic, m.ContentTopic, s.FullNodeHost.ID())
	}

	_, err := s.LightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestSubscribeErrorHandling() {
	var messages []WakuMsg

	// Prepare data
	messages = append(messages, WakuMsg{
		PubSubTopic:  "",
		ContentTopic: s.TestContentTopic,
		Payload:      "N/A",
	})

	messages = append(messages, WakuMsg{
		PubSubTopic:  s.TestTopic,
		ContentTopic: "",
		Payload:      "N/A",
	})

	// Subscribe with empty pubsub
	s.ContentFilter = protocol.ContentFilter{PubsubTopic: messages[0].PubSubTopic, ContentTopics: protocol.NewContentTopicSet(messages[0].ContentTopic)}
	_, err := s.LightNode.Subscribe(s.ctx, s.ContentFilter, WithPeer(s.FullNodeHost.ID()))
	s.Require().Error(err)

	// Subscribe with empty content topic
	s.ContentFilter = protocol.ContentFilter{PubsubTopic: messages[1].PubSubTopic, ContentTopics: protocol.NewContentTopicSet(messages[1].ContentTopic)}
	_, err = s.LightNode.Subscribe(s.ctx, s.ContentFilter, WithPeer(s.FullNodeHost.ID()))
	s.Require().Error(err)

}

func (s *FilterTestSuite) TestMultipleFullNodeSubscriptions() {
	log := utils.Logger()
	s.Log = log
	s.wg = &sync.WaitGroup{}

	// Create test context
	s.ctx, s.ctxCancel = context.WithTimeout(context.Background(), 10*time.Second) // Test can't exceed 10 seconds

	fullNodeIDHex := make([]byte, hex.EncodedLen(len([]byte(s.FullNodeHost.ID()))))
	_ = hex.Encode(fullNodeIDHex, []byte(s.FullNodeHost.ID()))

	s.Log.Info("Already subscribed to", zap.String("fullNode", string(fullNodeIDHex)))

	// This will overwrite values with the second node info
	s.MakeWakuFilterFullNode(s.TestTopic, false)

	// Connect to second full and relay node
	s.LightNodeHost.Peerstore().AddAddr(s.FullNodeHost.ID(), tests.GetHostAddress(s.FullNodeHost), peerstore.PermanentAddrTTL)
	err := s.LightNodeHost.Peerstore().AddProtocols(s.FullNodeHost.ID(), FilterSubscribeID_v20beta1)
	s.Require().NoError(err)

	fullNodeIDHex = make([]byte, hex.EncodedLen(len([]byte(s.FullNodeHost.ID()))))
	_ = hex.Encode(fullNodeIDHex, []byte(s.FullNodeHost.ID()))

	s.Log.Info("Subscribing to second", zap.String("fullNode", string(fullNodeIDHex)))

	// Subscribe to the second full node
	s.ContentFilter = protocol.ContentFilter{PubsubTopic: s.TestTopic, ContentTopics: protocol.NewContentTopicSet(s.TestContentTopic)}
	_, err = s.LightNode.Subscribe(s.ctx, s.ContentFilter, WithPeer(s.FullNodeHost.ID()))
	s.Require().NoError(err)

	_, err = s.LightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)
}

func (s *FilterTestSuite) TestSubscribeMultipleLightNodes() {

	// Create test context
	s.ctx, s.ctxCancel = context.WithTimeout(context.Background(), 10*time.Second) // Test can't exceed 10 seconds

	lightNodeData := s.GetWakuFilterLightNode()
	lightNode2 := lightNodeData.LightNode
	err := lightNode2.Start(context.Background())
	s.Require().NoError(err)

	// Connect node2
	lightNode2.h.Peerstore().AddAddr(s.FullNodeHost.ID(), tests.GetHostAddress(s.FullNodeHost), peerstore.PermanentAddrTTL)

	messages := s.prepareData(2, true, true, true, nil)

	// Subscribe separately: light node 1 -> full node
	contentFilter := protocol.ContentFilter{PubsubTopic: messages[0].PubSubTopic, ContentTopics: protocol.NewContentTopicSet(messages[0].ContentTopic)}
	_, err = s.LightNode.Subscribe(s.ctx, contentFilter, WithPeer(s.FullNodeHost.ID()))
	s.Require().NoError(err)

	// Subscribe separately: light node 2 -> full node
	contentFilter2 := protocol.ContentFilter{PubsubTopic: messages[1].PubSubTopic, ContentTopics: protocol.NewContentTopicSet(messages[1].ContentTopic)}
	_, err = lightNode2.Subscribe(s.ctx, contentFilter2, WithPeer(s.FullNodeHost.ID()))
	s.Require().NoError(err)

	// Unsubscribe
	_, err = s.LightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

	_, err = lightNode2.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestSubscribeFullNode2FullNode() {

	var (
		testTopic        = "/waku/2/go/filter/test2"
		testContentTopic = "TopicB"
	)

	// Create test context
	s.ctx, s.ctxCancel = context.WithTimeout(context.Background(), 10*time.Second)

	nodeData := s.GetWakuFilterFullNode(testTopic, false)
	fullNode2 := nodeData.FullNode

	// Connect nodes
	fullNode2.h.Peerstore().AddAddr(s.FullNodeHost.ID(), tests.GetHostAddress(s.FullNodeHost), peerstore.PermanentAddrTTL)

	// Get stream
	stream, err := fullNode2.h.NewStream(s.ctx, s.FullNodeHost.ID(), FilterSubscribeID_v20beta1)
	s.Require().NoError(err)

	// Prepare subscribe request
	subscribeRequest := &pb.FilterSubscribeRequest{
		FilterSubscribeType: pb.FilterSubscribeRequest_SUBSCRIBE,
		PubsubTopic:         &testTopic,
		ContentTopics:       []string{testContentTopic},
	}

	// Subscribe full node 2 -> full node 1
	fullNode2.subscribe(s.ctx, stream, subscribeRequest)

	// Check the pubsub topic related to the first node is stored within the second node
	pubsubTopics, hasTopics := fullNode2.subscriptions.Get(s.FullNodeHost.ID())
	s.Require().True(hasTopics)

	// Check the pubsub topic is what we have set
	contentTopics, hasTestPubsubTopic := pubsubTopics[testTopic]
	s.Require().True(hasTestPubsubTopic)

	// Check the content topic is what we have set
	_, hasTestContentTopic := contentTopics[testContentTopic]
	s.Require().True(hasTestContentTopic)

}

func (s *FilterTestSuite) TestFilterSubscription() {
	contentFilter := protocol.ContentFilter{PubsubTopic: s.TestTopic, ContentTopics: protocol.NewContentTopicSet(s.TestContentTopic)}

	// Subscribe
	s.subscribe(s.TestTopic, s.TestContentTopic, s.FullNodeHost.ID())

	// Returns no error and SubscriptionDetails for existing subscription
	_, err := s.LightNode.FilterSubscription(s.FullNodeHost.ID(), contentFilter)
	s.Require().NoError(err)

	otherFilter := protocol.ContentFilter{PubsubTopic: "34583495", ContentTopics: protocol.NewContentTopicSet("sjfa402")}

	// Returns error and nil SubscriptionDetails for non existent subscription
	nonSubscription, err := s.LightNode.FilterSubscription(s.FullNodeHost.ID(), otherFilter)
	s.Require().Error(err)
	s.Require().Nil(nonSubscription)

	// Create new host/peer - not related to any node
	host, err := tests.MakeHost(context.Background(), 54321, rand.Reader)
	s.Require().NoError(err)

	// Returns error and nil SubscriptionDetails for unrelated host/peer
	nonSubscription, err = s.LightNode.FilterSubscription(host.ID(), contentFilter)
	s.Require().Error(err)
	s.Require().Nil(nonSubscription)

}

func (s *FilterTestSuite) TestHandleFilterSubscribeOptions() {
	contentFilter := protocol.ContentFilter{PubsubTopic: s.TestTopic, ContentTopics: protocol.NewContentTopicSet(s.TestContentTopic)}

	// Subscribe
	s.subscribe(s.TestTopic, s.TestContentTopic, s.FullNodeHost.ID())

	// With valid peer
	opts := []FilterSubscribeOption{WithPeer(s.FullNodeHost.ID())}

	// Positive case
	_, _, err := s.LightNode.handleFilterSubscribeOptions(s.ctx, contentFilter, opts)
	s.Require().NoError(err)

	addr := s.FullNodeHost.Addrs()[0]

	// Combine mutually exclusive options
	opts = []FilterSubscribeOption{WithPeer(s.FullNodeHost.ID()), WithPeerAddr(addr)}

	// Should fail on wrong option combination
	_, _, err = s.LightNode.handleFilterSubscribeOptions(s.ctx, contentFilter, opts)
	s.Require().Error(err)

}
