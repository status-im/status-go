package publish

import (
	"context"
	"crypto/rand"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

type MockMessageSentCheck struct {
	Messages map[string]map[common.Hash]uint32
}

func (m *MockMessageSentCheck) Add(topic string, messageID common.Hash, time uint32) {
	if m.Messages[topic] == nil {
		m.Messages[topic] = make(map[common.Hash]uint32)
	}
	m.Messages[topic][messageID] = time
}

func (m *MockMessageSentCheck) DeleteByMessageIDs(messageIDs []common.Hash) {
}

func (m *MockMessageSentCheck) SetStorePeerID(peerID peer.ID) {
}

func (m *MockMessageSentCheck) Start() {
}

func TestNewSenderWithUnknownMethod(t *testing.T) {
	sender, err := NewMessageSender(UnknownMethod, nil, nil, nil)
	require.NotNil(t, err)
	require.Nil(t, sender)
}

func TestNewSenderWithRelay(t *testing.T) {
	_, relayNode := createRelayNode(t)
	err := relayNode.Start(context.Background())
	require.Nil(t, err)
	defer relayNode.Stop()

	_, err = relayNode.Subscribe(context.Background(), protocol.NewContentFilter("test-pubsub-topic"))
	require.Nil(t, err)
	publisher := NewDefaultPublisher(nil, relayNode)
	sender, err := NewMessageSender(Relay, publisher, nil, utils.Logger())
	require.Nil(t, err)
	require.NotNil(t, sender)
	require.Nil(t, sender.messageSentCheck)
	require.Equal(t, Relay, sender.publishMethod)

	msg := &pb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		Timestamp:    utils.GetUnixEpoch(),
		ContentTopic: "test-content-topic",
	}
	envelope := protocol.NewEnvelope(msg, *utils.GetUnixEpoch(), "test-pubsub-topic")
	req := NewRequest(context.TODO(), envelope)
	err = sender.Send(req)
	require.Nil(t, err)
}

func TestNewSenderWithRelayAndMessageSentCheck(t *testing.T) {
	_, relayNode := createRelayNode(t)
	err := relayNode.Start(context.Background())
	require.Nil(t, err)
	defer relayNode.Stop()

	_, err = relayNode.Subscribe(context.Background(), protocol.NewContentFilter("test-pubsub-topic"))
	require.Nil(t, err)
	publisher := NewDefaultPublisher(nil, relayNode)
	sender, err := NewMessageSender(Relay, publisher, nil, utils.Logger())

	check := &MockMessageSentCheck{Messages: make(map[string]map[common.Hash]uint32)}
	sender.WithMessageSentCheck(check)
	require.Nil(t, err)
	require.NotNil(t, sender)
	require.NotNil(t, sender.messageSentCheck)
	require.Equal(t, Relay, sender.publishMethod)

	msg := &pb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		Timestamp:    utils.GetUnixEpoch(),
		ContentTopic: "test-content-topic",
	}
	envelope := protocol.NewEnvelope(msg, *utils.GetUnixEpoch(), "test-pubsub-topic")
	req := NewRequest(context.TODO(), envelope)

	require.Equal(t, 0, len(check.Messages))

	err = sender.Send(req)
	require.Nil(t, err)
	require.Equal(t, 1, len(check.Messages))
	require.Equal(
		t,
		uint32(msg.GetTimestamp()/int64(time.Second)),
		check.Messages["test-pubsub-topic"][common.BytesToHash(envelope.Hash().Bytes())],
	)
}

func TestNewSenderWithLightPush(t *testing.T) {
	sender, err := NewMessageSender(LightPush, nil, nil, nil)
	require.Nil(t, err)
	require.NotNil(t, sender)
	require.Equal(t, LightPush, sender.publishMethod)
}

func createRelayNode(t *testing.T) (host.Host, *relay.WakuRelay) {
	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)
	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)
	bcaster := relay.NewBroadcaster(10)
	relay := relay.NewWakuRelay(bcaster, 0, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	relay.SetHost(host)
	err = bcaster.Start(context.Background())
	require.NoError(t, err)

	return host, relay
}

func TestMessageSentEmitter(t *testing.T) {
	host, relayNode := createRelayNode(t)
	err := relayNode.Start(context.Background())
	require.Nil(t, err)
	defer relayNode.Stop()

	_, err = relayNode.Subscribe(context.Background(), protocol.NewContentFilter("test-pubsub-topic"))
	require.Nil(t, err)
	publisher := NewDefaultPublisher(nil, relayNode)
	sender, err := NewMessageSender(Relay, publisher, nil, utils.Logger())
	require.Nil(t, err)

	check := &MockMessageSentCheck{Messages: make(map[string]map[common.Hash]uint32)}
	sender.WithMessageSentCheck(check)
	sender.WithMessageSentEmitter(host)

	msg := &pb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		Timestamp:    utils.GetUnixEpoch(),
		ContentTopic: "test-content-topic",
	}
	envelope := protocol.NewEnvelope(msg, *utils.GetUnixEpoch(), "test-pubsub-topic")
	req := NewRequest(context.TODO(), envelope)

	wg := sync.WaitGroup{}
	wg.Add(1)
	sub, err := host.EventBus().Subscribe(new(MessageSent))
	require.Nil(t, err)
	defer sub.Close()

	go func() {
		for msgSentEvt := range sub.Out() {
			msgSent := msgSentEvt.(MessageSent)
			require.Equal(t, uint32(len(msg.Payload)), msgSent.Size)
			wg.Done()
		}
	}()

	err = sender.Send(req)
	require.Nil(t, err)
	go wg.Wait()
}
