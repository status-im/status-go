package dnsdisc

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/dnsdisc"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/stretchr/testify/require"
)

type mapResolver map[string]string

func (mr mapResolver) LookupTXT(ctx context.Context, name string) ([]string, error) {
	if record, ok := mr[name]; ok {
		return []string{record}, nil
	}
	return nil, errors.New("not found")
}

var signingKeyForTesting, _ = crypto.ToECDSA(hexutil.MustDecode("0xdc599867fc513f8f5e2c2c9c489cde5e71362d1d9ec6e693e0de063236ed1240"))

func makeTestTree(domain string, nodes []*enode.Node, links []string) (*dnsdisc.Tree, string) {
	tree, err := dnsdisc.MakeTree(1, nodes, links)
	if err != nil {
		panic(err)
	}
	url, err := tree.Sign(signingKeyForTesting, domain)
	if err != nil {
		panic(err)
	}
	return tree, url
}

func parseNodes(rec []string) []*enode.Node {
	var ns []*enode.Node
	for _, r := range rec {
		var n enode.Node
		if err := n.UnmarshalText([]byte(r)); err != nil {
			panic(err)
		}
		ns = append(ns, &n)
	}
	return ns
}

func TestRetrieveNodes(t *testing.T) {
	nodes := []string{
		"enr:-Ji4QAa0VR5P27XvDEZzuFf1lnO6OGzm4hPhVtVYPFqlB-9vZnZtc-lzmEqY4stHFTIazRnSzwhlYne0UMIAmFMZ8o2GAYwawiLNgmlkgnY0gmlwhMCoAWSJc2VjcDI1NmsxoQLtnTLtFmyU8AFqO8Jw4X9zBfB6fWJxsMk9YpyrPeNPkoN0Y3CCw6qDdWRwgsm6hXdha3UyAQ",
		"enr:-Ji4QPr-1R0uv6QSYSwtsjG-ksFvW6zEWRlIzkJGmr9SAPjcWmU7xM-3njzP0ByLhP3xNBBxeF_V5baEjITy6RuPKtuGAYwawtZPgmlkgnY0gmlwhMCoAWSJc2VjcDI1NmsxoQJyiENqCiVwzkluXBexKPA4eeLZU_Q2v0f0gRen_xoQaoN0Y3CCxJ6DdWRwgt4uhXdha3UyAQ",
	}
	tree, url := makeTestTree("n", parseNodes(nodes), nil)
	resolver := mapResolver(tree.ToTXT("n"))

	var opts []DNSDiscoveryOption
	opts = append(opts, WithResolver(resolver))

	discoveredNodes, err := RetrieveNodes(context.Background(), url, opts...)

	require.NoError(t, err)
	require.Equal(t, len(discoveredNodes), 2)
}

func TestExclusiveOpts(t *testing.T) {
	var opts []DNSDiscoveryOption

	tree, url := makeTestTree("n", nil, nil)
	resolver := mapResolver(tree.ToTXT("n"))

	opts = append(opts, WithNameserver("1.1.1.1"), WithResolver(resolver))
	_, err := RetrieveNodes(context.Background(), url, opts...)
	require.Equal(t, err, ErrExclusiveOpts)

	opts = append(opts, WithResolver(resolver), WithNameserver("1.1.1.1"))
	_, err = RetrieveNodes(context.Background(), url, opts...)
	require.Equal(t, err, ErrExclusiveOpts)
}
