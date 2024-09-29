package node

import (
	"fmt"

	"github.com/libp2p/go-libp2p/core/metrics"
	"github.com/libp2p/go-libp2p/p2p/metricshelper"
	"github.com/prometheus/client_golang/prometheus"
)

var gitVersion = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "waku_version",
		Help: "The go-waku version",
	},
	[]string{"git_version"},
)

var peerDials = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "waku_peers_dials",
		Help: "Number of peer dials",
	})

var connectedPeers = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "waku_connected_peers",
		Help: "Number of connected peers",
	})

var peerStoreSize = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "waku_peer_store_size",
		Help: "Size of Peer Store",
	})

var bandwidthTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "libp2p_network_bytes_total",
		Help: "Bandwidth usage total",
	},
	[]string{"direction"},
)

var collectors = []prometheus.Collector{
	gitVersion,
	peerDials,
	connectedPeers,
	peerStoreSize,
	bandwidthTotal,
}

// Metrics exposes the functions required to update prometheus metrics for the waku node
type Metrics interface {
	RecordVersion(version string, commit string)
	RecordDial()
	RecordPeerConnected()
	RecordPeerDisconnected()
	SetPeerStoreSize(int)
	RecordBandwidth(metrics.Stats)
}

type metricsImpl struct {
	reg prometheus.Registerer
}

func newMetrics(reg prometheus.Registerer) Metrics {
	metricshelper.RegisterCollectors(reg, collectors...)
	return &metricsImpl{
		reg: reg,
	}
}

// RecordVersion registers a metric with the current version and commit of go-waku
func (m *metricsImpl) RecordVersion(version string, commit string) {
	v := fmt.Sprintf("%s-%s", version, commit)
	gitVersion.WithLabelValues(v).Inc()
}

// RecordDial increases the counter for the number of dials
func (m *metricsImpl) RecordDial() {
	peerDials.Inc()
}

// RecordPeerConnected increases the metrics for the number of connected peers
func (m *metricsImpl) RecordPeerConnected() {
	connectedPeers.Inc()
}

// RecordPeerDisconnected decreases the metrics for the number of connected peers
func (m *metricsImpl) RecordPeerDisconnected() {
	connectedPeers.Dec()
}

func (m *metricsImpl) SetPeerStoreSize(size int) {
	peerStoreSize.Set(float64(size))
}

func (m *metricsImpl) RecordBandwidth(stats metrics.Stats) {
	bandwidthTotal.WithLabelValues("in").Add(float64(stats.TotalIn))
	bandwidthTotal.WithLabelValues("out").Add(float64(stats.TotalOut))

}
