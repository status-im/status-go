package peerstore

import (
	"context"
	cr "crypto/rand"
	"fmt"
	"testing"

	"github.com/mr-tron/base58/base58"
	ma "github.com/multiformats/go-multiaddr"
	mh "github.com/multiformats/go-multihash"
)

type peerpair struct {
	ID   string
	Addr ma.Multiaddr
}

func randomPeer(b *testing.B) *peerpair {
	buf := make([]byte, 50)

	for {
		n, err := cr.Read(buf)
		if err != nil {
			b.Fatal(err)
		}
		if n > 0 {
			break
		}
	}

	id, err := mh.Encode(buf, mh.SHA2_256)
	if err != nil {
		b.Fatal(err)
	}
	b58ID := base58.Encode(id)

	addr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/6666/ipfs/%s", b58ID))
	if err != nil {
		b.Fatal(err)
	}

	return &peerpair{b58ID, addr}
}

func addressProducer(ctx context.Context, b *testing.B, addrs chan *peerpair) {
	defer close(addrs)
	for {
		p := randomPeer(b)
		select {
		case addrs <- p:
		case <-ctx.Done():
			return
		}
	}
}
