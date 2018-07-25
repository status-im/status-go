package verifier

import (
	"context"

	"github.com/ethereum/go-ethereum/p2p/discover"
)

// LocalVerifier verifies nodes based on a provided local list.
type LocalVerifier struct {
	KnownPeers map[discover.NodeID]struct{}
}

// NewLocalVerifier returns a new LocalVerifier instance.
func NewLocalVerifier(peers []discover.NodeID) *LocalVerifier {
	knownPeers := make(map[discover.NodeID]struct{})
	for _, peer := range peers {
		knownPeers[peer] = struct{}{}
	}

	return &LocalVerifier{KnownPeers: knownPeers}
}

// VerifyNode checks if a given node is trusted using a local list.
func (v *LocalVerifier) VerifyNode(_ context.Context, nodeID discover.NodeID) bool {
	if _, ok := v.KnownPeers[nodeID]; ok {
		return true
	}
	return false
}
