package gethbridge

import (
	whispertypes "github.com/status-im/status-go/protocol/transport/whisper/types"
	statusproto "github.com/status-im/status-go/protocol/types"
	whisper "github.com/status-im/whisper/whisperv6"
)

type gethEnvelopeWrapper struct {
	envelope *whisper.Envelope
}

// NewGethEnvelopeWrapper returns an object that wraps Geth's Envelope in a whispertypes interface
func NewGethEnvelopeWrapper(e *whisper.Envelope) whispertypes.Envelope {
	return &gethEnvelopeWrapper{
		envelope: e,
	}
}

// GetGethEnvelopeFrom retrieves the underlying whisper Envelope struct from a wrapped Envelope interface
func GetGethEnvelopeFrom(f whispertypes.Envelope) *whisper.Envelope {
	return f.(*gethEnvelopeWrapper).envelope
}

func (w *gethEnvelopeWrapper) Hash() statusproto.Hash {
	return statusproto.Hash(w.envelope.Hash())
}

func (w *gethEnvelopeWrapper) Bloom() []byte {
	return w.envelope.Bloom()
}
