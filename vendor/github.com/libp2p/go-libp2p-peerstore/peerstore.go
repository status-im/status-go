package peerstore

import (
	"fmt"
	"io"

	peer "github.com/libp2p/go-libp2p-peer"
)

var _ Peerstore = (*peerstore)(nil)

type peerstore struct {
	Metrics

	KeyBook
	AddrBook
	ProtoBook
	PeerMetadata
}

// NewPeerstore creates a data structure that stores peer data, backed by the
// supplied implementations of KeyBook, AddrBook and PeerMetadata.
func NewPeerstore(kb KeyBook, ab AddrBook, pb ProtoBook, md PeerMetadata) Peerstore {
	return &peerstore{
		KeyBook:      kb,
		AddrBook:     ab,
		ProtoBook:    pb,
		PeerMetadata: md,
		Metrics:      NewMetrics(),
	}
}

func (ps *peerstore) Close() (err error) {
	var errs []error
	weakClose := func(name string, c interface{}) {
		if cl, ok := c.(io.Closer); ok {
			if err = cl.Close(); err != nil {
				errs = append(errs, fmt.Errorf("%s error: %s", name, err))
			}
		}
	}

	weakClose("keybook", ps.KeyBook)
	weakClose("addressbook", ps.AddrBook)
	weakClose("protobook", ps.ProtoBook)
	weakClose("peermetadata", ps.PeerMetadata)

	if len(errs) > 0 {
		return fmt.Errorf("failed while closing peerstore; err(s): %q", errs)
	}
	return nil
}

func (ps *peerstore) Peers() peer.IDSlice {
	set := map[peer.ID]struct{}{}
	for _, p := range ps.PeersWithKeys() {
		set[p] = struct{}{}
	}
	for _, p := range ps.PeersWithAddrs() {
		set[p] = struct{}{}
	}

	pps := make(peer.IDSlice, 0, len(set))
	for p := range set {
		pps = append(pps, p)
	}
	return pps
}

func (ps *peerstore) PeerInfo(p peer.ID) PeerInfo {
	return PeerInfo{
		ID:    p,
		Addrs: ps.AddrBook.Addrs(p),
	}
}

func PeerInfos(ps Peerstore, peers peer.IDSlice) []PeerInfo {
	pi := make([]PeerInfo, len(peers))
	for i, p := range peers {
		pi[i] = ps.PeerInfo(p)
	}
	return pi
}

func PeerInfoIDs(pis []PeerInfo) peer.IDSlice {
	ps := make(peer.IDSlice, len(pis))
	for i, pi := range pis {
		ps[i] = pi.ID
	}
	return ps
}
