package peers2

import (
	"context"

	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/status-im/status-go/peers"
)

type FoundPeersHandler interface {
	Handle(*discv5.Node) bool
}

type AcceptAllPeersHandler struct{}

func (h *AcceptAllPeersHandler) Handle(node *discv5.Node) bool {
	return true
}

type SkipSelfPeersHandler struct {
	self discover.NodeID
}

func (h *SkipSelfPeersHandler) Handle(node *discv5.Node) bool {
	return h.self != discover.NodeID(node.ID)
}

type VerifierFoundPeersHandler struct {
	verifier peers.Verifier
}

func (h *VerifierFoundPeersHandler) Handle(node *discv5.Node) bool {
	return h.verifier.VerifyNode(context.TODO(), discover.NodeID(node.ID))
}
