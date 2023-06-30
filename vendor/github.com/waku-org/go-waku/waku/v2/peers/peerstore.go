package peers

import (
	"sync"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
)

type Origin int64

const (
	Unknown Origin = iota
	Discv5
	Static
	PeerExchange
	DnsDiscovery
	Rendezvous
)

const peerOrigin = "origin"
const peerENR = "enr"

type ConnectionFailures struct {
	sync.RWMutex
	failures map[peer.ID]int
}

type WakuPeerstoreImpl struct {
	peerStore    peerstore.Peerstore
	connFailures ConnectionFailures
}

type WakuPeerstore interface {
	SetOrigin(p peer.ID, origin Origin) error
	Origin(p peer.ID, origin Origin) (Origin, error)
	PeersByOrigin(origin Origin) peer.IDSlice
	SetENR(p peer.ID, enr *enode.Node) error
	ENR(p peer.ID, origin Origin) (*enode.Node, error)
	AddConnFailure(p peer.AddrInfo)
	ResetConnFailures(p peer.AddrInfo)
	ConnFailures(p peer.AddrInfo) int
}

func NewWakuPeerstore(p peerstore.Peerstore) peerstore.Peerstore {
	return &WakuPeerstoreImpl{
		peerStore: p,
		connFailures: ConnectionFailures{
			failures: make(map[peer.ID]int),
		},
	}
}

func (ps *WakuPeerstoreImpl) SetOrigin(p peer.ID, origin Origin) error {
	return ps.peerStore.Put(p, peerOrigin, origin)
}

func (ps *WakuPeerstoreImpl) Origin(p peer.ID, origin Origin) (Origin, error) {
	result, err := ps.peerStore.Get(p, peerOrigin)
	if err != nil {
		return Unknown, err
	}

	return result.(Origin), nil
}

func (ps *WakuPeerstoreImpl) PeersByOrigin(origin Origin) peer.IDSlice {
	var result peer.IDSlice
	for _, p := range ps.Peers() {
		_, err := ps.Origin(p, origin)
		if err == nil {
			result = append(result, p)
		}
	}
	return result
}

func (ps *WakuPeerstoreImpl) SetENR(p peer.ID, enr *enode.Node) error {
	return ps.peerStore.Put(p, peerENR, enr)
}

func (ps *WakuPeerstoreImpl) ENR(p peer.ID, origin Origin) (*enode.Node, error) {
	result, err := ps.peerStore.Get(p, peerENR)
	if err != nil {
		return nil, err
	}
	return result.(*enode.Node), nil
}

func (ps *WakuPeerstoreImpl) AddConnFailure(p peer.AddrInfo) {
	ps.connFailures.Lock()
	defer ps.connFailures.Unlock()
	ps.connFailures.failures[p.ID]++
}

func (ps *WakuPeerstoreImpl) ResetConnFailures(p peer.AddrInfo) {
	ps.connFailures.Lock()
	defer ps.connFailures.Unlock()
	ps.connFailures.failures[p.ID] = 0
}

func (ps *WakuPeerstoreImpl) ConnFailures(p peer.AddrInfo) int {
	ps.connFailures.RLock()
	defer ps.connFailures.RUnlock()
	return ps.connFailures.failures[p.ID]
}
