package types

import (
	"fmt"

	enstypes "github.com/status-im/status-go/eth-node/types/ens"
	"go.uber.org/zap"
)

// EnodeID is a unique identifier for each node.
type EnodeID [32]byte

// ID prints as a long hexadecimal number.
func (n EnodeID) String() string {
	return fmt.Sprintf("%x", n[:])
}

type Node interface {
	NewENSVerifier(logger *zap.Logger) enstypes.ENSVerifier
	GetWhisper(ctx interface{}) (Whisper, error)
	AddPeer(url string) error
	RemovePeer(url string) error
}
