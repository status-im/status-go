package peermanager

import (
	"context"
	"github.com/libp2p/go-libp2p/core/control"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peerstore"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"testing"
)

type mockConnMultiaddrs struct {
	local, remote ma.Multiaddr
}

func (m mockConnMultiaddrs) LocalMultiaddr() ma.Multiaddr {
	return m.local
}

func (m mockConnMultiaddrs) RemoteMultiaddr() ma.Multiaddr {
	return m.remote
}

func TestConnectionGater(t *testing.T) {

	log := utils.Logger()

	_, h1, _ := makeWakuRelay(t, log)
	_, h2, _ := makeWakuRelay(t, log)

	h1.Peerstore().AddAddrs(h2.ID(), h2.Addrs(), peerstore.PermanentAddrTTL)
	err := h1.Connect(context.Background(), h2.Peerstore().PeerInfo(h2.ID()))
	require.NoError(t, err)

	h2.Peerstore().AddAddrs(h1.ID(), h1.Addrs(), peerstore.PermanentAddrTTL)
	err = h2.Connect(context.Background(), h1.Peerstore().PeerInfo(h1.ID()))
	require.NoError(t, err)

	peerA := h1.ID()

	remoteAddr1 := ma.StringCast("/ip4/1.2.3.4/tcp/1234")

	connGater := NewConnectionGater(2, log)

	// Test peer blocking
	allow := connGater.InterceptPeerDial(peerA)
	require.True(t, allow)

	// Test connection was secured and upgraded
	allow = connGater.InterceptSecured(network.DirInbound, peerA, &mockConnMultiaddrs{local: nil, remote: nil})
	require.True(t, allow)

	connection1 := h1.Network().Conns()[0]

	allow, reason := connGater.InterceptUpgraded(connection1)
	require.True(t, allow)
	require.Equal(t, control.DisconnectReason(0), reason)

	// Test addr and subnet blocking
	allow = connGater.InterceptAddrDial(peerA, remoteAddr1)
	require.True(t, allow)

	// Bellow the connection limit
	allow = connGater.InterceptAccept(&mockConnMultiaddrs{local: nil, remote: remoteAddr1})
	require.True(t, allow)

	ip, err := manet.ToIP(remoteAddr1)
	require.NoError(t, err)
	connGater.limiter[ip.String()] = 3

	// Above the connection limit
	allow = connGater.InterceptAccept(&mockConnMultiaddrs{local: nil, remote: remoteAddr1})
	require.False(t, allow)

	// Call twice NotifyDisconnect to get bellow the limit(2): 3 -> 1
	connGater.NotifyDisconnect(remoteAddr1)
	connGater.NotifyDisconnect(remoteAddr1)

	// Bellow the connection limit again
	allow = connGater.validateInboundConn(remoteAddr1)
	require.True(t, allow)

}
