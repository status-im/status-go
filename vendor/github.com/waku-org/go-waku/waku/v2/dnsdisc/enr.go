package dnsdisc

import (
	"context"

	"github.com/ethereum/go-ethereum/p2p/dnsdisc"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/prometheus/client_golang/prometheus"
	wenr "github.com/waku-org/go-waku/waku/v2/protocol/enr"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
)

type dnsDiscoveryParameters struct {
	nameserver string
}

type DNSDiscoveryOption func(*dnsDiscoveryParameters)

// WithNameserver is a DnsDiscoveryOption that configures the nameserver to use
func WithNameserver(nameserver string) DNSDiscoveryOption {
	return func(params *dnsDiscoveryParameters) {
		params.nameserver = nameserver
	}
}

type DiscoveredNode struct {
	PeerID   peer.ID
	PeerInfo peer.AddrInfo
	ENR      *enode.Node
}

var metrics Metrics

// SetPrometheusRegisterer is used to setup a custom prometheus registerer for metrics
func SetPrometheusRegisterer(reg prometheus.Registerer, logger *zap.Logger) {
	metrics = newMetrics(reg)
}

func init() {
	SetPrometheusRegisterer(prometheus.DefaultRegisterer, utils.Logger())
}

// RetrieveNodes returns a list of multiaddress given a url to a DNS discoverable ENR tree
func RetrieveNodes(ctx context.Context, url string, opts ...DNSDiscoveryOption) ([]DiscoveredNode, error) {
	var discoveredNodes []DiscoveredNode

	params := new(dnsDiscoveryParameters)
	for _, opt := range opts {
		opt(params)
	}

	client := dnsdisc.NewClient(dnsdisc.Config{
		Resolver: GetResolver(ctx, params.nameserver),
	})

	tree, err := client.SyncTree(url)
	if err != nil {
		metrics.RecordError(treeSyncFailure)
		return nil, err
	}

	for _, node := range tree.Nodes() {
		peerID, m, err := wenr.Multiaddress(node)
		if err != nil {
			metrics.RecordError(peerInfoFailure)
			return nil, err
		}

		infoAddr, err := peer.AddrInfosFromP2pAddrs(m...)
		if err != nil {
			return nil, err
		}

		var info peer.AddrInfo
		for _, i := range infoAddr {
			if i.ID == peerID {
				info = i
				break
			}
		}

		d := DiscoveredNode{
			PeerID:   peerID,
			PeerInfo: info,
		}

		if hasUDP(node) {
			d.ENR = node
		}

		discoveredNodes = append(discoveredNodes, d)
	}

	metrics.RecordDiscoveredNodes(len(discoveredNodes))

	return discoveredNodes, nil
}

func hasUDP(node *enode.Node) bool {
	enrUDP := new(enr.UDP)
	if err := node.Record().Load(enr.WithEntry(enrUDP.ENRKey(), enrUDP)); err != nil {
		return false
	}
	return true
}
