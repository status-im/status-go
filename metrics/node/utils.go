package node

import "github.com/ethereum/go-ethereum/p2p"

func getNodePeerRemoteAddresses(server *p2p.Server) []string {
	var ret []string
	for _, peer := range server.PeersInfo() {
		ret = append(ret, peer.Network.RemoteAddress)
	}
	return ret
}
