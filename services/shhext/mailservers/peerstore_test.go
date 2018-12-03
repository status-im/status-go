package mailservers

import (
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/stretchr/testify/require"
)

func RandomeNode() (*enode.Node, error) {
	pkey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	return enode.NewV4(&pkey.PublicKey, nil, 0, 0), nil
}

func TestUpdateResetsInternalStorage(t *testing.T) {
	store := NewPeerStore()
	r1, err := RandomeNode()
	require.NoError(t, err)
	r2, err := RandomeNode()
	require.NoError(t, err)
	store.Update([]*enode.Node{r1, r2})
	require.True(t, store.Exist(r1.ID()))
	require.True(t, store.Exist(r2.ID()))
	store.Update([]*enode.Node{r2})
	require.False(t, store.Exist(r1.ID()))
	require.True(t, store.Exist(r2.ID()))
}

func TestGetNodeByID(t *testing.T) {
	store := NewPeerStore()
	r1, err := RandomeNode()
	require.NoError(t, err)
	store.Update([]*enode.Node{r1})
	require.Equal(t, r1, store.Get(r1.ID()))
	require.Nil(t, store.Get(enode.ID{1}))
}

type fakePeerProvider struct {
	peers []*p2p.Peer
}

func (f fakePeerProvider) Peers() []*p2p.Peer {
	return f.peers
}

func TestNoConnected(t *testing.T) {
	provider := fakePeerProvider{}
	store := NewPeerStore()
	_, err := GetFirstConnected(provider, store)
	require.EqualError(t, ErrNoConnected, err.Error())
}

func TestGetFirstConnected(t *testing.T) {
	numPeers := 3
	nodes := make([]*enode.Node, numPeers)
	peers := make([]*p2p.Peer, numPeers)
	nodesMap := getNRandomNodes(t, numPeers)
	i := 0
	for _, node := range nodesMap {
		nodes[i] = node
		peers[i] = p2p.NewPeer(node.ID(), node.ID().String(), nil)
		i++
	}
	store := NewPeerStore()
	provider := fakePeerProvider{peers}
	_, err := GetFirstConnected(provider, store)
	require.EqualError(t, ErrNoConnected, err.Error())
	store.Update(nodes)
	node, err := GetFirstConnected(provider, store)
	require.NoError(t, err)
	require.Contains(t, nodesMap, node.ID())
}
