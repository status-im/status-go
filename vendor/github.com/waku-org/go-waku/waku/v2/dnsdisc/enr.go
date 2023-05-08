package dnsdisc

import (
	"context"

	"github.com/ethereum/go-ethereum/p2p/dnsdisc"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/metrics"
	wenr "github.com/waku-org/go-waku/waku/v2/protocol/enr"

	ma "github.com/multiformats/go-multiaddr"
)

type dnsDiscoveryParameters struct {
	nameserver string
}

type DnsDiscoveryOption func(*dnsDiscoveryParameters)

// WithNameserver is a DnsDiscoveryOption that configures the nameserver to use
func WithNameserver(nameserver string) DnsDiscoveryOption {
	return func(params *dnsDiscoveryParameters) {
		params.nameserver = nameserver
	}
}

type DiscoveredNode struct {
	PeerID    peer.ID
	Addresses []ma.Multiaddr
	ENR       *enode.Node
}

// RetrieveNodes returns a list of multiaddress given a url to a DNS discoverable ENR tree
func RetrieveNodes(ctx context.Context, url string, opts ...DnsDiscoveryOption) ([]DiscoveredNode, error) {
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
		metrics.RecordDnsDiscoveryError(ctx, "tree_sync_failure")
		return nil, err
	}

	for _, node := range tree.Nodes() {
		peerID, m, err := wenr.Multiaddress(node)
		if err != nil {
			metrics.RecordDnsDiscoveryError(ctx, "peer_info_failure")
			return nil, err
		}

		d := DiscoveredNode{
			PeerID:    peerID,
			Addresses: m,
		}

		if hasUDP(node) {
			d.ENR = node
		}

		discoveredNodes = append(discoveredNodes, d)
	}

	return discoveredNodes, nil
}

func hasUDP(node *enode.Node) bool {
	enrUDP := new(enr.UDP)
	if err := node.Record().Load(enr.WithEntry(enrUDP.ENRKey(), enrUDP)); err != nil {
		return false
	}
	return true
}
