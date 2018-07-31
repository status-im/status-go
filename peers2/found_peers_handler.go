package peers2

import (
	"context"

	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/status-im/status-go/peers"
)

// FoundPeersHandler verifies if a found node should be processed further.
type FoundPeersHandler interface {
	Handle(*discv5.Node) bool
}

// AcceptAllPeersHandler accepts all peers.
type AcceptAllPeersHandler struct{}

// Handle returns always true.
func (h AcceptAllPeersHandler) Handle(node *discv5.Node) bool {
	return true
}

// SkipSelfPeersHandler accepts all nodes except for the itself.
type SkipSelfPeersHandler struct {
	self discover.NodeID
}

// Handle returns true if a found node it not itself.
func (h SkipSelfPeersHandler) Handle(node *discv5.Node) bool {
	return h.self != discover.NodeID(node.ID)
}

// VerifierFoundPeersHandler verifies a peer using a verifier object.
type VerifierFoundPeersHandler struct {
	verifier peers.Verifier
}

// Handle returns true only if the Verifier confirms the node.
func (h VerifierFoundPeersHandler) Handle(node *discv5.Node) bool {
	return h.verifier.VerifyNode(context.TODO(), discover.NodeID(node.ID))
}
