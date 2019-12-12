package gethbridge

import (
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/enode"
	gethens "github.com/status-im/status-go/eth-node/bridge/geth/ens"
	"github.com/status-im/status-go/eth-node/types"
	enstypes "github.com/status-im/status-go/eth-node/types/ens"
	"github.com/status-im/status-go/whisper/v6"
	"go.uber.org/zap"
)

type gethNodeWrapper struct {
	stack *node.Node
}

func NewNodeBridge(stack *node.Node) types.Node {
	return &gethNodeWrapper{stack: stack}
}

func (w *gethNodeWrapper) NewENSVerifier(logger *zap.Logger) enstypes.ENSVerifier {
	return gethens.NewVerifier(logger)
}

func (w *gethNodeWrapper) GetWhisper(ctx interface{}) (types.Whisper, error) {
	var nativeWhisper *whisper.Whisper
	if ctx == nil || ctx == w {
		err := w.stack.Service(&nativeWhisper)
		if err != nil {
			return nil, err
		}
	} else {
		switch serviceProvider := ctx.(type) {
		case *node.ServiceContext:
			err := serviceProvider.Service(&nativeWhisper)
			if err != nil {
				return nil, err
			}
		}
	}
	if nativeWhisper == nil {
		panic("Whisper service is not available")
	}

	return NewGethWhisperWrapper(nativeWhisper), nil
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
