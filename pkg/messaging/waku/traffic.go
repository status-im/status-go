package waku

import (
	"cmp"
	"slices"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/metrics"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"

	"github.com/status-im/status-go/pkg/messaging/waku/types"
)

// trafficCounter is libp2p's bandwidth counter plus the moment it started counting.
type trafficCounter struct {
	*metrics.BandwidthCounter

	mu    sync.Mutex
	since time.Time
}

func newTrafficCounter() *trafficCounter {
	return &trafficCounter{BandwidthCounter: metrics.NewBandwidthCounter(), since: time.Now()}
}

func (c *trafficCounter) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.BandwidthCounter.Reset()
	c.since = time.Now()
}

func (c *trafficCounter) Report() types.TrafficReport {
	c.mu.Lock()
	defer c.mu.Unlock()

	totals := c.GetBandwidthTotals()
	report := types.TrafficReport{
		Since:     c.since,
		Until:     time.Now(),
		Totals:    trafficOf(totals),
		Protocols: make([]types.ProtocolTraffic, 0),
		Peers:     make([]types.PeerTraffic, 0),
	}
	for id, stats := range c.GetBandwidthByProtocol() {
		report.Protocols = append(report.Protocols, types.ProtocolTraffic{Protocol: id, Traffic: trafficOf(stats)})
	}
	for id, stats := range c.GetBandwidthByPeer() {
		report.Peers = append(report.Peers, types.PeerTraffic{Peer: id, Traffic: trafficOf(stats)})
	}
	slices.SortFunc(report.Protocols, func(a, b types.ProtocolTraffic) int {
		return heaviestFirst(a.Traffic, b.Traffic, a.Protocol, b.Protocol)
	})
	slices.SortFunc(report.Peers, func(a, b types.PeerTraffic) int {
		return heaviestFirst(a.Traffic, b.Traffic, a.Peer, b.Peer)
	})
	return report
}

func trafficOf(stats metrics.Stats) types.Traffic {
	return types.Traffic{BytesSent: stats.TotalOut, BytesReceived: stats.TotalIn}
}

// heaviestFirst orders by bytes, then by name so equal rows keep their place.
func heaviestFirst[ID protocol.ID | peer.ID](a, b types.Traffic, aID, bID ID) int {
	if byBytes := cmp.Compare(b.BytesSent+b.BytesReceived, a.BytesSent+a.BytesReceived); byBytes != 0 {
		return byBytes
	}
	return cmp.Compare(aID, bID)
}

func (w *Waku) TrafficReport() types.TrafficReport {
	return w.traffic.Report()
}

func (w *Waku) ResetTrafficStats() {
	w.traffic.Reset()
}
