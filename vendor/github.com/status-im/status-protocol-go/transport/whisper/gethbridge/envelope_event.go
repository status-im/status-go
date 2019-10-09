package gethbridge

import (
	whispertypes "github.com/status-im/status-protocol-go/transport/whisper/types"
	statusproto "github.com/status-im/status-protocol-go/types"
	whisper "github.com/status-im/whisper/whisperv6"
)

// NewGethEnvelopeEventWrapper returns a whispertypes.EnvelopeEvent object that mimics Geth's EnvelopeEvent
func NewGethEnvelopeEventWrapper(envelopeEvent *whisper.EnvelopeEvent) *whispertypes.EnvelopeEvent {
	if envelopeEvent == nil {
		panic("envelopeEvent should not be nil")
	}

	wrappedData := envelopeEvent.Data
	switch data := envelopeEvent.Data.(type) {
	case []whisper.EnvelopeError:
		wrappedData := make([]whispertypes.EnvelopeError, len(data))
		for index, envError := range data {
			wrappedData[index] = *NewGethEnvelopeErrorWrapper(&envError)
		}
	case *whisper.MailServerResponse:
		wrappedData = NewGethMailServerResponseWrapper(data)
	case whisper.SyncEventResponse:
		wrappedData = NewGethSyncEventResponseWrapper(data)
	}
	return &whispertypes.EnvelopeEvent{
		Event: whispertypes.EventType(envelopeEvent.Event),
		Hash:  statusproto.Hash(envelopeEvent.Hash),
		Batch: statusproto.Hash(envelopeEvent.Batch),
		Peer:  whispertypes.EnodeID(envelopeEvent.Peer),
		Data:  wrappedData,
	}
}
