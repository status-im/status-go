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

// AcceptAllPeers accepts all peers.
type AcceptAllPeers struct{}

// Handle returns always true.
func (h AcceptAllPeers) Handle(node *discv5.Node) bool {
	return true
}

// SkipSelfPeers accepts all nodes except for the itself.
type SkipSelfPeers struct {
	self discover.NodeID
}

// Handle returns true if a found node it not itself.
func (h SkipSelfPeers) Handle(node *discv5.Node) bool {
	return h.self != discover.NodeID(node.ID)
}

// VerifierFoundPeers verifies a peer using a verifier object.
type VerifierFoundPeers struct {
	verifier peers.Verifier
}

// Handle returns true only if the Verifier confirms the node.
func (h VerifierFoundPeers) Handle(node *discv5.Node) bool {
	return h.verifier.VerifyNode(context.TODO(), discover.NodeID(node.ID))
}
