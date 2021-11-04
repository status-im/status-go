package discovery

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/p2p/dnsdisc"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/libp2p/go-libp2p-core/peer"

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
		m, err := EnodeToMultiAddr(node)
		if err != nil {
			return nil, err
		}

		multiAddrs = append(multiAddrs, m)
	}

	return multiAddrs, nil
}

func EnodeToMultiAddr(node *enode.Node) (ma.Multiaddr, error) {
	peerID, err := peer.IDFromPublicKey(&ECDSAPublicKey{node.Pubkey()})
	if err != nil {
		return nil, err
	}

	return ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", node.IP(), node.TCP(), peerID))
}
