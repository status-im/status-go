package peers

import (
	"github.com/vacp2p/mvds/state"
)

type Persistence interface {
	Add(state.GroupID, state.PeerID) error
	GetByGroupID(group state.GroupID) ([]state.PeerID, error)
	Exists(state.GroupID, state.PeerID) (bool, error)
}
