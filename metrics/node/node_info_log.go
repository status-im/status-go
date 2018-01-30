// +build !metrics metrics,prometheus

package node

import (
	"github.com/ethereum/go-ethereum/node"
	"github.com/status-im/status-go/geth/log"
)

func updateNodeInfo(node *node.Node) {
	server := node.Server()
	if server == nil {
		log.Warn("Failed to get a server")
		return
	}

	log.Debug("Metrics node_info", "id", server.NodeInfo().ID)
}

func updateNodePeers(node *node.Node) {
	server := node.Server()
	if server == nil {
		log.Warn("Failed to get a server")
		return
	}

	log.Debug("Metrics node_peers", "remote_addresses", getNodePeerRemoteAddresses(server))
}
