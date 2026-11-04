package legacy_store

import (
	"context"
	"crypto/rand"
	"testing"

	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/peermanager"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"google.golang.org/protobuf/proto"
)

func TestQueryOptions(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pubSubTopic := "/waku/2/go/store/test"
	contentTopic := "/test/2/my-app/proto"

	// Init hosts with unique ports
	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)
	host, err := tests.MakeHost(ctx, port, rand.Reader)
	require.NoError(t, err)

	port2, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)
	host2, err := tests.MakeHost(ctx, port2, rand.Reader)
	require.NoError(t, err)

	// Let peer manager reside at host
	pm := peermanager.NewPeerManager(5, 5, nil, nil, true, utils.Logger())
	pm.SetHost(host)

	// Add host2 to peerstore
	host.Peerstore().AddAddr(host2.ID(), tests.GetHostAddress(host2), peerstore.PermanentAddrTTL)
	err = host.Peerstore().AddProtocols(host2.ID(), StoreID_v20beta4)
	require.NoError(t, err)

	// Create message and subscription
	msg := tests.CreateWakuMessage(contentTopic, utils.GetUnixEpoch())
	sub := SimulateSubscription([]*protocol.Envelope{
		protocol.NewEnvelope(msg, *utils.GetUnixEpoch(), pubSubTopic),
	})

	// Create store and save our msg into it
	s := NewWakuStore(MemoryDB(t), pm, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	s.SetHost(host)
	_ = s.storeMessage(protocol.NewEnvelope(msg, *utils.GetUnixEpoch(), pubSubTopic))

	err = s.Start(ctx, sub)
	require.NoError(t, err)
	defer s.Stop()

	// Create store2 and save our msg into it
	s2 := NewWakuStore(MemoryDB(t), pm, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	s2.SetHost(host2)
	_ = s2.storeMessage(protocol.NewEnvelope(msg, *utils.GetUnixEpoch(), pubSubTopic))

	sub2 := relay.NewSubscription(protocol.NewContentFilter(relay.DefaultWakuTopic))

	err = s2.Start(ctx, sub2)
	require.NoError(t, err)
	defer s2.Stop()

	q := Query{
		PubsubTopic:   pubSubTopic,
		ContentTopics: []string{contentTopic},
		StartTime:     nil,
		EndTime:       nil,
	}

	// Test no peers available
	_, err = s2.Query(ctx, q)
	require.Error(t, err)

	// Test peerId and peerAddr options are mutually exclusive
	_, err = s.Query(ctx, q, WithPeer(host2.ID()), WithPeerAddr(tests.GetHostAddress(host2)))
	require.Error(t, err)

	// Test peerAddr and peerId options are mutually exclusive
	_, err = s.Query(ctx, q, WithPeerAddr(tests.GetHostAddress(host2)), WithPeer(host2.ID()))
	require.Error(t, err)

	// Test WithRequestID
	result, err := s.Query(ctx, q, WithPeer(host2.ID()), WithRequestID([]byte("requestID")))
	require.NoError(t, err)
	require.True(t, proto.Equal(msg, result.Messages[0]))

	// Save cursor to use it in query with cursor option
	c := result.Cursor()

	// Test WithCursor
	result, err = s.Query(ctx, q, WithPeer(host2.ID()), WithCursor(c))
	require.NoError(t, err)
	require.True(t, proto.Equal(msg, result.Messages[0]))

	// Test WithFastestPeerSelection
	_, err = s.Query(ctx, q, WithFastestPeerSelection())
	require.NoError(t, err)
	require.True(t, proto.Equal(msg, result.Messages[0]))

	emptyPubSubTopicQuery := Query{
		PubsubTopic:   "",
		ContentTopics: []string{contentTopic},
		StartTime:     nil,
		EndTime:       nil,
	}

	// Test empty PubSubTopic provided in Query
	result, err = s.Query(ctx, emptyPubSubTopicQuery, WithPeer(host2.ID()))
	require.NoError(t, err)
	require.True(t, proto.Equal(msg, result.Messages[0]))

}
