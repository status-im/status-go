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
		log.Warn("Failed to get a node", "err", err.Error())
		return nil
	}

	server := node.Server()
	if server == nil {
		log.Warn("Failed to get a server")
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

func (m *metrics) peersInfo() interface{} {
	if server := m.server(); server != nil {
		return server.PeersInfo()
	}

	return nil
}

func startDebugServer(addr string, m *metrics) error {
	expvar.Publish("node_info", expvar.Func(m.nodeInfo))
	expvar.Publish("peers_info", expvar.Func(m.peersInfo))

	return http.ListenAndServe(addr, nil)
}
