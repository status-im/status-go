package main

import (
	"expvar"
	"net/http"

	"github.com/status-im/status-go/geth/api"
	"github.com/status-im/status-go/geth/log"
)

type metrics struct {
	backend *api.StatusBackend
}

func newMetrics(b *api.StatusBackend) *metrics {
	return &metrics{b}
}

func (m *metrics) peersInfo() interface{} {
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

	return server.PeersInfo()
}

func startDebugServer(addr string, m *metrics) error {
	expvar.Publish("peers_info", expvar.Func(m.peersInfo))

	return http.ListenAndServe(addr, nil)
}
