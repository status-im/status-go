package les

import (
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

type ulc struct {
	trustedKeys        map[string]struct{}
	minTrustedFraction int
}

func newULC(ulcConfig *eth.ULCConfig) *ulc {
	if ulcConfig == nil {
		return nil
	}

	m := make(map[string]struct{}, len(ulcConfig.TrustedServers))
	for _, id := range ulcConfig.TrustedServers {
		node, err := discover.ParseNode(id)
		if err != nil {
			continue
		}
		m[node.ID.String()] = struct{}{}
	}

	return &ulc{m, ulcConfig.MinTrustedFraction}
}

func (u *ulc) isTrusted(p discover.NodeID) bool {
	if u.trustedKeys == nil {
		return false
	}
	_, ok := u.trustedKeys[p.String()]
	return ok
}
