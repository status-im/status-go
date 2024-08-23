package peerstore

import (
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"golang.org/x/exp/maps"
)

// Origin is used to determine how the peer is identified,
// either it is statically added or discovered via one of the discovery protocols
type Origin int64

const (
	Unknown Origin = iota
	Discv5
	Static
	PeerExchange
	DNSDiscovery
	Rendezvous
	PeerManager
)

const peerOrigin = "origin"
const peerENR = "enr"
const peerDirection = "direction"
const peerPubSubTopics = "pubSubTopics"
const peerScore = "score"

// ConnectionFailures contains connection failure information towards all peers
type ConnectionFailures struct {
	sync.RWMutex
	failures map[peer.ID]int
}

// WakuPeerstoreImpl is a implementation of WakuPeerStore
type WakuPeerstoreImpl struct {
	peerStore    peerstore.Peerstore
	connFailures ConnectionFailures
}

// WakuPeerstore is an interface for implementing WakuPeerStore
type WakuPeerstore interface {
	SetOrigin(p peer.ID, origin Origin) error
	Origin(p peer.ID) (Origin, error)
	PeersByOrigin(origin Origin) peer.IDSlice
	SetENR(p peer.ID, enr *enode.Node) error
	ENR(p peer.ID) (*enode.Node, error)
	AddConnFailure(pID peer.ID)
	ResetConnFailures(pID peer.ID)
	ConnFailures(pID peer.ID) int

	SetDirection(p peer.ID, direction network.Direction) error
	Direction(p peer.ID) (network.Direction, error)

	AddPubSubTopic(p peer.ID, topic string) error
	RemovePubSubTopic(p peer.ID, topic string) error
	PubSubTopics(p peer.ID) (protocol.TopicSet, error)
	SetPubSubTopics(p peer.ID, topics []string) error
	PeersByPubSubTopics(pubSubTopics []string, specificPeers ...peer.ID) peer.IDSlice
	PeersByPubSubTopic(pubSubTopic string, specificPeers ...peer.ID) peer.IDSlice

	SetScore(peer.ID, float64) error
	Score(peer.ID) (float64, error)
}

// NewWakuPeerstore creates a new WakuPeerStore object
func NewWakuPeerstore(p peerstore.Peerstore) peerstore.Peerstore {
	return &WakuPeerstoreImpl{
		peerStore: p,
		connFailures: ConnectionFailures{
			failures: make(map[peer.ID]int),
		},
	}
}

// SetOrigin sets origin for a specific peer.
func (ps *WakuPeerstoreImpl) SetOrigin(p peer.ID, origin Origin) error {
	return ps.peerStore.Put(p, peerOrigin, origin)
}

// Origin fetches the origin for a specific peer.
func (ps *WakuPeerstoreImpl) Origin(p peer.ID) (Origin, error) {
	result, err := ps.peerStore.Get(p, peerOrigin)
	if err != nil {
		return Unknown, err
	}

	return result.(Origin), nil
}

// SetScore sets score for a specific peer.
func (ps *WakuPeerstoreImpl) SetScore(p peer.ID, score float64) error {
	return ps.peerStore.Put(p, peerScore, score)
}

// Score fetches the peerScore for a specific peer.
func (ps *WakuPeerstoreImpl) Score(p peer.ID) (float64, error) {
	result, err := ps.peerStore.Get(p, peerScore)
	if err != nil {
		return -1, err
	}

	return result.(float64), nil
}

// PeersByOrigin returns the list of peers for a specific origin
func (ps *WakuPeerstoreImpl) PeersByOrigin(expectedOrigin Origin) peer.IDSlice {
	var result peer.IDSlice
	for _, p := range ps.Peers() {
		actualOrigin, err := ps.Origin(p)
		if err == nil && actualOrigin == expectedOrigin {
			result = append(result, p)
		}
	}
	return result
}

// SetENR sets the ENR record a peer
func (ps *WakuPeerstoreImpl) SetENR(p peer.ID, enr *enode.Node) error {
	return ps.peerStore.Put(p, peerENR, enr)
}

// ENR fetches the ENR record for a peer
func (ps *WakuPeerstoreImpl) ENR(p peer.ID) (*enode.Node, error) {
	result, err := ps.peerStore.Get(p, peerENR)
	if err != nil {
		return nil, err
	}
	return result.(*enode.Node), nil
}

// AddConnFailure increments connectionFailures for a peer
func (ps *WakuPeerstoreImpl) AddConnFailure(pID peer.ID) {
	ps.connFailures.Lock()
	defer ps.connFailures.Unlock()
	ps.connFailures.failures[pID]++
}

// ResetConnFailures resets connectionFailures for a peer to 0
func (ps *WakuPeerstoreImpl) ResetConnFailures(pID peer.ID) {
	ps.connFailures.Lock()
	defer ps.connFailures.Unlock()
	ps.connFailures.failures[pID] = 0
}

// ConnFailures fetches connectionFailures for a peer
func (ps *WakuPeerstoreImpl) ConnFailures(pID peer.ID) int {
	ps.connFailures.RLock()
	defer ps.connFailures.RUnlock()
	return ps.connFailures.failures[pID]
}

// SetDirection sets connection direction for a specific peer.
func (ps *WakuPeerstoreImpl) SetDirection(p peer.ID, direction network.Direction) error {
	return ps.peerStore.Put(p, peerDirection, direction)
}

// Direction fetches the connection direction (Inbound or outBound) for a specific peer
func (ps *WakuPeerstoreImpl) Direction(p peer.ID) (network.Direction, error) {
	result, err := ps.peerStore.Get(p, peerDirection)
	if err != nil {
		return network.DirUnknown, err
	}

	return result.(network.Direction), nil
}

// AddPubSubTopic adds a new pubSubTopic for a peer
func (ps *WakuPeerstoreImpl) AddPubSubTopic(p peer.ID, topic string) error {
	existingTopics, err := ps.PubSubTopics(p)
	if err != nil {
		return err
	}

	if _, found := existingTopics[topic]; found {
		return nil
	}
	existingTopics[topic] = struct{}{}
	return ps.peerStore.Put(p, peerPubSubTopics, maps.Keys(existingTopics))
}

// RemovePubSubTopic removes a pubSubTopic from the peer
func (ps *WakuPeerstoreImpl) RemovePubSubTopic(p peer.ID, topic string) error {
	existingTopics, err := ps.PubSubTopics(p)
	if err != nil {
		return err
	}

	if len(existingTopics) == 0 {
		return nil
	}

	delete(existingTopics, topic)

	err = ps.SetPubSubTopics(p, maps.Keys(existingTopics))
	if err != nil {
		return err
	}
	return nil
}

// SetPubSubTopics sets pubSubTopics for a peer, it also overrides existing ones that were set previously..
func (ps *WakuPeerstoreImpl) SetPubSubTopics(p peer.ID, topics []string) error {
	return ps.peerStore.Put(p, peerPubSubTopics, topics)
}

// PubSubTopics fetches list of pubSubTopics for a peer
func (ps *WakuPeerstoreImpl) PubSubTopics(p peer.ID) (protocol.TopicSet, error) {
	result, err := ps.peerStore.Get(p, peerPubSubTopics)
	if err != nil {
		if errors.Is(err, peerstore.ErrNotFound) {
			return protocol.NewTopicSet(), nil
		} else {
			return nil, err
		}
	}
	return protocol.NewTopicSet((result.([]string))...), nil
}

// PeersByPubSubTopic Returns list of peers that support list of pubSubTopics
// If specifiPeers are listed, filtering is done from them otherwise from all peers in peerstore
func (ps *WakuPeerstoreImpl) PeersByPubSubTopics(pubSubTopics []string, specificPeers ...peer.ID) peer.IDSlice {
	if specificPeers == nil {
		specificPeers = ps.Peers()
	}
	var result peer.IDSlice
	for _, p := range specificPeers {
		peerMatch := true
		peerTopics, err := ps.PubSubTopics(p)
		if err == nil {
			for _, t := range pubSubTopics {
				if _, ok := peerTopics[t]; !ok {
					peerMatch = false
					break
				}
			}
			if peerMatch {
				result = append(result, p)
			}
		} //Note: skipping a peer in case of an error as there would be others available.
	}
	return result
}

// PeersByPubSubTopic Returns list of peers that support a single pubSubTopic
// If specifiPeers are listed, filtering is done from them otherwise from all peers in peerstore
func (ps *WakuPeerstoreImpl) PeersByPubSubTopic(pubSubTopic string, specificPeers ...peer.ID) peer.IDSlice {
	if specificPeers == nil {
		specificPeers = ps.Peers()
	}
	var result peer.IDSlice
	for _, p := range specificPeers {
		topics, err := ps.PubSubTopics(p)
		if err == nil {
			for topic := range topics {
				if topic == pubSubTopic {
					result = append(result, p)
				}
			}
		} //Note: skipping a peer in case of an error as there would be others available.
	}
	return result
}
