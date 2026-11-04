package peer_exchange

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoremem"
	"github.com/multiformats/go-multiaddr"
	"github.com/waku-org/go-waku/waku/v2/onlinechecker"
	"github.com/waku-org/go-waku/waku/v2/peermanager"
	wps "github.com/waku-org/go-waku/waku/v2/peerstore"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/discv5"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	wenr "github.com/waku-org/go-waku/waku/v2/protocol/enr"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func createHostWithDiscv5(t *testing.T, bootnode ...*enode.Node) (host.Host, *discv5.DiscoveryV5) {
	ps, err := pstoremem.NewPeerstore()
	require.NoError(t, err)
	wakuPeerStore := wps.NewWakuPeerstore(ps)

	host, _, prvKey := tests.CreateHost(t, libp2p.Peerstore(wakuPeerStore))

	port, err := tests.FindFreePort(t, "127.0.0.1", 3)
	require.NoError(t, err)
	ip, _ := tests.ExtractIP(host.Addrs()[0])
	l, err := tests.NewLocalnode(prvKey, ip, port, wenr.NewWakuEnrBitfield(false, false, false, true), nil, utils.Logger())
	require.NoError(t, err)
	discv5PeerConn1 := discv5.NewTestPeerDiscoverer()
	d, err := discv5.NewDiscoveryV5(prvKey, l, discv5PeerConn1, prometheus.DefaultRegisterer, utils.Logger(), discv5.WithUDPPort(uint(port)), discv5.WithBootnodes(bootnode))
	require.NoError(t, err)
	d.SetHost(host)

	return host, d
}

func TestRetrieveProvidePeerExchangePeers(t *testing.T) {
	// H1
	host1, _, prvKey1 := tests.CreateHost(t)
	udpPort1, err := tests.FindFreePort(t, "127.0.0.1", 3)
	require.NoError(t, err)
	ip1, _ := tests.ExtractIP(host1.Addrs()[0])
	l1, err := tests.NewLocalnode(prvKey1, ip1, udpPort1, wenr.NewWakuEnrBitfield(false, false, false, true), nil, utils.Logger())
	require.NoError(t, err)
	err = wenr.Update(utils.Logger(), l1, wenr.WithWakuRelaySharding(protocol.RelayShards{ClusterID: 16, ShardIDs: []uint16{32}}))
	require.NoError(t, err)
	discv5PeerConn1 := discv5.NewTestPeerDiscoverer()
	d1, err := discv5.NewDiscoveryV5(prvKey1, l1, discv5PeerConn1, prometheus.DefaultRegisterer, utils.Logger(), discv5.WithUDPPort(uint(udpPort1)))
	require.NoError(t, err)
	d1.SetHost(host1)

	// H2
	host2, _, prvKey2 := tests.CreateHost(t)
	ip2, _ := tests.ExtractIP(host2.Addrs()[0])
	udpPort2, err := tests.FindFreePort(t, "127.0.0.1", 3)
	require.NoError(t, err)
	l2, err := tests.NewLocalnode(prvKey2, ip2, udpPort2, wenr.NewWakuEnrBitfield(false, false, false, true), nil, utils.Logger())
	require.NoError(t, err)
	err = wenr.Update(utils.Logger(), l2, wenr.WithWakuRelaySharding(protocol.RelayShards{ClusterID: 16, ShardIDs: []uint16{32}}))
	require.NoError(t, err)
	discv5PeerConn2 := discv5.NewTestPeerDiscoverer()
	d2, err := discv5.NewDiscoveryV5(prvKey2, l2, discv5PeerConn2, prometheus.DefaultRegisterer, utils.Logger(), discv5.WithUDPPort(uint(udpPort2)), discv5.WithBootnodes([]*enode.Node{d1.Node()}))
	require.NoError(t, err)
	d2.SetHost(host2)

	// H3
	host3, _, _ := tests.CreateHost(t)

	defer d1.Stop()
	defer d2.Stop()
	defer host1.Close()
	defer host2.Close()
	defer host3.Close()

	err = d1.Start(context.Background())
	require.NoError(t, err)

	err = d2.Start(context.Background())
	require.NoError(t, err)

	// mount peer exchange
	pxPeerConn1 := discv5.NewTestPeerDiscoverer()
	px1, err := NewWakuPeerExchange(d1, 0, pxPeerConn1, nil, prometheus.DefaultRegisterer, utils.Logger())
	require.NoError(t, err)
	px1.SetHost(host1)

	pxPeerConn3 := discv5.NewTestPeerDiscoverer()
	px3, err := NewWakuPeerExchange(nil, 0, pxPeerConn3, nil, prometheus.DefaultRegisterer, utils.Logger())
	require.NoError(t, err)
	px3.SetHost(host3)

	err = px1.Start(context.Background())
	require.NoError(t, err)

	err = px3.Start(context.Background())
	require.NoError(t, err)

	host3.Peerstore().AddAddrs(host1.ID(), host1.Addrs(), peerstore.PermanentAddrTTL)
	err = host3.Peerstore().AddProtocols(host1.ID(), PeerExchangeID_v20alpha1)
	require.NoError(t, err)

	time.Sleep(3 * time.Second) // Wait some time for peers to be discovered

	err = px3.Request(context.Background(), 1, WithPeer(host1.ID()))
	require.NoError(t, err)

	time.Sleep(3 * time.Second) //  Give the algorithm some time to work its magic

	require.True(t, pxPeerConn3.HasPeer(host2.ID()))

	px1.Stop()
	px3.Stop()
}

func TestRetrieveFilteredPeerExchangePeers(t *testing.T) {
	// H1
	host1, _, prvKey1 := tests.CreateHost(t)
	udpPort1, err := tests.FindFreePort(t, "127.0.0.1", 3)
	require.NoError(t, err)
	ip1, _ := tests.ExtractIP(host1.Addrs()[0])
	l1, err := tests.NewLocalnode(prvKey1, ip1, udpPort1, wenr.NewWakuEnrBitfield(false, false, false, true), nil, utils.Logger())
	require.NoError(t, err)

	discv5PeerConn1 := discv5.NewTestPeerDiscoverer()
	d1, err := discv5.NewDiscoveryV5(prvKey1, l1, discv5PeerConn1, prometheus.DefaultRegisterer, utils.Logger(), discv5.WithUDPPort(uint(udpPort1)))
	require.NoError(t, err)
	d1.SetHost(host1)

	// H2
	host2, _, prvKey2 := tests.CreateHost(t)
	ip2, _ := tests.ExtractIP(host2.Addrs()[0])
	udpPort2, err := tests.FindFreePort(t, "127.0.0.1", 3)
	require.NoError(t, err)
	l2, err := tests.NewLocalnode(prvKey2, ip2, udpPort2, wenr.NewWakuEnrBitfield(false, false, false, true), nil, utils.Logger())
	require.NoError(t, err)
	rs, err := protocol.NewRelayShards(1, 2)
	require.NoError(t, err)
	l2.Set(enr.WithEntry(wenr.ShardingBitVectorEnrField, rs.BitVector()))
	discv5PeerConn2 := discv5.NewTestPeerDiscoverer()
	d2, err := discv5.NewDiscoveryV5(prvKey2, l2, discv5PeerConn2, prometheus.DefaultRegisterer, utils.Logger(), discv5.WithUDPPort(uint(udpPort2)), discv5.WithBootnodes([]*enode.Node{d1.Node()}))
	require.NoError(t, err)
	d2.SetHost(host2)

	// H3
	host3, _, _ := tests.CreateHost(t)

	defer d1.Stop()
	defer d2.Stop()
	defer host1.Close()
	defer host2.Close()
	defer host3.Close()

	err = d1.Start(context.Background())
	require.NoError(t, err)

	err = d2.Start(context.Background())
	require.NoError(t, err)

	time.Sleep(3 * time.Second) // Wait some time for peers to be discovered

	// mount peer exchange
	pxPeerConn1 := discv5.NewTestPeerDiscoverer()
	px1, err := NewWakuPeerExchange(d1, 0, pxPeerConn1, nil, prometheus.DefaultRegisterer, utils.Logger())
	require.NoError(t, err)
	px1.SetHost(host1)

	pxPeerConn3 := discv5.NewTestPeerDiscoverer()
	px3, err := NewWakuPeerExchange(nil, 0, pxPeerConn3, nil, prometheus.DefaultRegisterer, utils.Logger())
	require.NoError(t, err)
	px3.SetHost(host3)

	err = px1.Start(context.Background())
	require.NoError(t, err)

	err = px3.Start(context.Background())
	require.NoError(t, err)

	host3.Peerstore().AddAddrs(host1.ID(), host1.Addrs(), peerstore.PermanentAddrTTL)
	err = host3.Peerstore().AddProtocols(host1.ID(), PeerExchangeID_v20alpha1)
	require.NoError(t, err)

	//Try with shard that is not registered.
	err = px3.Request(context.Background(), 1, WithPeer(host1.ID()), FilterByShard(1, 3))
	require.NoError(t, err)

	time.Sleep(3 * time.Second) //  Give the algorithm some time to work its magic

	require.False(t, pxPeerConn3.HasPeer(host2.ID()))

	//Try without shard filtering

	err = px3.Request(context.Background(), 1, WithPeer(host1.ID()))
	require.NoError(t, err)

	time.Sleep(3 * time.Second) //  Give the algorithm some time to work its magic

	require.True(t, pxPeerConn3.HasPeer(host2.ID()))

	err = px3.Request(context.Background(), 1, WithPeer(host1.ID()), FilterByShard(1, 2))
	require.NoError(t, err)

	time.Sleep(3 * time.Second) //  Give the algorithm some time to work its magic

	require.True(t, pxPeerConn3.HasPeer(host2.ID()))

	px1.Stop()
	px3.Stop()
}

func TestPeerExchangeOptions(t *testing.T) {

	// Prepare host1
	host1, _, prvKey1 := tests.CreateHost(t)
	udpPort1, err := tests.FindFreePort(t, "127.0.0.1", 3)
	require.NoError(t, err)
	ip1, _ := tests.ExtractIP(host1.Addrs()[0])
	l1, err := tests.NewLocalnode(prvKey1, ip1, udpPort1, wenr.NewWakuEnrBitfield(false, false, false, true), nil, utils.Logger())
	require.NoError(t, err)
	discv5PeerConn1 := discv5.NewTestPeerDiscoverer()
	d1, err := discv5.NewDiscoveryV5(prvKey1, l1, discv5PeerConn1, prometheus.DefaultRegisterer, utils.Logger(), discv5.WithUDPPort(uint(udpPort1)))
	require.NoError(t, err)
	d1.SetHost(host1)

	// Mount peer exchange
	pxPeerConn1 := discv5.NewTestPeerDiscoverer()
	px1, err := NewWakuPeerExchange(d1, 0, pxPeerConn1, nil, prometheus.DefaultRegisterer, utils.Logger())
	require.NoError(t, err)
	px1.SetHost(host1)

	// Test WithPeerAddr()
	params := new(PeerExchangeRequestParameters)
	params.host = px1.h
	params.log = px1.log
	params.pm = px1.pm

	optList := DefaultOptions(px1.h)
	optList = append(optList, WithPeerAddr(host1.Addrs()[0]))
	for _, opt := range optList {
		err := opt(params)
		require.NoError(t, err)
	}

	require.Equal(t, host1.Addrs()[0], params.peerAddr)

	// Test WithFastestPeerSelection()
	optList = DefaultOptions(px1.h)
	optList = append(optList, WithFastestPeerSelection(host1.ID()))
	for _, opt := range optList {
		err := opt(params)
		require.NoError(t, err)
	}

	require.Equal(t, peermanager.LowestRTT, params.peerSelectionType)
	require.Equal(t, host1.ID(), params.preferredPeers[0])

}

func TestRetrieveProvidePeerExchangeWithPMAndPeerAddr(t *testing.T) {
	log := utils.Logger()

	// H1 + H2 with discovery on
	host1, d1 := createHostWithDiscv5(t)
	host2, d2 := createHostWithDiscv5(t, d1.Node())

	// H3
	ps3, err := pstoremem.NewPeerstore()
	require.NoError(t, err)
	wakuPeerStore3 := wps.NewWakuPeerstore(ps3)
	host3, _, _ := tests.CreateHost(t, libp2p.Peerstore(wakuPeerStore3))

	defer d1.Stop()
	defer d2.Stop()
	defer host1.Close()
	defer host2.Close()
	defer host3.Close()

	err = d1.Start(context.Background())
	require.NoError(t, err)

	err = d2.Start(context.Background())
	require.NoError(t, err)

	// Prepare peer manager for host3
	pm3 := peermanager.NewPeerManager(10, 20, nil, nil, true, log)
	pm3.SetHost(host3)
	pxPeerConn3, err := peermanager.NewPeerConnectionStrategy(pm3, onlinechecker.NewDefaultOnlineChecker(true), 30*time.Second, utils.Logger())
	require.NoError(t, err)
	pxPeerConn3.SetHost(host3)
	err = pxPeerConn3.Start(context.Background())
	require.NoError(t, err)

	// mount peer exchange
	pxPeerConn1 := discv5.NewTestPeerDiscoverer()
	px1, err := NewWakuPeerExchange(d1, 0, pxPeerConn1, nil, prometheus.DefaultRegisterer, utils.Logger())
	require.NoError(t, err)
	px1.SetHost(host1)

	px3, err := NewWakuPeerExchange(nil, 0, pxPeerConn3, pm3, prometheus.DefaultRegisterer, utils.Logger())
	require.NoError(t, err)
	px3.SetHost(host3)

	err = px1.Start(context.Background())
	require.NoError(t, err)

	time.Sleep(5 * time.Second)

	// Check peer store of the client doesn't contain peer2 before the request
	require.False(t, slices.Contains(host3.Peerstore().Peers(), host2.ID()))

	// Construct multi address like example "/ip4/0.0.0.0/tcp/30304/p2p/16Uiu2HAmBu5zRFzBGAzzMAuGWhaxN2EwcbW7CzibELQELzisf192"
	hostInfo, err := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", host1.ID()))
	require.NoError(t, err)
	host1MultiAddr := host1.Addrs()[0].Encapsulate(hostInfo)
	require.NoError(t, err)

	log.Info("Connecting to peer", zap.String(host1MultiAddr.String(), "to provide 1 peer"))
	err = px3.Request(context.Background(), 1, WithPeerAddr(host1MultiAddr))
	require.NoError(t, err)

	time.Sleep(5 * time.Second)

	// Check peer store of the client does contain peer2 after the request
	require.True(t, slices.Contains(host3.Peerstore().Peers(), host2.ID()))

}

func TestRetrieveProvidePeerExchangeWithPMOnly(t *testing.T) {
	log := utils.Logger()

	// H1 + H2 + H4 with discovery on
	host1, d1 := createHostWithDiscv5(t)
	host2, d2 := createHostWithDiscv5(t, d1.Node())
	host4, d4 := createHostWithDiscv5(t, d1.Node())

	// H3
	ps3, err := pstoremem.NewPeerstore()
	require.NoError(t, err)
	wakuPeerStore3 := wps.NewWakuPeerstore(ps3)
	host3, _, _ := tests.CreateHost(t, libp2p.Peerstore(wakuPeerStore3))

	defer d1.Stop()
	defer d2.Stop()
	defer d4.Stop()
	defer host1.Close()
	defer host2.Close()
	defer host3.Close()
	defer host4.Close()

	err = d1.Start(context.Background())
	require.NoError(t, err)

	err = d2.Start(context.Background())
	require.NoError(t, err)

	err = d4.Start(context.Background())
	require.NoError(t, err)

	// Prepare peer manager for host3
	pm3 := peermanager.NewPeerManager(10, 20, nil, nil, true, log)
	pm3.SetHost(host3)
	pxPeerConn3, err := peermanager.NewPeerConnectionStrategy(pm3, onlinechecker.NewDefaultOnlineChecker(true), 30*time.Second, utils.Logger())
	require.NoError(t, err)
	pxPeerConn3.SetHost(host3)
	err = pxPeerConn3.Start(context.Background())
	require.NoError(t, err)

	// Add peer exchange server to client's peer store
	host3.Peerstore().AddAddrs(host1.ID(), host1.Addrs(), peerstore.PermanentAddrTTL)
	err = host3.Peerstore().AddProtocols(host1.ID(), PeerExchangeID_v20alpha1)
	require.NoError(t, err)

	// mount peer exchange
	pxPeerConn1 := discv5.NewTestPeerDiscoverer()
	px1, err := NewWakuPeerExchange(d1, 0, pxPeerConn1, nil, prometheus.DefaultRegisterer, utils.Logger())
	require.NoError(t, err)
	px1.SetHost(host1)

	px3, err := NewWakuPeerExchange(nil, 0, pxPeerConn3, pm3, prometheus.DefaultRegisterer, utils.Logger())
	require.NoError(t, err)
	px3.SetHost(host3)

	err = px1.Start(context.Background())
	require.NoError(t, err)

	time.Sleep(5 * time.Second)

	// Check peer store of the client doesn't contain peer2 and peer4 before the request
	require.False(t, slices.Contains(host3.Peerstore().Peers(), host2.ID()))
	require.False(t, slices.Contains(host3.Peerstore().Peers(), host4.ID()))

	// Connect to PM to get peers
	err = px3.Request(context.Background(), 2, WithAutomaticPeerSelection(host1.ID()))
	require.NoError(t, err)

	time.Sleep(5 * time.Second)

	// Check peer store of the client does contain peer2 and peer4 after the request
	require.True(t, slices.Contains(host3.Peerstore().Peers(), host2.ID()))
	require.True(t, slices.Contains(host3.Peerstore().Peers(), host4.ID()))

}

func TestRecordError(t *testing.T) {

	m := newMetrics(prometheus.DefaultRegisterer)

	// Increment error counter for rateLimitFailure 7 times
	for i := 0; i < 7; i++ {
		m.RecordError(rateLimitFailure)
	}

	// Retrieve metric values
	counter, _ := peerExchangeErrors.GetMetricWithLabelValues(string(rateLimitFailure))
	rateLimitFailures := &dto.Metric{}

	// Store values into metric client struct
	err := counter.Write(rateLimitFailures)
	require.NoError(t, err)

	// Check the count is in
	require.Equal(t, 7, int(rateLimitFailures.GetCounter().GetValue()))

}
