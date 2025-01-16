package ext

import (
	"go.uber.org/zap"

	enstypes "github.com/status-im/status-go/eth-node/types/ens"
	wakutypes "github.com/status-im/status-go/waku/types"
)

type TestNodeWrapper struct {
	waku wakutypes.Waku
}

func NewTestNodeWrapper(waku wakutypes.Waku) *TestNodeWrapper {
	return &TestNodeWrapper{waku: waku}
}

func (w *TestNodeWrapper) NewENSVerifier(_ *zap.Logger) enstypes.ENSVerifier {
	panic("not implemented")
}

func (w *TestNodeWrapper) GetWaku(_ interface{}) (wakutypes.Waku, error) {
	return w.waku, nil
}

func (w *TestNodeWrapper) GetWakuV2(_ interface{}) (wakutypes.Waku, error) {
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
