package waku

import (
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/pkg/messaging/waku/types"
)

func TestTrafficCounterReport(t *testing.T) {
	const relay, store = protocol.ID("/vac/waku/relay/2.0.0"), protocol.ID("/vac/waku/store-query/3.0.0")
	const alice, bob = peer.ID("alice"), peer.ID("bob")

	counter := newTrafficCounter()
	started := counter.since

	// libp2p reports every stream write twice: to the totals and to its protocol and peer.
	send := func(size int64, proto protocol.ID, p peer.ID) {
		counter.LogSentMessage(size)
		counter.LogSentMessageStream(size, proto, p)
	}
	receive := func(size int64, proto protocol.ID, p peer.ID) {
		counter.LogRecvMessage(size)
		counter.LogRecvMessageStream(size, proto, p)
	}
	send(100, store, alice)
	receive(5000, store, alice)
	send(700, relay, alice)
	receive(9000, relay, bob)

	// libp2p folds what it logged into its totals about once a second.
	var report types.TrafficReport
	require.Eventually(t, func() bool {
		report = counter.Report()
		return report.Totals.BytesReceived == 14000 && len(report.Peers) == 2
	}, 5*time.Second, 50*time.Millisecond)

	require.Equal(t, started, report.Since)
	require.False(t, report.Until.Before(report.Since))
	require.Equal(t, types.Traffic{BytesSent: 800, BytesReceived: 14000}, report.Totals)
	require.Equal(t, []types.ProtocolTraffic{
		{Protocol: relay, Traffic: types.Traffic{BytesSent: 700, BytesReceived: 9000}},
		{Protocol: store, Traffic: types.Traffic{BytesSent: 100, BytesReceived: 5000}},
	}, report.Protocols)
	require.Equal(t, []types.PeerTraffic{
		{Peer: bob, Traffic: types.Traffic{BytesReceived: 9000}},
		{Peer: alice, Traffic: types.Traffic{BytesSent: 800, BytesReceived: 5000}},
	}, report.Peers)
}

func TestTrafficCounterReset(t *testing.T) {
	counter := newTrafficCounter()
	started := counter.since
	counter.LogSentMessage(100)
	counter.LogSentMessageStream(100, "/ipfs/ping/1.0.0", "alice")
	require.Eventually(t, func() bool {
		return counter.Report().Totals.BytesSent == 100
	}, 5*time.Second, 50*time.Millisecond)

	counter.Reset()
	report := counter.Report()

	require.False(t, report.Since.Before(started))
	require.Zero(t, report.Totals)
	require.Empty(t, report.Protocols)
	require.Empty(t, report.Peers)
}
