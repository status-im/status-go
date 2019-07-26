package peers

import (
	"github.com/vacp2p/mvds/state"
)

type Persistence interface {
	Add(state.GroupID, state.PeerID) error
	GetByGroupID(group state.GroupID) ([]state.PeerID, error)
	Exists(state.GroupID, state.PeerID) (bool, error)
}

type MemoryPersistence struct {
	peers map[state.GroupID][]state.PeerID
}

func NewMemoryPersistence() *MemoryPersistence {
	return &MemoryPersistence{
		peers: make(map[state.GroupID][]state.PeerID),
	}
}

func (p *MemoryPersistence) Add(groupID state.GroupID, peerID state.PeerID) error {
	p.peers[groupID] = append(p.peers[groupID], peerID)
	return nil
}

func (p *MemoryPersistence) Exists(groupID state.GroupID, peerID state.PeerID) (bool, error) {
	for _, peer := range p.peers[groupID] {
		if peer == peerID {
			return true, nil
		}
	}
	return false, nil
}

func (p *MemoryPersistence) GetByGroupID(groupID state.GroupID) ([]state.PeerID, error) {
	return p.peers[groupID], nil
}
