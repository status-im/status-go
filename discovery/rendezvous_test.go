package discovery

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	lcrypto "github.com/libp2p/go-libp2p-crypto"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/status-im/rendezvous/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/storage"
)

func TestRendezvousDiscovery(t *testing.T) {
	priv, _, err := lcrypto.GenerateKeyPair(lcrypto.Secp256k1, 0)
	require.NoError(t, err)
	laddr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/7777"))
	require.NoError(t, err)
	db, err := leveldb.Open(storage.NewMemStorage(), nil)
	require.NoError(t, err)
	srv := server.NewServer(laddr, priv, server.NewStorage(db))
	require.NoError(t, srv.Start())

	identity, err := crypto.GenerateKey()
	require.NoError(t, err)
	node := discover.NewNode(discover.PubkeyID(&identity.PublicKey), net.IP{10, 10, 10, 10}, 10, 20)
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
