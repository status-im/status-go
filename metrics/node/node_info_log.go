// +build !metrics

package node

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
)

func updateNodeInfo(node *node.Node) {
	server := node.Server()
	logger := log.New("package", "status-go/metrics/whisper")
	if server == nil {
		logger.Warn("Failed to get a server")
		return
	}

	logger.Debug("Metrics node_info", "id", server.NodeInfo().ID)
}

func updateNodePeers(node *node.Node) {
	server := node.Server()
	logger := log.New("package", "status-go/metrics/whisper")
	if server == nil {
		logger.Warn("Failed to get a server")
		return
	}

	logger.Debug("Metrics node_peers", "remote_addresses", getNodePeerRemoteAddresses(server))
}
