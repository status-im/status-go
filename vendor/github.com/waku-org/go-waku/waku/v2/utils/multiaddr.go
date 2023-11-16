package utils

import (
	"fmt"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

// EncapsulatePeerID takes a peer.ID and adds a p2p component to all multiaddresses it receives
func EncapsulatePeerID(peerID peer.ID, addrs ...multiaddr.Multiaddr) []multiaddr.Multiaddr {
	hostInfo, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", peerID.Pretty()))
	var result []multiaddr.Multiaddr
	for _, addr := range addrs {
		result = append(result, addr.Encapsulate(hostInfo))
	}
	return result
}
