package utils

import (
	"fmt"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

// EncapsulatePeerID takes a peer.ID and adds a p2p component to all multiaddresses it receives
func EncapsulatePeerID(peerID peer.ID, addrs ...multiaddr.Multiaddr) []multiaddr.Multiaddr {
	hostInfo, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", peerID.String()))
	var result []multiaddr.Multiaddr
	for _, addr := range addrs {
		result = append(result, addr.Encapsulate(hostInfo))
	}
	return result
}

func MultiAddrSetEquals(m1 map[string]multiaddr.Multiaddr, m2 map[string]multiaddr.Multiaddr) bool {
	if len(m1) != len(m2) {
		return false
	}
	for k1, v1 := range m1 {
		v2, ok := m2[k1]
		if !ok || !v1.Equal(v2) {
			return false
		}
	}
	return true
}

func MultiAddrSet(addr ...multiaddr.Multiaddr) map[string]multiaddr.Multiaddr {
	r := make(map[string]multiaddr.Multiaddr)
	for _, a := range addr {
		r[a.String()] = a
	}
	return r
}
