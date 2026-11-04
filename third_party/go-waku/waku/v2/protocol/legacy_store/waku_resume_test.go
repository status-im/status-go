package legacy_store

import (
	"context"
	"testing"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peer"
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

func TestFindLastSeenMessage(t *testing.T) {
	now := *utils.GetUnixEpoch()
	msg1 := protocol.NewEnvelope(tests.CreateWakuMessage("1", proto.Int64(now+1)), *utils.GetUnixEpoch(), "test")
	msg2 := protocol.NewEnvelope(tests.CreateWakuMessage("2", proto.Int64(now+2)), *utils.GetUnixEpoch(), "test")
	msg3 := protocol.NewEnvelope(tests.CreateWakuMessage("3", proto.Int64(now+3)), *utils.GetUnixEpoch(), "test")
	msg4 := protocol.NewEnvelope(tests.CreateWakuMessage("4", proto.Int64(now+4)), *utils.GetUnixEpoch(), "test")
	msg5 := protocol.NewEnvelope(tests.CreateWakuMessage("5", proto.Int64(now+5)), *utils.GetUnixEpoch(), "test")

	s := NewWakuStore(MemoryDB(t), nil, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	_ = s.storeMessage(msg1)
	_ = s.storeMessage(msg3)
	_ = s.storeMessage(msg5)
	_ = s.storeMessage(msg2)
	_ = s.storeMessage(msg4)

	lastSeen, err := s.findLastSeen()
	require.NoError(t, err)

	require.Equal(t, msg5.Message().GetTimestamp(), lastSeen)
}

func TestResume(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host1, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	s1 := NewWakuStore(MemoryDB(t), nil, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	s1.SetHost(host1)
	sub := relay.NewSubscription(protocol.NewContentFilter(relay.DefaultWakuTopic))
	err = s1.Start(ctx, sub)
	require.NoError(t, err)

	defer s1.Stop()

	now := *utils.GetUnixEpoch()
	for i := int64(0); i < 10; i++ {
		var contentTopic = "1"
		if i%2 == 0 {
			contentTopic = "2"
		}

		wakuMessage := tests.CreateWakuMessage(contentTopic, proto.Int64(now+i+1))
		msg := protocol.NewEnvelope(wakuMessage, *utils.GetUnixEpoch(), "test")
		_ = s1.storeMessage(msg)
	}

	host2, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	s2 := NewWakuStore(MemoryDB(t), nil, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	s2.SetHost(host2)
	sub1 := relay.NewSubscription(protocol.NewContentFilter(relay.DefaultWakuTopic))

	err = s2.Start(ctx, sub1)
	require.NoError(t, err)
	defer s2.Stop()

	host2.Peerstore().AddAddr(host1.ID(), tests.GetHostAddress(host1), peerstore.PermanentAddrTTL)
	err = host2.Peerstore().AddProtocols(host1.ID(), StoreID_v20beta4)
	require.NoError(t, err)

	msgCount, err := s2.Resume(ctx, "test", []peer.ID{host1.ID()})

	require.NoError(t, err)
	require.Equal(t, 10, msgCount)

	allMsgs, err := s2.msgProvider.GetAll()
	require.NoError(t, err)

	require.Len(t, allMsgs, 10)

	// Test duplication
	msgCount, err = s2.Resume(ctx, "test", []peer.ID{host1.ID()})

	require.NoError(t, err)
	require.Equal(t, 0, msgCount)
}

func TestResumeWithListOfPeers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Host that does not support store protocol
	invalidHost, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	host1, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	s1 := NewWakuStore(MemoryDB(t), nil, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	s1.SetHost(host1)
	sub := relay.NewSubscription(protocol.NewContentFilter(relay.DefaultWakuTopic))

	err = s1.Start(ctx, sub)
	require.NoError(t, err)

	defer s1.Stop()

	msg0 := &pb.WakuMessage{Payload: []byte{1, 2, 3}, ContentTopic: "2", Timestamp: utils.GetUnixEpoch()}

	_ = s1.storeMessage(protocol.NewEnvelope(msg0, *utils.GetUnixEpoch(), "test"))

	host2, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	s2 := NewWakuStore(MemoryDB(t), nil, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	s2.SetHost(host2)
	sub1 := relay.NewSubscription(protocol.NewContentFilter(relay.DefaultWakuTopic))

	err = s2.Start(ctx, sub1)
	require.NoError(t, err)
	defer s2.Stop()

	host2.Peerstore().AddAddr(host1.ID(), tests.GetHostAddress(host1), peerstore.PermanentAddrTTL)
	err = host2.Peerstore().AddProtocols(host1.ID(), StoreID_v20beta4)
	require.NoError(t, err)

	msgCount, err := s2.Resume(ctx, "test", []peer.ID{invalidHost.ID(), host1.ID()})

	require.NoError(t, err)
	require.Equal(t, 1, msgCount)

	allMsgs, err := s2.msgProvider.GetAll()
	require.NoError(t, err)
	require.Len(t, allMsgs, 1)
}

func TestResumeWithoutSpecifyingPeer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host1, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	s1 := NewWakuStore(MemoryDB(t), nil, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	s1.SetHost(host1)
	sub := relay.NewSubscription(protocol.NewContentFilter(relay.DefaultWakuTopic))

	err = s1.Start(ctx, sub)
	require.NoError(t, err)

	defer s1.Stop()

	msg0 := &pb.WakuMessage{Payload: []byte{1, 2, 3}, ContentTopic: "2"}

	_ = s1.storeMessage(protocol.NewEnvelope(msg0, *utils.GetUnixEpoch(), "test"))

	host2, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	s2 := NewWakuStore(MemoryDB(t), nil, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	s2.SetHost(host2)
	sub1 := relay.NewSubscription(protocol.NewContentFilter(relay.DefaultWakuTopic))

	err = s2.Start(ctx, sub1)
	require.NoError(t, err)

	defer s2.Stop()

	host2.Peerstore().AddAddr(host1.ID(), tests.GetHostAddress(host1), peerstore.PermanentAddrTTL)
	err = host2.Peerstore().AddProtocols(host1.ID(), StoreID_v20beta4)
	require.NoError(t, err)

	_, err = s2.Resume(ctx, "test", []peer.ID{})
	require.Error(t, err)
}
