package verifier

import (
	"context"

	"github.com/ethereum/go-ethereum/p2p/discover"
)

// LocalVerifier verifies nodes based on a provided local list.
type LocalVerifier struct {
	KnownPeers map[discover.NodeID]bool
}

// NewLocalVerifier returns a new LocalVerifier instance.
func NewLocalVerifier(peers []discover.NodeID) *LocalVerifier {
	knownPeers := make(map[discover.NodeID]bool)
	for _, peer := range peers {
		knownPeers[peer] = true
	}

	return &LocalVerifier{KnownPeers: knownPeers}
}

// VerifyNode checks if a given node is trusted using a local list.
func (v *LocalVerifier) VerifyNode(_ context.Context, nodeID discover.NodeID) bool {
	if res, ok := v.KnownPeers[nodeID]; ok {
		return res
	}
	return false
}
