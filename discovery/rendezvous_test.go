package discovery

import (
	"net"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	lcrypto "github.com/libp2p/go-libp2p-crypto"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/status-im/rendezvous/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/storage"
)

func makeTestRendezvousServer(t *testing.T, addr string) *server.Server {
	priv, _, err := lcrypto.GenerateKeyPair(lcrypto.Secp256k1, 0)
	require.NoError(t, err)
	laddr, err := ma.NewMultiaddr(addr)
	require.NoError(t, err)
	db, err := leveldb.Open(storage.NewMemStorage(), nil)
	require.NoError(t, err)
	srv := server.NewServer(laddr, priv, server.NewStorage(db))
	require.NoError(t, srv.Start())
	return srv
}

func TestRendezvousDiscovery(t *testing.T) {
	srv := makeTestRendezvousServer(t, "/ip4/127.0.0.1/tcp/7777")
	defer srv.Stop()
	identity, err := crypto.GenerateKey()
	require.NoError(t, err)
	node := enode.NewV4(&identity.PublicKey, net.IP{10, 10, 10, 10}, 10, 20)
	c, err := NewRendezvous([]ma.Multiaddr{srv.Addr()}, identity, node)
	require.NoError(t, err)
	require.NoError(t, c.Start())
	require.True(t, c.Running())

	topic := "test"
	stop := make(chan struct{})
	go func() { assert.NoError(t, c.Register(topic, stop)) }()

	period := make(chan time.Duration, 1)
	period <- 100 * time.Millisecond
	found := make(chan *discv5.Node, 1)
	lookup := make(chan bool)
	go func() { assert.NoError(t, c.Discover(topic, period, found, lookup)) }()

	select {
	case n := <-found:
		assert.Equal(t, discv5.PubkeyID(&identity.PublicKey), n.ID)
	case <-time.After(10 * time.Second):
		assert.Fail(t, "failed waiting to discover a node")
	}
	close(stop)
	close(period)
}

func TestMakeRecordReturnsCachedRecord(t *testing.T) {
	identity, err := crypto.GenerateKey()
	require.NoError(t, err)
	record := enr.Record{}
	require.NoError(t, enode.SignV4(&record, identity))
	c := NewRendezvousWithENR(nil, record)
	rst, err := c.MakeRecord()
	require.NoError(t, err)
	require.NotNil(t, enode.V4ID{}.NodeAddr(&rst))
	require.Equal(t, enode.V4ID{}.NodeAddr(&record), enode.V4ID{}.NodeAddr(&rst))
}

func TestRendezvousRegisterAndDiscoverExitGracefully(t *testing.T) {
	r, err := NewRendezvous(make([]ma.Multiaddr, 1), nil, nil)
	require.NoError(t, err)
	require.NoError(t, r.Start())
	require.NoError(t, r.Stop())
	require.EqualError(t, errDiscoveryIsStopped, r.register("", enr.Record{}).Error())
	_, err = r.discoverRequest(nil, "")
	require.EqualError(t, errDiscoveryIsStopped, err.Error())
}

func BenchmarkRendezvousStart(b *testing.B) {
	identity, err := crypto.GenerateKey()
	require.NoError(b, err)
	addr, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/7777")
	require.NoError(b, err)
	node := enode.NewV4(&identity.PublicKey, net.IP{10, 10, 10, 10}, 10, 20)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		c, err := NewRendezvous([]ma.Multiaddr{addr}, identity, node)
		require.NoError(b, err)
		require.NoError(b, c.Start())
	}
}
