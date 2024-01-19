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

func MultiAddrSet(addr ...multiaddr.Multiaddr) map[multiaddr.Multiaddr]struct{} {
	r := make(map[multiaddr.Multiaddr]struct{})
	for _, a := range addr {
		r[a] = struct{}{}
	}
	return r
}
