package node

import (
	"errors"
	"strings"

	prom "github.com/prometheus/client_golang/prometheus"
)

var (
	nodePeersGauge = prom.NewGaugeVec(prom.GaugeOpts{
		Name: "p2p_peers_count",
		Help: "Current numbers of peers split by name.",
	}, []string{"type", "version", "platform"})
	nodePeersAbsolute = prom.NewGauge(prom.GaugeOpts{
		Name: "p2p_peers_absolute",
		Help: "Absolute number of connected peers.",
	})
	nodeMaxPeersGauge = prom.NewGauge(prom.GaugeOpts{
		Name: "p2p_peers_max",
		Help: "Maximum number of peers that can connect.",
	})
)

func init() {
	prom.MustRegister(nodePeersGauge)
	prom.MustRegister(nodePeersAbsolute)
	prom.MustRegister(nodeMaxPeersGauge)
}

func labelsFromNodeName(name string) (prom.Labels, error) {
	tokens := strings.Split(name, "/")
	if len(tokens) == 4 {
		return prom.Labels{
			"type":     tokens[0],
			"version":  tokens[1],
			"platform": tokens[2],
		}, nil
	} else if len(tokens) == 3 {
		return prom.Labels{
			"type":     tokens[0],
			"version":  "unknown",
			"platform": tokens[1],
		}, nil
	} else {
		return nil, errors.New("wrong number of segments in name")
	}
}
