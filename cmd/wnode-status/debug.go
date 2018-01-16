package main

import (
	"expvar"
	"net/http"

	"github.com/ethereum/go-ethereum/p2p"

	"github.com/status-im/status-go/geth/api"
	"github.com/status-im/status-go/geth/log"
)

type metrics struct {
	backend *api.StatusBackend
}

func newMetrics(b *api.StatusBackend) *metrics {
	return &metrics{b}
}

func (m *metrics) server() *p2p.Server {
	node, err := m.backend.NodeManager().Node()
	if err != nil {
		log.Error("Failed to get a node", "err", err.Error())
		return nil
	}

	server := node.Server()
	if server == nil {
		log.Error("Failed to get a server")
		return nil
	}

	return server
}

func (m *metrics) nodeInfo() interface{} {
	if server := m.server(); server != nil {
		return server.NodeInfo()
	}

	return nil
}

func (m *metrics) peersIPs() interface{} {
	server := m.server()
	if server == nil {
		return nil
	}

	var ret []string
	for _, peer := range server.PeersInfo() {
		ret = append(ret, peer.Network.RemoteAddress)
	}
	return ret
}

func startDebugServer(addr string, m *metrics) error {
	expvar.Publish("node_info", expvar.Func(m.nodeInfo))
	expvar.Publish("peers_ips", expvar.Func(m.peersIPs))

	return http.ListenAndServe(addr, nil)
}
