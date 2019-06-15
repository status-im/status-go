package state

import (
	"sync"
)

type memorySyncState struct {
	sync.Mutex

	state map[GroupID]map[MessageID]map[PeerID]State
}

func NewSyncState() *memorySyncState {
	return &memorySyncState{
		state: make(map[GroupID]map[MessageID]map[PeerID]State),
	}
}

func (s *memorySyncState) Get(group GroupID, id MessageID, peer PeerID) (State, error) {
	s.Lock()
	defer s.Unlock()

	state, _ := s.state[group][id][peer]
	return state, nil
}

func (s *memorySyncState) Set(group GroupID, id MessageID, peer PeerID, newState State) error {
	s.Lock()
	defer s.Unlock()

	if _, ok := s.state[group]; !ok {
		s.state[group] = make(map[MessageID]map[PeerID]State)
	}

	if _, ok := s.state[group][id]; !ok {
		s.state[group][id] = make(map[PeerID]State)
	}

	s.state[group][id][peer] = newState
	return nil
}

func (s *memorySyncState) Remove(group GroupID, id MessageID, peer PeerID) error {
	s.Lock()
	defer s.Unlock()

	delete(s.state[group][id], peer)
	return nil
}

func (s *memorySyncState) Map(epoch int64, process func(GroupID, MessageID, PeerID, State) State) error {
	s.Lock()
	defer s.Unlock()

	for group, syncstate := range s.state {
		for id, peers := range syncstate {
			for peer, state := range peers {
				if state.SendEpoch < epoch {
					continue
				}

				s.state[group][id][peer] = process(group, id, peer, state)
			}
		}
	}

	return nil
}
