package filter

import (
	"context"
	"crypto/rand"
	"strconv"
	"time"

	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/waku-org/go-waku/logging"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"go.uber.org/zap"
)

func (s *FilterTestSuite) TestUnsubscribeSingleContentTopic() {

	var newContentTopic = "TopicB"

	// Initial subscribe
	s.subscribe(s.TestTopic, s.TestContentTopic, s.FullNodeHost.ID())
	s.subscribe(s.TestTopic, newContentTopic, s.FullNodeHost.ID())

	// Message is possible to receive for original contentTopic
	s.waitForMsg(&WakuMsg{s.TestTopic, s.TestContentTopic, "test_msg"})

	// Message is possible to receive for new contentTopic
	s.waitForMsg(&WakuMsg{s.TestTopic, newContentTopic, "test_msg"})

	_ = s.unsubscribe(s.TestTopic, newContentTopic, s.FullNodeHost.ID())

	// Message should not be received for new contentTopic as it was unsubscribed
	s.waitForTimeout(&WakuMsg{s.TestTopic, newContentTopic, "test_msg"})

	// Message is still possible to receive for original contentTopic
	s.waitForMsg(&WakuMsg{s.TestTopic, s.TestContentTopic, "test_msg2"})

	_, err := s.LightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestUnsubscribeMultiContentTopic() {

	var messages = s.prepareData(3, false, true, true, nil)

	// Subscribe with 3 content topics
	for _, m := range messages {
		s.subscribe(m.PubSubTopic, m.ContentTopic, s.FullNodeHost.ID())
	}

	// All messages should be received
	s.waitForMessages(messages)

	// Unsubscribe with the last 2 content topics
	for _, m := range messages[1:] {
		_ = s.unsubscribe(m.PubSubTopic, m.ContentTopic, s.FullNodeHost.ID())
	}

	// Messages should not be received for the last two contentTopics as it was unsubscribed
	for _, m := range messages[1:] {
		s.waitForTimeout(&WakuMsg{m.PubSubTopic, m.ContentTopic, m.Payload})
	}

	// Message is still possible to receive for the first contentTopic
	s.waitForMsg(&WakuMsg{messages[0].PubSubTopic, messages[0].ContentTopic, messages[0].Payload})

	_, err := s.LightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestUnsubscribeMultiPubSubMultiContentTopic() {

	s.ctx, s.ctxCancel = context.WithTimeout(context.Background(), 20*time.Second)

	s.MakeWakuFilterLightNode()
	s.StartLightNode()

	s.MakeWakuFilterFullNode(s.TestTopic, true)

	// Connect nodes
	s.LightNodeHost.Peerstore().AddAddr(s.FullNodeHost.ID(), tests.GetHostAddress(s.FullNode.h), peerstore.PermanentAddrTTL)
	err := s.LightNodeHost.Peerstore().AddProtocols(s.FullNodeHost.ID(), FilterSubscribeID_v20beta1)
	s.Require().NoError(err)

	messages := s.prepareData(2, true, true, true, nil)

	// Subscribe
	for _, m := range messages {
		s.subDetails = append(s.subDetails, s.getSub(m.PubSubTopic, m.ContentTopic, s.FullNodeHost.ID())...)
		_, err = s.relayNode.Subscribe(context.Background(), protocol.NewContentFilter(m.PubSubTopic))
		s.Require().NoError(err)
	}

	// All messages should be received
	s.waitForMessages(messages)

	// Unsubscribe
	for _, m := range messages {
		_ = s.unsubscribe(m.PubSubTopic, m.ContentTopic, s.FullNodeHost.ID())
	}

	// No messages can be received with previous subscriptions
	for i, m := range messages {
		s.waitForTimeoutFromChan(&WakuMsg{m.PubSubTopic, m.ContentTopic, m.Payload}, s.subDetails[i].C)
	}
}

func (s *FilterTestSuite) TestUnsubscribeErrorHandling() {

	s.ctx, s.ctxCancel = context.WithTimeout(context.Background(), 20*time.Second)

	s.MakeWakuFilterLightNode()
	s.StartLightNode()

	s.MakeWakuFilterFullNode(s.TestTopic, true)

	// Connect nodes
	s.LightNodeHost.Peerstore().AddAddr(s.FullNodeHost.ID(), tests.GetHostAddress(s.FullNodeHost), peerstore.PermanentAddrTTL)
	err := s.LightNodeHost.Peerstore().AddProtocols(s.FullNodeHost.ID(), FilterSubscribeID_v20beta1)
	s.Require().NoError(err)

	var messages, invalidMessages []WakuMsg

	messages = s.prepareData(2, false, true, true, nil)

	// Prepare "invalid" data for unsubscribe
	invalidMessages = append(invalidMessages,
		WakuMsg{
			PubSubTopic:  "",
			ContentTopic: messages[0].ContentTopic,
			Payload:      "N/A",
		},
		WakuMsg{
			PubSubTopic:  messages[0].PubSubTopic,
			ContentTopic: "",
			Payload:      "N/A",
		},
		WakuMsg{
			PubSubTopic:  "/waku/2/go/filter/not_subscribed",
			ContentTopic: "not_subscribed_topic",
			Payload:      "N/A",
		})

	// Subscribe with valid topics
	for _, m := range messages {
		s.subscribe(m.PubSubTopic, m.ContentTopic, s.FullNodeHost.ID())
		_, err = s.relayNode.Subscribe(context.Background(), protocol.NewContentFilter(m.PubSubTopic))
		s.Require().NoError(err)
	}

	// All messages should be possible to receive for subscribed topics
	s.waitForMessages(messages)

	// Unsubscribe with empty pubsub
	contentFilter := protocol.ContentFilter{PubsubTopic: invalidMessages[0].PubSubTopic,
		ContentTopics: protocol.NewContentTopicSet(invalidMessages[0].ContentTopic)}
	_, err = s.LightNode.Unsubscribe(s.ctx, contentFilter, WithPeer(s.FullNodeHost.ID()))
	s.Require().Error(err)

	// Unsubscribe with empty content topic
	contentFilter = protocol.ContentFilter{PubsubTopic: invalidMessages[1].PubSubTopic,
		ContentTopics: protocol.NewContentTopicSet(invalidMessages[1].ContentTopic)}
	_, err = s.LightNode.Unsubscribe(s.ctx, contentFilter, WithPeer(s.FullNodeHost.ID()))
	s.Require().Error(err)

	// Unsubscribe with non-existent topics, expect no error to prevent attacker from topic guessing
	contentFilter = protocol.ContentFilter{PubsubTopic: invalidMessages[2].PubSubTopic,
		ContentTopics: protocol.NewContentTopicSet(invalidMessages[2].ContentTopic)}
	_, err = s.LightNode.Unsubscribe(s.ctx, contentFilter, WithPeer(s.FullNodeHost.ID()))
	s.Require().NoError(err)

	// All messages should be still possible to receive for subscribed topics
	s.waitForMessages(messages)

	_, err = s.LightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestUnsubscribeAllWithoutContentTopics() {

	var messages = s.prepareData(2, false, true, true, nil)

	// Subscribe with 2 content topics
	for _, m := range messages {
		s.subscribe(m.PubSubTopic, m.ContentTopic, s.FullNodeHost.ID())
	}

	// All messages should be received
	s.waitForMessages(messages)

	// Unsubscribe all with peer specification
	_, err := s.LightNode.UnsubscribeAll(s.ctx, WithPeer(s.FullNodeHost.ID()))
	s.Require().NoError(err)

	// Messages should not be received for any contentTopics
	for _, m := range messages {
		s.waitForTimeout(&WakuMsg{m.PubSubTopic, m.ContentTopic, m.Payload})
	}
}

func (s *FilterTestSuite) TestUnsubscribeAllDiffPubSubContentTopics() {

	s.ctx, s.ctxCancel = context.WithTimeout(context.Background(), 20*time.Second)

	s.MakeWakuFilterLightNode()
	s.StartLightNode()

	s.MakeWakuFilterFullNode(s.TestTopic, true)

	// Connect nodes
	s.LightNodeHost.Peerstore().AddAddr(s.FullNodeHost.ID(), tests.GetHostAddress(s.FullNode.h), peerstore.PermanentAddrTTL)
	err := s.LightNodeHost.Peerstore().AddProtocols(s.FullNodeHost.ID(), FilterSubscribeID_v20beta1)
	s.Require().NoError(err)

	messages := s.prepareData(2, true, true, true, nil)

	// Subscribe
	for _, m := range messages {
		s.subDetails = append(s.subDetails, s.getSub(m.PubSubTopic, m.ContentTopic, s.FullNodeHost.ID())...)
		_, err = s.relayNode.Subscribe(context.Background(), protocol.NewContentFilter(m.PubSubTopic))
		s.Require().NoError(err)
	}

	// All messages should be received
	s.waitForMessages(messages)

	// Unsubscribe all without any specification
	_, err = s.LightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

	// No messages can be received with previous subscriptions
	for i, m := range messages {
		s.waitForTimeoutFromChan(&WakuMsg{m.PubSubTopic, m.ContentTopic, m.Payload}, s.subDetails[i].C)
	}

}

func (s *FilterTestSuite) TestUnsubscribeAllUnrelatedPeer() {

	var messages = s.prepareData(2, false, true, false, nil)

	// Subscribe with 2 content topics
	for _, m := range messages {
		s.subscribe(m.PubSubTopic, m.ContentTopic, s.FullNodeHost.ID())
	}

	// All messages should be received
	s.waitForMessages(messages)

	// Create new host - not related to any node
	host, err := tests.MakeHost(context.Background(), 12345, rand.Reader)
	s.Require().NoError(err)

	s.Log.Info("Host ID", logging.HostID("FullNode", s.FullNodeHost.ID()))
	s.Log.Info("Host ID", logging.HostID("LightNode", s.LightNodeHost.ID()))
	s.Log.Info("Host ID", logging.HostID("Unrelated", host.ID()))

	// Unsubscribe all with unrelated peer specification
	pushResult, err := s.LightNode.UnsubscribeAll(s.ctx, WithPeer(host.ID()))

	for e := range pushResult.errs {
		s.Log.Info("Push Result ", zap.String("error", strconv.Itoa(e)))
	}

	// All messages should be received because peer ID used was not related to any subscription
	s.waitForMessages(messages)

	// Expect error for unsubscribe from non existing peer
	s.Require().Error(err)
}
