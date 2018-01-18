// +build metrics,!prometheus

package node

import (
	"encoding/json"
	"expvar"

	"github.com/ethereum/go-ethereum/node"
	"github.com/status-im/status-go/geth/log"
)

var (
	nodeInfo  = expvar.NewString("node_info")
	nodePeers = expvar.NewString("node_peers")
)

func updateNodeInfo(node *node.Node) {
	server := node.Server()
	if server == nil {
		log.Warn("Failed to get a server")
		return
	}

	nodeInfo.Set(server.NodeInfo().ID)
}

func updateNodePeers(node *node.Node) {
	server := node.Server()
	if server == nil {
		log.Warn("Failed to get a server")
		return
	}

	val, _ := json.Marshal(getNodePeerRemoteAddresses(server))
	nodePeers.Set(string(val))
}
