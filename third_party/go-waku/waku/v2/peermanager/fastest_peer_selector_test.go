package peermanager

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func TestRTT(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	h1, _ := tests.MakeHost(ctx, 0, rand.Reader)
	h2, _ := tests.MakeHost(ctx, 0, rand.Reader)
	h3, _ := tests.MakeHost(ctx, 0, rand.Reader)

	h1.Peerstore().AddAddrs(h2.ID(), h2.Addrs(), peerstore.PermanentAddrTTL)
	h1.Peerstore().AddAddrs(h3.ID(), h3.Addrs(), peerstore.PermanentAddrTTL)

	rtt := NewFastestPeerSelector(utils.Logger())
	rtt.SetHost(h1)

	_, err := rtt.FastestPeer(ctx, peer.IDSlice{h2.ID(), h3.ID()})
	require.NoError(t, err)

	// Simulate H3 being no longer available
	h3.Close()

	_, err = rtt.FastestPeer(ctx, peer.IDSlice{h3.ID()})
	require.ErrorIs(t, err, utils.ErrNoPeersAvailable)

	// H3 should never return
	for i := 0; i < 100; i++ {
		p, err := rtt.FastestPeer(ctx, peer.IDSlice{h2.ID(), h3.ID()})
		if err != nil {
			require.ErrorIs(t, err, utils.ErrNoPeersAvailable)
		} else {
			require.NotEqual(t, h3.ID(), p)
		}
	}
}
