package gethnode

import (
	"go.uber.org/zap"

	enstypes "github.com/status-im/status-go/eth-node/types/ens"
	wakutypes "github.com/status-im/status-go/waku/types"
)

type Node interface {
	NewENSVerifier(logger *zap.Logger) enstypes.ENSVerifier
	GetWaku(ctx interface{}) (wakutypes.Waku, error)
	GetWakuV2(ctx interface{}) (wakutypes.Waku, error)
	AddPeer(url string) error
	RemovePeer(url string) error
	PeersCount() int
}
