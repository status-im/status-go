package ext

import (
	"github.com/status-im/status-go/eth-node/types"
	enstypes "github.com/status-im/status-go/eth-node/types/ens"
	"go.uber.org/zap"
)

type TestNodeWrapper struct {
	whisper types.Whisper
	waku    types.Waku
}

func NewTestNodeWrapper(whisper types.Whisper, waku types.Waku) *TestNodeWrapper {
	return &TestNodeWrapper{whisper: whisper, waku: waku}
}

func (w *TestNodeWrapper) NewENSVerifier(_ *zap.Logger) enstypes.ENSVerifier {
	panic("not implemented")
}

func (w *TestNodeWrapper) GetWhisper(_ interface{}) (types.Whisper, error) {
	return w.whisper, nil
}

func (w *TestNodeWrapper) GetWaku(_ interface{}) (types.Waku, error) {
	return w.waku, nil
}

func (w *TestNodeWrapper) AddPeer(url string) error {
	panic("not implemented")
}

func (w *TestNodeWrapper) RemovePeer(url string) error {
	panic("not implemented")
}
