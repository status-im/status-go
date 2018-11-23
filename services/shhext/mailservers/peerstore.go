package mailservers

import (
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

var (
	// ErrNoConnected returned when mail servers are not connected.
	ErrNoConnected = errors.New("no connected mail servers")
	// ErrNoProvider returned if provider for connectected mail servers wasn't set.
	ErrNoProvider = errors.New("no provider for connected mail servers")
)

// PeersProvider is an interface for requesting peers.
type PeersProvider interface {
	Peers() []*p2p.Peer
}

// NewPeerStore returns an instance of PeerStore.
func NewPeerStore() *PeerStore {
	return &PeerStore{nodes: map[enode.ID]*enode.Node{}}
}

// PeerStore stores list of selected mail servers and keeps N of them connected.
type PeerStore struct {
	mu    sync.RWMutex
	nodes map[enode.ID]*enode.Node

	server PeersProvider
}

// Exist confirms that peers was added to a store.
func (ps *PeerStore) Exist(peer enode.ID) bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	_, exist := ps.nodes[peer]
	return exist
}

// Get returns instance of the node with requested ID or nil if ID is not found.
func (ps *PeerStore) Get(peer enode.ID) *enode.Node {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.nodes[peer]
}

// Update updates peers locally.
func (ps *PeerStore) Update(nodes []*enode.Node) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.nodes = map[enode.ID]*enode.Node{}
	for _, n := range nodes {
		ps.nodes[n.ID()] = n
	}
}

// UsePeersProvider sets given provider as a provider for PeerStore.
func (ps *PeerStore) UsePeersProvider(provider PeersProvider) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	ps.server = provider
}

// GetFirstConnected returns first connected peers that is also added to a peer store.
// Raises ErrNoConnected.
func (ps *PeerStore) GetFirstConnected() (*enode.Node, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	if ps.server == nil {
		return nil, ErrNoProvider
	}
	peers := ps.server.Peers()
	for _, p := range peers {
		if n, exist := ps.nodes[p.ID()]; exist {
			return n, nil
		}
	}
	return nil, ErrNoConnected
}
