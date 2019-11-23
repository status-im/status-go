package gethbridge

import (
	"github.com/status-im/status-go/eth-node/types"
	whisper "github.com/status-im/whisper/whisperv6"
)

// NewGethEnvelopeEventWrapper returns a types.EnvelopeEvent object that mimics Geth's EnvelopeEvent
func NewGethEnvelopeEventWrapper(envelopeEvent *whisper.EnvelopeEvent) *types.EnvelopeEvent {
	if envelopeEvent == nil {
		panic("envelopeEvent should not be nil")
	}

	wrappedData := envelopeEvent.Data
	switch data := envelopeEvent.Data.(type) {
	case []whisper.EnvelopeError:
		wrappedData := make([]types.EnvelopeError, len(data))
		for index, envError := range data {
			wrappedData[index] = *NewGethEnvelopeErrorWrapper(&envError)
		}
	case *whisper.MailServerResponse:
		wrappedData = NewGethMailServerResponseWrapper(data)
	case whisper.SyncEventResponse:
		wrappedData = NewGethSyncEventResponseWrapper(data)
	}
	return &types.EnvelopeEvent{
		Event: types.EventType(envelopeEvent.Event),
		Hash:  types.Hash(envelopeEvent.Hash),
		Batch: types.Hash(envelopeEvent.Batch),
		Peer:  types.EnodeID(envelopeEvent.Peer),
		Data:  wrappedData,
	}
}
