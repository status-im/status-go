package wakuv2

import (
	"github.com/libp2p/go-libp2p/core/peer"

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

func (hr *HistoryProcessorWrapper) OnRequestFailed(requestID []byte, peerInfo peer.AddrInfo, err error) {
	hr.waku.onHistoricMessagesRequestFailed(requestID, peerInfo, err)
}
