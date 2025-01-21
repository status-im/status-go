package gethbridge

import (
	"errors"

	"go.uber.org/zap"

	"github.com/status-im/status-go/wakuv1"
	"github.com/status-im/status-go/wakuv2"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/enode"

	gethens "github.com/status-im/status-go/eth-node/bridge/geth/ens"
	gethnode "github.com/status-im/status-go/eth-node/node"
	enstypes "github.com/status-im/status-go/eth-node/types/ens"

	wakutypes "github.com/status-im/status-go/waku/types"
)

type gethNodeWrapper struct {
	stack *node.Node
	waku1 *wakuv1.Waku
	waku2 *wakuv2.Waku
}

func NewNodeBridge(stack *node.Node, waku1 *wakuv1.Waku, waku2 *wakuv2.Waku) gethnode.Node {
	return &gethNodeWrapper{stack: stack, waku1: waku1, waku2: waku2}
}

func (w *gethNodeWrapper) Poll() {
	// noop
}

func (w *gethNodeWrapper) NewENSVerifier(logger *zap.Logger) enstypes.ENSVerifier {
	return gethens.NewVerifier(logger)
}

func (w *gethNodeWrapper) SetWaku1(waku *wakuv1.Waku) {
	w.waku1 = waku
}

func (w *gethNodeWrapper) SetWaku2(waku *wakuv2.Waku) {
	w.waku2 = waku
}

func (w *gethNodeWrapper) GetWaku(ctx interface{}) (wakutypes.Waku, error) {
	if w.waku1 == nil {
		return nil, errors.New("waku service is not available")
	}

	return w.waku1, nil
}

func (w *gethNodeWrapper) GetWakuV2(ctx interface{}) (wakutypes.Waku, error) {
	if w.waku2 == nil {
		return nil, errors.New("waku service is not available")
	}

	return w.waku2, nil
}

func (w *gethNodeWrapper) AddPeer(url string) error {
	parsedNode, err := enode.ParseV4(url)
	if err != nil {
		return err
	}

	w.stack.Server().AddPeer(parsedNode)

	return nil
}

func (w *gethNodeWrapper) RemovePeer(url string) error {
	parsedNode, err := enode.ParseV4(url)
	if err != nil {
		return err
	}

	w.stack.Server().RemovePeer(parsedNode)

	return nil
}

func (w *gethNodeWrapper) PeersCount() int {
	if w.stack.Server() == nil {
		return 0
	}
	return len(w.stack.Server().Peers())
}
