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
)

// PeersProvider is an interface for requesting list of peers.
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

// GetFirstConnected returns first connected peer that is also added to a peer store.
// Raises ErrNoConnected if no peers are added to a peer store.
func GetFirstConnected(provider PeersProvider, store *PeerStore) (*enode.Node, error) {
	peers := provider.Peers()
	for _, p := range peers {
		if store.Exist(p.ID()) {
			return p.Node(), nil
		}
	}
	return nil, ErrNoConnected
}
