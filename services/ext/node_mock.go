package ext

import (
	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/types"
	enstypes "github.com/status-im/status-go/eth-node/types/ens"
)

type TestNodeWrapper struct {
	waku types.Waku
}

func NewTestNodeWrapper(waku types.Waku) *TestNodeWrapper {
	return &TestNodeWrapper{waku: waku}
}

func (w *TestNodeWrapper) NewENSVerifier(_ *zap.Logger) enstypes.ENSVerifier {
	panic("not implemented")
}

func (w *TestNodeWrapper) GetWaku(_ interface{}) (types.Waku, error) {
	return w.waku, nil
}

func (w *TestNodeWrapper) GetWakuV2(_ interface{}) (types.Waku, error) {
	return w.waku, nil
}

func (w *TestNodeWrapper) PeersCount() int {
	return 1
}

func (w *TestNodeWrapper) AddPeer(url string) error {
	panic("not implemented")
}

func (w *TestNodeWrapper) RemovePeer(url string) error {
	panic("not implemented")
}
