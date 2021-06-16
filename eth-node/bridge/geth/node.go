package gethbridge

import (
	"errors"

	"go.uber.org/zap"

	"github.com/status-im/status-go/waku"
	"github.com/status-im/status-go/wakuv2"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/enode"

	gethens "github.com/status-im/status-go/eth-node/bridge/geth/ens"
	"github.com/status-im/status-go/eth-node/types"
	enstypes "github.com/status-im/status-go/eth-node/types/ens"
)

type gethNodeWrapper struct {
	stack *node.Node
}

func NewNodeBridge(stack *node.Node) types.Node {
	return &gethNodeWrapper{stack: stack}
}

func (w *gethNodeWrapper) Poll() {
	// noop
}

func (w *gethNodeWrapper) NewENSVerifier(logger *zap.Logger) enstypes.ENSVerifier {
	return gethens.NewVerifier(logger)
}

func (w *gethNodeWrapper) GetWaku(ctx interface{}) (types.Waku, error) {
	var nativeWaku *waku.Waku
	if ctx == nil || ctx == w {
		err := w.stack.Service(&nativeWaku)
		if err != nil {
			return nil, err
		}
	} else {
		switch serviceProvider := ctx.(type) {
		case *node.ServiceContext:
			err := serviceProvider.Service(&nativeWaku)
			if err != nil {
				return nil, err
			}
		}
	}
	if nativeWaku == nil {
		return nil, errors.New("waku service is not available")
	}

	return NewGethWakuWrapper(nativeWaku), nil
}

func (w *gethNodeWrapper) GetWakuV2(ctx interface{}) (types.Waku, error) {
	var nativeWaku *wakuv2.Waku
	if ctx == nil || ctx == w {
		err := w.stack.Service(&nativeWaku)
		if err != nil {
			return nil, err
		}
	} else {
		switch serviceProvider := ctx.(type) {
		case *node.ServiceContext:
			err := serviceProvider.Service(&nativeWaku)
			if err != nil {
				return nil, err
			}
		}
	}
	if nativeWaku == nil {
		return nil, errors.New("waku service is not available")
	}

	return NewGethWakuV2Wrapper(nativeWaku), nil
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
