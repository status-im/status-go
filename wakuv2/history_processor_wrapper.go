package wakuv2

import (
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/status-im/status-go/wakuv2/common"
	"github.com/waku-org/go-waku/waku/v2/api/history"
	"github.com/waku-org/go-waku/waku/v2/protocol"
)

type HistoryProcessorWrapper struct {
	waku *Waku
}

func NewHistoryProcessorWrapper(waku *Waku) history.HistoryProcessor {
	return &HistoryProcessorWrapper{waku}
}

func (hr *HistoryProcessorWrapper) OnEnvelope(env *protocol.Envelope, processEnvelopes bool) error {
	return hr.waku.OnNewEnvelopes(env, common.StoreMessageType, processEnvelopes)
}

func (hr *HistoryProcessorWrapper) OnRequestFailed(requestID []byte, peerID peer.ID, err error) {
	hr.waku.onHistoricMessagesRequestFailed(requestID, peerID, err)
}
