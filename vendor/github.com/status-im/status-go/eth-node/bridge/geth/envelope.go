package gethbridge

import (
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/whisper"
)

type gethEnvelopeWrapper struct {
	envelope *whisper.Envelope
}

// NewGethEnvelopeWrapper returns an object that wraps Geth's Envelope in a types interface
func NewGethEnvelopeWrapper(e *whisper.Envelope) types.Envelope {
	return &gethEnvelopeWrapper{
		envelope: e,
	}
}

// GetGethEnvelopeFrom retrieves the underlying whisper Envelope struct from a wrapped Envelope interface
func GetGethEnvelopeFrom(f types.Envelope) *whisper.Envelope {
	return f.(*gethEnvelopeWrapper).envelope
}

func (w *gethEnvelopeWrapper) Hash() types.Hash {
	return types.Hash(w.envelope.Hash())
}

func (w *gethEnvelopeWrapper) Bloom() []byte {
	return w.envelope.Bloom()
}
