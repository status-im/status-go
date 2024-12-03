package filter

import (
	"context"
	"crypto/rand"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/suite"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/onlinechecker"
	"github.com/waku-org/go-waku/waku/v2/peermanager"
	wps "github.com/waku-org/go-waku/waku/v2/peerstore"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/protocol/subscription"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type LightNodeData struct {
	LightNode     *WakuFilterLightNode
	LightNodeHost host.Host
}

type FullNodeData struct {
	relayNode    *relay.WakuRelay
	RelaySub     *relay.Subscription
	FullNodeHost host.Host
	Broadcaster  relay.Broadcaster
	FullNode     *WakuFilterFullNode
}

type FilterTestSuite struct {
	suite.Suite
	LightNodeData
	FullNodeData

	TestTopic        string
	TestContentTopic string
	ctx              context.Context
	ctxCancel        context.CancelFunc
	wg               *sync.WaitGroup
	ContentFilter    protocol.ContentFilter
	subDetails       []*subscription.SubscriptionDetails

	Log *zap.Logger
}

const DefaultTestPubSubTopic = "/waku/2/go/filter/test"
const DefaultTestContentTopic = "/test/10/my-app"

type WakuMsg struct {
	PubSubTopic  string
	ContentTopic string
	Payload      string
}

func (s *FilterTestSuite) SetupTest() {
	log := utils.Logger()
	s.Log = log

	s.Log.Info("SetupTest()")
	// Use a pointer to WaitGroup so that to avoid copying
	// https://pkg.go.dev/sync#WaitGroup
	s.wg = &sync.WaitGroup{}

	// Create test context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // Test can't exceed 10 seconds
	s.ctx = ctx
	s.ctxCancel = cancel

	s.TestTopic = DefaultTestPubSubTopic
	s.TestContentTopic = DefaultTestContentTopic

	s.MakeWakuFilterLightNode()
	s.LightNode.peerPingInterval = 1 * time.Second
	s.StartLightNode()

	//TODO: Add tests to verify broadcaster.

	s.MakeWakuFilterFullNode(s.TestTopic, false)

	s.ConnectToFullNode(s.LightNode, s.FullNode)

}

func (s *FilterTestSuite) TearDownTest() {
	s.FullNode.Stop()
	s.LightNode.Stop()
	s.RelaySub.Unsubscribe()
	s.LightNode.Stop()
	s.ctxCancel()
}

func (s *FilterTestSuite) ConnectToFullNode(h1 *WakuFilterLightNode, h2 *WakuFilterFullNode) {
	mAddr := tests.GetAddr(h2.h)
	_, err := h1.pm.AddPeer(mAddr, wps.Static, []string{s.TestTopic}, FilterSubscribeID_v20beta1)
	s.Log.Info("add peer", zap.Stringer("mAddr", mAddr))
	s.Require().NoError(err)
}

func (s *FilterTestSuite) GetWakuRelay(topic string) FullNodeData {

	broadcaster := relay.NewBroadcaster(10)
	s.Require().NoError(broadcaster.Start(context.Background()))

	port, err := tests.FindFreePort(s.T(), "", 5)
	s.Require().NoError(err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	s.Require().NoError(err)

	relay := relay.NewWakuRelay(broadcaster, 0, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, s.Log, relay.WithMaxMsgSize(1024*1024))
	relay.SetHost(host)

	err = relay.Start(context.Background())
	s.Require().NoError(err)

	sub, err := relay.Subscribe(context.Background(), protocol.NewContentFilter(topic))
	s.Require().NoError(err)

	return FullNodeData{relay, sub[0], host, broadcaster, nil}
}

func (s *FilterTestSuite) GetWakuFilterFullNode(topic string, withRegisterAll bool) FullNodeData {

	nodeData := s.GetWakuRelay(topic)

	node2Filter := NewWakuFilterFullNode(timesource.NewDefaultClock(), prometheus.DefaultRegisterer, s.Log, WithFullNodeRateLimiter(rate.Inf, 0))
	node2Filter.SetHost(nodeData.FullNodeHost)

	var sub *relay.Subscription
	if withRegisterAll {
		sub = nodeData.Broadcaster.RegisterForAll()
	} else {
		sub = nodeData.Broadcaster.Register(protocol.NewContentFilter(topic))
	}

	err := node2Filter.Start(s.ctx, sub)
	s.Require().NoError(err)

	nodeData.FullNode = node2Filter

	return nodeData
}

func (s *FilterTestSuite) MakeWakuFilterFullNode(topic string, withRegisterAll bool) {
	nodeData := s.GetWakuFilterFullNode(topic, withRegisterAll)

	s.FullNodeData = nodeData
}

func (s *FilterTestSuite) GetWakuFilterLightNode() LightNodeData {
	port, err := tests.FindFreePort(s.T(), "", 5)
	s.Require().NoError(err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	s.Require().NoError(err)
	b := relay.NewBroadcaster(10)
	s.Require().NoError(b.Start(context.Background()))
	pm := peermanager.NewPeerManager(5, 5, nil, nil, true, s.Log)
	filterPush := NewWakuFilterLightNode(b, pm, timesource.NewDefaultClock(), onlinechecker.NewDefaultOnlineChecker(true), prometheus.DefaultRegisterer, s.Log, WithLightNodeRateLimiter(rate.Inf, 0))
	filterPush.SetHost(host)
	pm.SetHost(host)
	return LightNodeData{filterPush, host}
}

func (s *FilterTestSuite) MakeWakuFilterLightNode() {
	s.LightNodeData = s.GetWakuFilterLightNode()
}

func (s *FilterTestSuite) StartLightNode() {
	err := s.LightNode.Start(context.Background())
	s.Require().NoError(err)
}

func (s *FilterTestSuite) waitForMsg(msg *WakuMsg) {
	s.waitForMsgFromChan(msg, s.subDetails[0].C)
}

func (s *FilterTestSuite) waitForMsgFromChan(msg *WakuMsg, ch chan *protocol.Envelope) {
	s.wg.Add(1)
	var msgFound = false
	go func() {
		defer s.wg.Done()
		select {
		case env := <-ch:
			for _, topic := range s.ContentFilter.ContentTopicsList() {
				if topic == env.Message().GetContentTopic() {
					msgFound = true
				}
			}
			s.Require().True(msgFound)
		case <-time.After(1 * time.Second):
			s.Require().Fail("Message timeout")
		case <-s.ctx.Done():
			s.Require().Fail("test exceeded allocated time")
		}
	}()

	if msg != nil {
		s.PublishMsg(msg)
	}

	s.wg.Wait()
}

func matchOneOfManyMsg(one WakuMsg, many []WakuMsg) bool {
	for _, m := range many {
		if m.PubSubTopic == one.PubSubTopic &&
			m.ContentTopic == one.ContentTopic &&
			m.Payload == one.Payload {
			return true
		}
	}

	return false
}

func (s *FilterTestSuite) waitForMessages(msgs []WakuMsg) {
	s.wg.Add(1)
	msgCount := len(msgs)
	found := 0
	subs := s.subDetails
	s.Log.Info("Expected messages ", zap.String("count", strconv.Itoa(msgCount)))
	s.Log.Info("Existing subscriptions ", zap.String("count", strconv.Itoa(len(subs))))

	go func() {
		defer s.wg.Done()
		for _, sub := range subs {
			s.Log.Info("Looking at ", zap.String("pubSubTopic", sub.ContentFilter.PubsubTopic))
			for i := 0; i < msgCount; i++ {
				select {
				case env, ok := <-sub.C:
					if !ok {
						continue
					}
					received := WakuMsg{
						PubSubTopic:  env.PubsubTopic(),
						ContentTopic: env.Message().GetContentTopic(),
						Payload:      string(env.Message().GetPayload()),
					}
					s.Log.Debug("received message ", zap.String("pubSubTopic", received.PubSubTopic), zap.String("contentTopic", received.ContentTopic), zap.String("payload", received.Payload))
					if matchOneOfManyMsg(received, msgs) {
						found++
					}
				case <-time.After(3 * time.Second):

				case <-s.ctx.Done():
					s.Require().Fail("test exceeded allocated time")
				}
			}
		}
	}()

	if msgs != nil {
		s.publishMessages(msgs)
	}

	s.wg.Wait()
	s.Require().Equal(msgCount, found)
}

func (s *FilterTestSuite) waitForTimeout(msg *WakuMsg) {
	s.waitForTimeoutFromChan(msg, s.subDetails[0].C)
}

func (s *FilterTestSuite) waitForTimeoutFromChan(msg *WakuMsg, ch chan *protocol.Envelope) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		select {
		case env, ok := <-ch:
			if ok {
				s.Require().Fail("should not receive another message", zap.String("payload", string(env.Message().Payload)))
			}
		case <-time.After(1 * time.Second):
			// Timeout elapsed, all good
		case <-s.ctx.Done():
			s.Require().Fail("waitForTimeout test exceeded allocated time")
		}
	}()

	s.PublishMsg(msg)

	s.wg.Wait()
}

func (s *FilterTestSuite) getSub(pubsubTopic string, contentTopic string, peer peer.ID) []*subscription.SubscriptionDetails {
	contentFilter := protocol.ContentFilter{PubsubTopic: pubsubTopic, ContentTopics: protocol.NewContentTopicSet(contentTopic)}

	subDetails, err := s.LightNode.Subscribe(s.ctx, contentFilter, WithPeer(peer))
	s.Require().NoError(err)

	time.Sleep(1 * time.Second)

	return subDetails
}
func (s *FilterTestSuite) subscribe(pubsubTopic string, contentTopic string, peer peer.ID) {

	for _, sub := range s.subDetails {
		if sub.ContentFilter.PubsubTopic == pubsubTopic {
			sub.Add(contentTopic)
			s.ContentFilter = sub.ContentFilter
			subDetails, err := s.LightNode.Subscribe(s.ctx, s.ContentFilter, WithPeer(peer))
			s.subDetails = subDetails
			s.Require().NoError(err)
			return
		}
	}

	s.subDetails = s.getSub(pubsubTopic, contentTopic, peer)
	s.ContentFilter = s.subDetails[0].ContentFilter
}

func (s *FilterTestSuite) unsubscribe(pubsubTopic string, contentTopic string, peer peer.ID) []*subscription.SubscriptionDetails {

	for _, sub := range s.subDetails {
		if sub.ContentFilter.PubsubTopic == pubsubTopic {
			topicsCount := len(sub.ContentFilter.ContentTopicsList())
			if topicsCount == 1 {
				_, err := s.LightNode.Unsubscribe(s.ctx, sub.ContentFilter, WithPeer(peer))
				s.Require().NoError(err)
			} else {
				sub.Remove(contentTopic)
			}
			s.ContentFilter = sub.ContentFilter
		}
	}

	return s.LightNode.Subscriptions()
}

func (s *FilterTestSuite) PublishMsg(msg *WakuMsg) {
	if len(msg.Payload) == 0 {
		msg.Payload = "123"
	}

	_, err := s.relayNode.Publish(s.ctx, tests.CreateWakuMessage(msg.ContentTopic, utils.GetUnixEpoch(), msg.Payload), relay.WithPubSubTopic(msg.PubSubTopic))
	s.Require().NoError(err)
}

func (s *FilterTestSuite) publishMessages(msgs []WakuMsg) {
	for _, m := range msgs {
		_, err := s.relayNode.Publish(s.ctx, tests.CreateWakuMessage(m.ContentTopic, utils.GetUnixEpoch(), m.Payload), relay.WithPubSubTopic(m.PubSubTopic))
		s.Require().NoError(err)
	}
}

func (s *FilterTestSuite) prepareData(quantity int, topics, contentTopics, payloads bool, sg tests.StringGenerator) []WakuMsg {
	var (
		pubsubTopic     = s.TestTopic        // Has to be the same with initial s.testTopic
		contentTopic    = s.TestContentTopic // Has to be the same with initial s.testContentTopic
		payload         = "test_msg"
		messages        []WakuMsg
		strMaxLenght    = 4097
		generatedString = ""
	)

	for i := 0; i < quantity; i++ {
		msg := WakuMsg{
			PubSubTopic:  pubsubTopic,
			ContentTopic: contentTopic,
			Payload:      payload,
		}

		if sg != nil {
			generatedString, _ = sg(strMaxLenght)

		}

		if topics {
			msg.PubSubTopic = fmt.Sprintf("%s%02d%s", pubsubTopic, i, generatedString)
		}

		if contentTopics {
			msg.ContentTopic = fmt.Sprintf("%s%02d%s", contentTopic, i, generatedString)
		}

		if payloads {
			msg.Payload = fmt.Sprintf("%s%02d%s", payload, i, generatedString)
		}

		messages = append(messages, msg)
	}

	return messages
}
