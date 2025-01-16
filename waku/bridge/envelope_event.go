package bridge

import (
	"github.com/status-im/status-go/eth-node/types"
	wakutypes "github.com/status-im/status-go/waku/types"
	"github.com/status-im/status-go/wakuv1"

	wakuv1common "github.com/status-im/status-go/wakuv1/common"
	wakuv2common "github.com/status-im/status-go/wakuv2/common"
)

// NewWakuEnvelopeEventWrapper returns a wakutypes.EnvelopeEvent object that mimics Geth's EnvelopeEvent
func NewWakuEnvelopeEventWrapper(envelopeEvent *wakuv1common.EnvelopeEvent) *wakutypes.EnvelopeEvent {
	if envelopeEvent == nil {
		panic("envelopeEvent should not be nil")
	}

	wrappedData := envelopeEvent.Data
	switch data := envelopeEvent.Data.(type) {
	case []wakuv1common.EnvelopeError:
		wrappedData := make([]wakutypes.EnvelopeError, len(data))
		for index := range data {
			wrappedData[index] = *NewWakuEnvelopeErrorWrapper(&data[index])
		}
	case *wakuv1.MailServerResponse:
		wrappedData = NewWakuMailServerResponseWrapper(data)
	}
	return &wakutypes.EnvelopeEvent{
		Event: wakutypes.EventType(envelopeEvent.Event),
		Hash:  types.Hash(envelopeEvent.Hash),
		Batch: types.Hash(envelopeEvent.Batch),
		Peer:  types.EnodeID(envelopeEvent.Peer),
		Data:  wrappedData,
	}
}

// NewWakuV2EnvelopeEventWrapper returns a wakutypes.EnvelopeEvent object that mimics Geth's EnvelopeEvent
func NewWakuV2EnvelopeEventWrapper(envelopeEvent *wakuv2common.EnvelopeEvent) *wakutypes.EnvelopeEvent {
	if envelopeEvent == nil {
		panic("envelopeEvent should not be nil")
	}

	wrappedData := envelopeEvent.Data
	switch data := envelopeEvent.Data.(type) {
	case []wakuv2common.EnvelopeError:
		wrappedData := make([]wakutypes.EnvelopeError, len(data))
		for index := range data {
			wrappedData[index] = *NewWakuV2EnvelopeErrorWrapper(&data[index])
		}
	}
	return &wakutypes.EnvelopeEvent{
		Event: wakutypes.EventType(envelopeEvent.Event),
		Hash:  types.Hash(envelopeEvent.Hash),
		Batch: types.Hash(envelopeEvent.Batch),
		Peer:  types.EnodeID(envelopeEvent.Peer),
		Data:  wrappedData,
	}
}
