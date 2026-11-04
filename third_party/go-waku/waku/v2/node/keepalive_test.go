package node

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	wps "github.com/waku-org/go-waku/waku/v2/peerstore"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func TestKeepAlive(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host1, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)
	host2, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)
	peerID2 := host2.ID()

	host1.Peerstore().AddAddrs(peerID2, host2.Addrs(), peerstore.PermanentAddrTTL)

	err = host1.Connect(ctx, host1.Peerstore().PeerInfo(peerID2))
	require.NoError(t, err)

	require.Len(t, host1.Network().Peers(), 1)

	ctx2, cancel2 := context.WithTimeout(ctx, 3*time.Second)
	defer cancel2()
	wg := &sync.WaitGroup{}

	w := &WakuNode{
		host: host1,
		wg:   wg,
		log:  utils.Logger(),
	}

	w.wg.Add(1)

	peerFailureSignalChan := make(chan bool, 1)
	w.pingPeer(ctx2, w.wg, peerID2, peerFailureSignalChan)
	require.NoError(t, ctx.Err())
	close(peerFailureSignalChan)
}

func TestPeriodicKeepAlive(t *testing.T) {
	hostAddr, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:0")

	host1, err := libp2p.New(libp2p.DefaultTransports, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)

	key, err := tests.RandomHex(32)
	require.NoError(t, err)
	prvKey, err := crypto.HexToECDSA(key)
	require.NoError(t, err)

	ctx := context.Background()

	wakuNode, err := New(
		WithPrivateKey(prvKey),
		WithHostAddress(hostAddr),
		WithWakuRelay(),
		WithKeepAlive(time.Minute, time.Second),
	)

	require.NoError(t, err)

	err = wakuNode.Start(ctx)
	require.NoError(t, err)

	node2MAddr, err := multiaddr.NewMultiaddr(host1.Addrs()[0].String() + "/p2p/" + host1.ID().String())
	require.NoError(t, err)
	_, err = wakuNode.AddPeer([]multiaddr.Multiaddr{node2MAddr}, wps.Static, []string{"waku/rs/1/1"})
	require.NoError(t, err)

	time.Sleep(time.Second * 2)

	defer wakuNode.Stop()
}
