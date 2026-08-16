package wakuv2

import (
	"github.com/waku-org/go-waku/waku/v2/protocol"

	"github.com/status-im/status-go/pkg/messaging/waku/common"
)

type HistoryProcessorWrapper struct {
	waku *Waku
}

func NewHistoryProcessorWrapper(waku *Waku) *HistoryProcessorWrapper {
	return &HistoryProcessorWrapper{waku}
}

func (hr *HistoryProcessorWrapper) OnEnvelope(env *protocol.Envelope, processEnvelopes bool) error {
	return hr.waku.OnNewEnvelopes(env, common.StoreMessageType, processEnvelopes)
}
