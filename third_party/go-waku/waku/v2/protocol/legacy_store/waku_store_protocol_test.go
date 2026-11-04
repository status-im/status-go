package legacy_store

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/waku-org/go-waku/waku/persistence"
	"github.com/waku-org/go-waku/waku/persistence/sqlite"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"google.golang.org/protobuf/proto"
)

// SimulateSubscription creates a subscription for a list of envelopes
func SimulateSubscription(msgs []*protocol.Envelope) *relay.Subscription {
	ch := make(chan *protocol.Envelope, len(msgs))
	for _, msg := range msgs {
		ch <- msg
	}
	close(ch)
	return &relay.Subscription{
		Unsubscribe: func() {},
		Ch:          ch,
	}
}

func TestWakuStoreProtocolQuery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host1, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	s1 := NewWakuStore(MemoryDB(t), nil, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	s1.SetHost(host1)

	topic1 := "1"
	pubsubTopic1 := "topic1"

	msg := &pb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		ContentTopic: topic1,
		Timestamp:    utils.GetUnixEpoch(),
	}

	// Simulate a message has been received via relay protocol
	sub := SimulateSubscription([]*protocol.Envelope{protocol.NewEnvelope(msg, *utils.GetUnixEpoch(), pubsubTopic1)})
	err = s1.Start(ctx, sub)
	require.NoError(t, err)
	defer s1.Stop()

	s2 := NewWakuStore(MemoryDB(t), nil, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	host2, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)
	s2.SetHost(host2)
	sub1 := relay.NewSubscription(protocol.NewContentFilter(relay.DefaultWakuTopic))

	err = s2.Start(ctx, sub1)
	require.NoError(t, err)
	defer s2.Stop()

	host2.Peerstore().AddAddr(host1.ID(), tests.GetHostAddress(host1), peerstore.PermanentAddrTTL)
	err = host2.Peerstore().AddProtocols(host1.ID(), StoreID_v20beta4)
	require.NoError(t, err)

	q := Query{
		PubsubTopic:   pubsubTopic1,
		ContentTopics: []string{topic1},
	}

	response, err := s2.Query(ctx, q, WithPeer(host1.ID()))

	require.NoError(t, err)
	require.Len(t, response.Messages, 1)
	require.True(t, proto.Equal(msg, response.Messages[0]))
}

func TestWakuStoreProtocolLocalQuery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host1, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	s1 := NewWakuStore(MemoryDB(t), nil, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	s1.SetHost(host1)

	topic1 := "1"
	pubsubTopic1 := "topic1"

	msg := &pb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		ContentTopic: topic1,
		Timestamp:    utils.GetUnixEpoch(),
	}

	// Simulate a message has been received via relay protocol
	sub := SimulateSubscription([]*protocol.Envelope{protocol.NewEnvelope(msg, *utils.GetUnixEpoch(), pubsubTopic1)})
	err = s1.Start(ctx, sub)
	require.NoError(t, err)
	defer s1.Stop()

	time.Sleep(100 * time.Millisecond)

	q := Query{
		PubsubTopic:   pubsubTopic1,
		ContentTopics: []string{topic1},
	}
	response, err := s1.Query(ctx, q, WithLocalQuery())

	require.NoError(t, err)
	require.Len(t, response.Messages, 1)
	require.Equal(t, msg, response.Messages[0])
}

func TestWakuStoreProtocolNext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host1, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	db := MemoryDB(t)
	s1 := NewWakuStore(db, nil, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	s1.SetHost(host1)

	topic1 := "1"
	pubsubTopic1 := "topic1"

	now := *utils.GetUnixEpoch()
	msg1 := tests.CreateWakuMessage(topic1, proto.Int64(now+1))
	msg2 := tests.CreateWakuMessage(topic1, proto.Int64(now+2))
	msg3 := tests.CreateWakuMessage(topic1, proto.Int64(now+3))
	msg4 := tests.CreateWakuMessage(topic1, proto.Int64(now+4))
	msg5 := tests.CreateWakuMessage(topic1, proto.Int64(now+5))

	sub := SimulateSubscription([]*protocol.Envelope{
		protocol.NewEnvelope(msg1, *utils.GetUnixEpoch(), pubsubTopic1),
		protocol.NewEnvelope(msg2, *utils.GetUnixEpoch(), pubsubTopic1),
		protocol.NewEnvelope(msg3, *utils.GetUnixEpoch(), pubsubTopic1),
		protocol.NewEnvelope(msg4, *utils.GetUnixEpoch(), pubsubTopic1),
		protocol.NewEnvelope(msg5, *utils.GetUnixEpoch(), pubsubTopic1),
	})
	err = s1.Start(ctx, sub)
	require.NoError(t, err)

	host2, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	host2.Peerstore().AddAddr(host1.ID(), tests.GetHostAddress(host1), peerstore.PermanentAddrTTL)
	err = host2.Peerstore().AddProtocols(host1.ID(), StoreID_v20beta4)
	require.NoError(t, err)

	s2 := NewWakuStore(MemoryDB(t), nil, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	s2.SetHost(host2)
	sub1 := relay.NewSubscription(protocol.NewContentFilter(relay.DefaultWakuTopic))

	err = s2.Start(ctx, sub1)
	require.NoError(t, err)
	defer s2.Stop()

	q := Query{
		PubsubTopic:   pubsubTopic1,
		ContentTopics: []string{topic1},
	}

	response, err := s2.Query(ctx, q, WithPeer(host1.ID()), WithAutomaticRequestID(), WithPaging(true, 2))
	require.NoError(t, err)
	require.Len(t, response.Messages, 2)
	require.Equal(t, response.Messages[0].Timestamp, msg1.Timestamp)
	require.Equal(t, response.Messages[1].Timestamp, msg2.Timestamp)

	response, err = s2.Next(ctx, response)
	require.NoError(t, err)
	require.Len(t, response.Messages, 2)
	require.True(t, response.started)
	require.Equal(t, response.Messages[0].Timestamp, msg3.Timestamp)
	require.Equal(t, response.Messages[1].Timestamp, msg4.Timestamp)

	response, err = s2.Next(ctx, response)
	require.NoError(t, err)
	require.True(t, response.started)
	require.Len(t, response.Messages, 1)
	require.Equal(t, response.Messages[0].Timestamp, msg5.Timestamp)

	// No more records available
	response, err = s2.Next(ctx, response)
	require.NoError(t, err)
	require.True(t, response.started)
	require.Len(t, response.Messages, 0)
}

func TestWakuStoreResult(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host1, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	db := MemoryDB(t)
	s1 := NewWakuStore(db, nil, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	s1.SetHost(host1)

	topic1 := "1"
	pubsubTopic1 := "topic1"

	now := *utils.GetUnixEpoch()
	msg1 := tests.CreateWakuMessage(topic1, proto.Int64(now+1))
	msg2 := tests.CreateWakuMessage(topic1, proto.Int64(now+2))
	msg3 := tests.CreateWakuMessage(topic1, proto.Int64(now+3))
	msg4 := tests.CreateWakuMessage(topic1, proto.Int64(now+4))
	msg5 := tests.CreateWakuMessage(topic1, proto.Int64(now+5))

	sub := SimulateSubscription([]*protocol.Envelope{
		protocol.NewEnvelope(msg1, *utils.GetUnixEpoch(), pubsubTopic1),
		protocol.NewEnvelope(msg2, *utils.GetUnixEpoch(), pubsubTopic1),
		protocol.NewEnvelope(msg3, *utils.GetUnixEpoch(), pubsubTopic1),
		protocol.NewEnvelope(msg4, *utils.GetUnixEpoch(), pubsubTopic1),
		protocol.NewEnvelope(msg5, *utils.GetUnixEpoch(), pubsubTopic1),
	})
	err = s1.Start(ctx, sub)
	require.NoError(t, err)

	host2, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	host2.Peerstore().AddAddr(host1.ID(), tests.GetHostAddress(host1), peerstore.PermanentAddrTTL)
	err = host2.Peerstore().AddProtocols(host1.ID(), StoreID_v20beta4)
	require.NoError(t, err)

	s2 := NewWakuStore(MemoryDB(t), nil, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	s2.SetHost(host2)
	sub1 := relay.NewSubscription(protocol.NewContentFilter(relay.DefaultWakuTopic))

	err = s2.Start(ctx, sub1)
	require.NoError(t, err)
	defer s2.Stop()

	q := Query{
		PubsubTopic:   pubsubTopic1,
		ContentTopics: []string{topic1},
	}

	result, err := s2.Query(ctx, q, WithPeer(host1.ID()), WithAutomaticRequestID(), WithPaging(true, 2))
	require.NoError(t, err)
	require.False(t, result.started)
	require.Len(t, result.GetMessages(), 0)
	require.False(t, result.IsComplete())

	// First iteration
	hasNext, err := result.Next(ctx)
	require.NoError(t, err)
	require.True(t, result.started)
	require.True(t, hasNext)
	require.Len(t, result.GetMessages(), 2)
	require.Equal(t, result.GetMessages()[0].Timestamp, msg1.Timestamp)
	require.Equal(t, result.GetMessages()[1].Timestamp, msg2.Timestamp)
	require.False(t, result.IsComplete())

	// Second iteration
	hasNext, err = result.Next(ctx)
	require.NoError(t, err)
	require.True(t, result.started)
	require.True(t, hasNext)
	require.Len(t, result.GetMessages(), 2)
	require.Equal(t, result.GetMessages()[0].Timestamp, msg3.Timestamp)
	require.Equal(t, result.GetMessages()[1].Timestamp, msg4.Timestamp)
	require.False(t, result.IsComplete())

	// Third iteration
	hasNext, err = result.Next(ctx)
	require.NoError(t, err)
	require.True(t, result.started)
	require.True(t, hasNext)
	require.Len(t, result.GetMessages(), 1)
	require.Equal(t, result.GetMessages()[0].Timestamp, msg5.Timestamp)
	require.True(t, result.IsComplete())

	// No more records available
	hasNext, err = result.Next(ctx)
	require.NoError(t, err)
	require.True(t, result.started)
	require.False(t, hasNext)
}

func TestWakuStoreProtocolFind(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host1, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	s1 := NewWakuStore(MemoryDB(t), nil, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	s1.SetHost(host1)

	topic1 := "1"
	pubsubTopic1 := "topic1"

	now := *utils.GetUnixEpoch()
	msg1 := tests.CreateWakuMessage(topic1, proto.Int64(now+1))
	msg2 := tests.CreateWakuMessage(topic1, proto.Int64(now+2))
	msg3 := tests.CreateWakuMessage(topic1, proto.Int64(now+3))
	msg4 := tests.CreateWakuMessage(topic1, proto.Int64(now+4))
	msg5 := tests.CreateWakuMessage(topic1, proto.Int64(now+5))
	msg6 := tests.CreateWakuMessage(topic1, proto.Int64(now+6))
	msg7 := tests.CreateWakuMessage("hello", proto.Int64(now+7))
	msg8 := tests.CreateWakuMessage(topic1, proto.Int64(now+8))
	msg9 := tests.CreateWakuMessage(topic1, proto.Int64(now+9))

	sub := SimulateSubscription([]*protocol.Envelope{
		protocol.NewEnvelope(msg1, *utils.GetUnixEpoch(), pubsubTopic1),
		protocol.NewEnvelope(msg2, *utils.GetUnixEpoch(), pubsubTopic1),
		protocol.NewEnvelope(msg3, *utils.GetUnixEpoch(), pubsubTopic1),
		protocol.NewEnvelope(msg4, *utils.GetUnixEpoch(), pubsubTopic1),
		protocol.NewEnvelope(msg5, *utils.GetUnixEpoch(), pubsubTopic1),
		protocol.NewEnvelope(msg6, *utils.GetUnixEpoch(), pubsubTopic1),
		protocol.NewEnvelope(msg7, *utils.GetUnixEpoch(), pubsubTopic1),
		protocol.NewEnvelope(msg8, *utils.GetUnixEpoch(), pubsubTopic1),
		protocol.NewEnvelope(msg9, *utils.GetUnixEpoch(), pubsubTopic1),
	})
	err = s1.Start(ctx, sub)
	require.NoError(t, err)
	defer s1.Stop()

	host2, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	host2.Peerstore().AddAddr(host1.ID(), tests.GetHostAddress(host1), peerstore.PermanentAddrTTL)
	err = host2.Peerstore().AddProtocols(host1.ID(), StoreID_v20beta4)
	require.NoError(t, err)

	s2 := NewWakuStore(MemoryDB(t), nil, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	s2.SetHost(host2)

	sub1 := relay.NewSubscription(protocol.NewContentFilter(relay.DefaultWakuTopic))

	err = s2.Start(ctx, sub1)
	require.NoError(t, err)
	defer s2.Stop()

	q := Query{
		PubsubTopic: pubsubTopic1,
	}

	fn := func(msg *pb.WakuMessage) (bool, error) {
		return msg.ContentTopic == "hello", nil
	}

	foundMsg, err := s2.Find(ctx, q, fn, WithPeer(host1.ID()), WithAutomaticRequestID(), WithPaging(true, 2))
	require.NoError(t, err)
	require.NotNil(t, foundMsg)
	require.Equal(t, "hello", foundMsg.ContentTopic)

	fn2 := func(msg *pb.WakuMessage) (bool, error) {
		return msg.ContentTopic == "bye", nil
	}

	foundMsg, err = s2.Find(ctx, q, fn2, WithPeer(host1.ID()), WithAutomaticRequestID(), WithPaging(true, 2))
	require.NoError(t, err)
	require.Nil(t, foundMsg)
}

func TestWakuStoreStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	messageProvider := MemoryDB(t)

	s := NewWakuStore(messageProvider, nil, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	s.SetHost(host)

	pubSubTopic := "/waku/2/go/store/test"
	contentTopic := "/test/2/my-app"

	msg := &pb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		ContentTopic: contentTopic,
		Timestamp:    utils.GetUnixEpoch(),
	}

	// Simulate a message has been received via relay protocol
	sub := SimulateSubscription([]*protocol.Envelope{protocol.NewEnvelope(msg, *utils.GetUnixEpoch(), pubSubTopic)})

	// Store has nil message provider -> Start should return nil/no error
	s.msgProvider = nil
	err = s.Start(ctx, sub)
	require.NoError(t, err)
	s.Stop()

	// Start again already started store -> Start should return nil/no error
	s.msgProvider = messageProvider
	err = s.Start(ctx, sub)
	require.NoError(t, err)
	err = s.Start(ctx, sub)
	require.NoError(t, err)
	defer s.Stop()

	// Store Start cannot start its message provider -> return error
	var brokenDB *sql.DB
	brokenDB, err = sqlite.NewDB("sqlite:///no.db", utils.Logger())
	require.NoError(t, err)

	dbStore, err := persistence.NewDBStore(prometheus.DefaultRegisterer, utils.Logger(), persistence.WithDB(brokenDB),
		persistence.WithRetentionPolicy(10, 0))
	require.NoError(t, err)

	s2 := NewWakuStore(dbStore, nil, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	s2.SetHost(host)

	err = s2.Start(ctx, sub)
	require.Error(t, err)
	defer s2.Stop()

}

func TestWakuStoreWithStaticSharding(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host1, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	s1 := NewWakuStore(MemoryDB(t), nil, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	s1.SetHost(host1)

	// Prepare pubsub topics for static sharding
	pubSubTopics := protocol.ShardsToTopics(20, []int{1, 2, 3, 4})

	// Prepare test messages
	now := *utils.GetUnixEpoch()
	msg1 := tests.CreateWakuMessage("hello", proto.Int64(now))

	nowPlusOne := proto.Int64(now + 1)
	msg2 := tests.CreateWakuMessage("/test/2/my-app/sharded", nowPlusOne)

	nowPlusTwo := proto.Int64(now + 2)
	msg3 := tests.CreateWakuMessage("/test/2/my-app/sharded", nowPlusTwo)

	// Subscribe to pubSubtopics and start store1 + host1 with them
	sub := SimulateSubscription([]*protocol.Envelope{
		protocol.NewEnvelope(msg1, *utils.GetUnixEpoch(), pubSubTopics[0]),
		protocol.NewEnvelope(msg2, *utils.GetUnixEpoch(), pubSubTopics[1]),
		protocol.NewEnvelope(msg3, *utils.GetUnixEpoch(), pubSubTopics[2]),
	})
	err = s1.Start(ctx, sub)
	require.NoError(t, err)
	defer s1.Stop()

	host2, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	host2.Peerstore().AddAddr(host1.ID(), tests.GetHostAddress(host1), peerstore.PermanentAddrTTL)
	err = host2.Peerstore().AddProtocols(host1.ID(), StoreID_v20beta4)
	require.NoError(t, err)

	s2 := NewWakuStore(MemoryDB(t), nil, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	s2.SetHost(host2)

	// Subscribe to different pubSubTopics[3] at store2 + host2
	sub1 := relay.NewSubscription(protocol.NewContentFilter(pubSubTopics[3]))

	err = s2.Start(ctx, sub1)
	require.NoError(t, err)
	defer s2.Stop()

	q1 := Query{
		PubsubTopic: pubSubTopics[0],
	}

	fn1 := func(msg *pb.WakuMessage) (bool, error) {
		return msg.ContentTopic == "hello", nil
	}

	// Find msg1 on the second host2+s2
	foundMsg, err := s2.Find(ctx, q1, fn1, WithPeer(host1.ID()), WithAutomaticRequestID(), WithPaging(true, 2))
	require.NoError(t, err)
	require.NotNil(t, foundMsg)
	require.Equal(t, "hello", foundMsg.ContentTopic)

	q2 := Query{
		PubsubTopic: pubSubTopics[1],
	}

	// Find msg2 on the second host2+s2; No other messages (msg3) should be found
	result, err := s2.Query(ctx, q2, WithPeer(host1.ID()), WithAutomaticRequestID(), WithPaging(true, 2))
	require.NoError(t, err)

	for i, m := range result.Messages {
		if i == 0 {
			require.Equal(t, "/test/2/my-app/sharded", m.ContentTopic)
			require.Equal(t, nowPlusOne, m.Timestamp)
		} else {
			require.Fail(t, "Unexpected message found")
		}
	}

}
