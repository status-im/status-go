package multiaddr

import "github.com/multiformats/go-multiaddr"

func MustDecode(multiaddrsStr string) *multiaddr.Multiaddr {
	maddr, err := multiaddr.NewMultiaddr(multiaddrsStr)
	if err != nil || maddr == nil {
		panic("could not decode multiaddr: " + multiaddrsStr)
	}
	return &maddr
}
