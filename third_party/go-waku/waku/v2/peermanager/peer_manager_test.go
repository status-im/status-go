package peermanager

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	libp2pProtocol "github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoremem"
	"github.com/multiformats/go-multiaddr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/discv5"
	"github.com/waku-org/go-waku/waku/v2/onlinechecker"
	wps "github.com/waku-org/go-waku/waku/v2/peerstore"
	wakuproto "github.com/waku-org/go-waku/waku/v2/protocol"
	wenr "github.com/waku-org/go-waku/waku/v2/protocol/enr"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func initTest(t *testing.T) (context.Context, *PeerManager, func()) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	// hosts
	h1, err := tests.MakeHost(ctx, 0, rand.Reader)
	require.NoError(t, err)

	// host 1 is used by peer manager
	pm := NewPeerManager(10, 20, nil, nil, true, utils.Logger())
	pm.SetHost(h1)

	return ctx, pm, func() {
		cancel()
		h1.Close()
	}
}

func TestServiceSlots(t *testing.T) {
	ctx, pm, deferFn := initTest(t)
	defer deferFn()

	h2, err := tests.MakeHost(ctx, 0, rand.Reader)
	require.NoError(t, err)
	defer h2.Close()

	h3, err := tests.MakeHost(ctx, 0, rand.Reader)
	require.NoError(t, err)
	defer h3.Close()
	// protocols
	protocol := libp2pProtocol.ID("test/protocol")
	protocol1 := libp2pProtocol.ID("test/protocol1")

	// add h2 peer to peer manager
	t.Log(h2.ID())
	_, err = pm.AddPeer([]multiaddr.Multiaddr{tests.GetAddr(h2)}, wps.Static, []string{""}, libp2pProtocol.ID(protocol))
	require.NoError(t, err)

	///////////////
	// getting peer for protocol
	///////////////

	var peerID peer.ID
	// select peer from pm, currently only h2 is set in pm
	peers, err := pm.SelectPeers(PeerSelectionCriteria{SelectionType: Automatic, Proto: protocol})
	require.NoError(t, err)
	require.Equal(t, h2.ID(), peers[0])

	// add h3 peer to peer manager
	_, err = pm.AddPeer([]multiaddr.Multiaddr{tests.GetAddr(h3)}, wps.Static, []string{""}, libp2pProtocol.ID(protocol))
	require.NoError(t, err)

	// check that returned peer is h2 or h3 peer
	peers, err = pm.SelectPeers(PeerSelectionCriteria{SelectionType: Automatic, Proto: protocol})
	peerID = peers[0]
	require.NoError(t, err)
	if peerID == h2.ID() || peerID == h3.ID() {
		//Test success
		t.Log("Random peer selection per protocol successful")
	} else {
		t.FailNow()
	}

	///////////////
	// getting peer for protocol1
	///////////////
	h4, err := tests.MakeHost(ctx, 0, rand.Reader)
	require.NoError(t, err)
	defer h4.Close()

	_, err = pm.SelectPeers(PeerSelectionCriteria{SelectionType: Automatic, Proto: protocol1})
	require.Error(t, err, utils.ErrNoPeersAvailable)

	// add h4 peer for protocol1
	_, err = pm.AddPeer([]multiaddr.Multiaddr{tests.GetAddr(h4)}, wps.Static, []string{""}, libp2pProtocol.ID(protocol1))
	require.NoError(t, err)

	//Test peer selection for protocol1
	peers, err = pm.SelectPeers(PeerSelectionCriteria{SelectionType: Automatic, Proto: protocol1})
	peerID = peers[0]

	require.NoError(t, err)
	require.Equal(t, peerID, h4.ID())

	_, err = pm.SelectPeerByContentTopics(protocol1, []string{""})
	require.Error(t, wakuproto.ErrInvalidFormat, err)

}

func TestPeerSelection(t *testing.T) {
	ctx, pm, deferFn := initTest(t)
	defer deferFn()

	h2, err := tests.MakeHost(ctx, 0, rand.Reader)
	require.NoError(t, err)
	defer h2.Close()

	h3, err := tests.MakeHost(ctx, 0, rand.Reader)
	require.NoError(t, err)
	defer h3.Close()

	protocol := libp2pProtocol.ID("test/protocol")
	_, err = pm.AddPeer([]multiaddr.Multiaddr{tests.GetAddr(h2)}, wps.Static, []string{"/waku/2/rs/2/1", "/waku/2/rs/2/2"}, protocol)
	require.NoError(t, err)

	_, err = pm.AddPeer([]multiaddr.Multiaddr{tests.GetAddr(h3)}, wps.Static, []string{"/waku/2/rs/2/1"}, protocol)
	require.NoError(t, err)

	var peerIDs peer.IDSlice

	peerIDs, err = pm.SelectPeers(PeerSelectionCriteria{SelectionType: Automatic, Proto: protocol})
	require.NoError(t, err)
	require.Len(t, peerIDs, 1) // Only 1 peer is returned randomly, because MaxPeers defaults to 1 when not set

	peerIDs, err = pm.SelectPeers(PeerSelectionCriteria{SelectionType: Automatic, Proto: protocol, PubsubTopics: []string{"/waku/2/rs/2/2"}})
	require.NoError(t, err)
	require.Len(t, peerIDs, 1)
	require.Equal(t, h2.ID(), peerIDs[0]) // Only h2 has this pubsub topic

	peerIDs, err = pm.SelectPeers(PeerSelectionCriteria{SelectionType: Automatic, Proto: protocol, PubsubTopics: []string{"/waku/2/rs/2/3"}})
	require.Error(t, utils.ErrNoPeersAvailable, err) // No peer has this pubsub topic
	require.Empty(t, peerIDs)

	peerIDs, err = pm.SelectPeers(PeerSelectionCriteria{SelectionType: Automatic, Proto: protocol, PubsubTopics: []string{"/waku/2/rs/2/1"}})
	require.NoError(t, err)
	require.Len(t, peerIDs, 1) // Both h2 and h3 have this pubsub topic, but only 1 peer is returned randomly because MaxPeers defaults to 1 when not set

	peerIDs, err = pm.SelectPeers(PeerSelectionCriteria{SelectionType: Automatic, Proto: protocol, PubsubTopics: []string{"/waku/2/rs/2/1"}, MaxPeers: 2})
	require.NoError(t, err)
	require.Len(t, peerIDs, 2)

	peerIDs, err = pm.SelectPeers(PeerSelectionCriteria{SelectionType: Automatic, Proto: protocol, PubsubTopics: []string{"/waku/2/rs/2/1"}, MaxPeers: 3})
	require.NoError(t, err)
	require.Len(t, peerIDs, 2)

	h4, err := tests.MakeHost(ctx, 0, rand.Reader)
	require.NoError(t, err)
	defer h4.Close()
	_, err = pm.AddPeer([]multiaddr.Multiaddr{tests.GetAddr(h4)}, wps.Static, []string{"/waku/2/rs/2/1"}, libp2pProtocol.ID(protocol))
	require.NoError(t, err)

	peerIDs, err = pm.SelectPeers(PeerSelectionCriteria{SelectionType: Automatic, Proto: protocol, PubsubTopics: []string{"/waku/2/rs/2/1"}, MaxPeers: 3})
	require.Len(t, peerIDs, 3)
	require.NoError(t, err)

	//Test for selectWithLowestRTT
	// NOTE: This test must go the last because it involves pinging peers, which modifies the list of supported protocols
	peerIDs, err = pm.SelectPeers(PeerSelectionCriteria{SelectionType: LowestRTT, Proto: protocol, PubsubTopics: []string{"/waku/2/rs/2/1"}, MaxPeers: 2})
	require.NoError(t, err)
	require.Len(t, peerIDs, 1) // With LowestRTT, only 1 peer is returned, even if MaxPeers is set
}

func TestDefaultProtocol(t *testing.T) {
	ctx, pm, deferFn := initTest(t)
	defer deferFn()
	///////////////
	// check peer for default protocol
	///////////////
	//Test empty peer selection for relay protocol
	_, err := pm.SelectPeers(PeerSelectionCriteria{SelectionType: Automatic, Proto: relay.WakuRelayID_v200})
	require.Error(t, err, utils.ErrNoPeersAvailable)

	///////////////
	// getting peer for default protocol
	///////////////
	h5, err := tests.MakeHost(ctx, 0, rand.Reader)
	require.NoError(t, err)
	defer h5.Close()

	//Test peer selection for relay protocol from peer store
	_, err = pm.AddPeer([]multiaddr.Multiaddr{tests.GetAddr(h5)}, wps.Static, []string{""}, relay.WakuRelayID_v200)
	require.NoError(t, err)

	// since we are not passing peerList, selectPeer fn using filterByProto checks in PeerStore for peers with same protocol.
	peerIDs, err := pm.SelectPeers(PeerSelectionCriteria{SelectionType: Automatic, Proto: relay.WakuRelayID_v200})
	require.NoError(t, err)
	require.Equal(t, h5.ID(), peerIDs[0])
}

func TestAdditionAndRemovalOfPeer(t *testing.T) {
	ctx, pm, deferFn := initTest(t)
	defer deferFn()
	///////////////
	// set h6 peer for protocol2 and remove that peer and check again
	///////////////
	//Test random peer selection
	protocol2 := libp2pProtocol.ID("test/protocol2")
	h6, err := tests.MakeHost(ctx, 0, rand.Reader)
	require.NoError(t, err)
	defer h6.Close()

	_, err = pm.AddPeer([]multiaddr.Multiaddr{tests.GetAddr(h6)}, wps.Static, []string{""}, protocol2)
	require.NoError(t, err)

	peers, err := pm.SelectPeers(PeerSelectionCriteria{SelectionType: Automatic, Proto: protocol2})
	require.NoError(t, err)
	require.Equal(t, peers[0], h6.ID())

	pm.RemovePeer(peers[0])
	_, err = pm.SelectPeers(PeerSelectionCriteria{SelectionType: Automatic, Proto: protocol2})
	require.Error(t, err, utils.ErrNoPeersAvailable)
}

func TestConnectToRelayPeers(t *testing.T) {

	ctx, pm, deferFn := initTest(t)
	pc, err := NewPeerConnectionStrategy(pm, onlinechecker.NewDefaultOnlineChecker(true), 120*time.Second, pm.logger)
	require.NoError(t, err)
	pc.SetHost(pm.host)
	err = pc.Start(ctx)
	require.NoError(t, err)
	pm.Start(ctx)

	defer deferFn()

	pm.connectToPeers()

}

func createHostWithDiscv5AndPM(t *testing.T, hostName string, topic string, enrField uint8, bootnode ...*enode.Node) (host.Host, *PeerManager, *discv5.DiscoveryV5) {
	ps, err := pstoremem.NewPeerstore()
	require.NoError(t, err)
	wakuPeerStore := wps.NewWakuPeerstore(ps)

	host, _, prvKey1 := tests.CreateHost(t, libp2p.Peerstore(wakuPeerStore))

	logger := utils.Logger().Named(hostName)

	udpPort, err := tests.FindFreeUDPPort(t, "127.0.0.1", 3)
	require.NoError(t, err)
	ip1, _ := tests.ExtractIP(host.Addrs()[0])
	localNode, err := tests.NewLocalnode(prvKey1, ip1, udpPort, enrField, nil, logger)
	require.NoError(t, err)

	rs, err := wakuproto.TopicsToRelayShards(topic)
	require.NoError(t, err)

	err = wenr.Update(utils.Logger(), localNode, wenr.WithWakuRelaySharding(rs[0]))
	require.NoError(t, err)
	pm := NewPeerManager(10, 20, nil, nil, true, logger)
	pm.SetHost(host)
	peerconn, err := NewPeerConnectionStrategy(pm, onlinechecker.NewDefaultOnlineChecker(true), 30*time.Second, logger)
	require.NoError(t, err)
	discv5, err := discv5.NewDiscoveryV5(prvKey1, localNode, peerconn, prometheus.DefaultRegisterer, logger, discv5.WithUDPPort(uint(udpPort)), discv5.WithBootnodes(bootnode))
	require.NoError(t, err)
	discv5.SetHost(host)
	pm.SetDiscv5(discv5)
	pm.SetPeerConnector(peerconn)

	return host, pm, discv5
}

func TestOnDemandPeerDiscovery(t *testing.T) {
	topic := "/waku/2/rs/1/1"

	// Host1 <-> Host2 <-> Host3
	host1, _, d1 := createHostWithDiscv5AndPM(t, "host1", topic, wenr.NewWakuEnrBitfield(true, true, false, true))

	host2, _, d2 := createHostWithDiscv5AndPM(t, "host2", topic, wenr.NewWakuEnrBitfield(false, true, true, true), d1.Node())
	host3, pm3, d3 := createHostWithDiscv5AndPM(t, "host3", topic, wenr.NewWakuEnrBitfield(true, true, true, true), d2.Node())

	defer d1.Stop()
	defer d2.Stop()
	defer d3.Stop()

	defer host1.Close()
	defer host2.Close()
	defer host3.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := d1.Start(ctx)
	require.NoError(t, err)

	err = d2.Start(ctx)
	require.NoError(t, err)

	err = d3.Start(ctx)
	require.NoError(t, err)

	//Discovery should fail for non-waku protocol
	_, err = pm3.SelectPeers(PeerSelectionCriteria{SelectionType: Automatic, PubsubTopics: []string{topic}, Proto: "/test"})
	require.Error(t, err)

	_, err = pm3.SelectPeers(PeerSelectionCriteria{SelectionType: Automatic, Proto: "/test"})
	require.Error(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var enrField uint8
	enrField |= (1 << 1)
	pm3.RegisterWakuProtocol("/vac/waku/store/2.0.0-beta4", enrField)
	peerIDs, err := pm3.SelectPeers(PeerSelectionCriteria{SelectionType: Automatic, PubsubTopics: []string{topic}, Proto: "/vac/waku/store/2.0.0-beta4", Ctx: ctx})
	require.NoError(t, err)
	require.Equal(t, host2.ID(), peerIDs[0])

	var enrField1 uint8

	enrField1 |= (1 << 3)
	pm3.RegisterWakuProtocol("/vac/waku/lightpush/2.0.0-beta1", enrField1)
	peerIDs, err = pm3.SelectPeers(PeerSelectionCriteria{SelectionType: Automatic, PubsubTopics: []string{topic}, Proto: "/vac/waku/lightpush/2.0.0-beta1", Ctx: ctx})
	require.NoError(t, err)
	require.Equal(t, host1.ID(), peerIDs[0])

}
