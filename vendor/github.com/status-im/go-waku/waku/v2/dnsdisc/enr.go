package dnsdisc

import (
	"context"

	"github.com/ethereum/go-ethereum/p2p/dnsdisc"
	"github.com/status-im/go-waku/waku/v2/utils"

	ma "github.com/multiformats/go-multiaddr"
)

type DnsDiscoveryParameters struct {
	nameserver string
}

type DnsDiscoveryOption func(*DnsDiscoveryParameters)

// WithMultiaddress is a WakuNodeOption that configures libp2p to listen on a list of multiaddresses
func WithNameserver(nameserver string) DnsDiscoveryOption {
	return func(params *DnsDiscoveryParameters) {
		params.nameserver = nameserver
	}
}

// RetrieveNodes returns a list of multiaddress given a url to a DNS discoverable
// ENR tree
func RetrieveNodes(ctx context.Context, url string, opts ...DnsDiscoveryOption) ([]ma.Multiaddr, error) {
	var multiAddrs []ma.Multiaddr

	params := new(DnsDiscoveryParameters)
	for _, opt := range opts {
		opt(params)
	}

	client := dnsdisc.NewClient(dnsdisc.Config{
		Resolver: GetResolver(ctx, params.nameserver),
	})

	tree, err := client.SyncTree(url)
	if err != nil {
		return nil, err
	}

	for _, node := range tree.Nodes() {
		m, err := utils.EnodeToMultiAddr(node)
		if err != nil {
			return nil, err
		}

		multiAddrs = append(multiAddrs, m)
	}

	return multiAddrs, nil
}
