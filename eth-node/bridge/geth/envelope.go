package gethbridge

import (
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/waku"
	"github.com/status-im/status-go/whisper/v6"
)

type gethEnvelopeWrapper struct {
	shhEnvelope  *whisper.Envelope
	wakuEnvelope *waku.Envelope
}

// NewWhisperEnvelopeWrapper returns an object that wraps Geth's Whisper Envelope in a types interface.
func NewWhisperEnvelopeWrapper(e *whisper.Envelope) types.Envelope {
	return &gethEnvelopeWrapper{
		shhEnvelope: e,
	}
}

// NewWakuEnvelopeWrapper returns an object that wraps Geth's Waku Envelope in a types interface.
func NewWakuEnvelopeWrapper(e *waku.Envelope) types.Envelope {
	return &gethEnvelopeWrapper{
		wakuEnvelope: e,
	}
}

// GetWhisperEnvelopeFrom retrieves the underlying Whisper Envelope struct from a wrapped Envelope interface.
func GetWhisperEnvelopeFrom(f types.Envelope) *whisper.Envelope {
	return f.(*gethEnvelopeWrapper).shhEnvelope
}

// GetWakuEnvelopeFrom retrieves the underlying Waku Envelope struct from a wrapped Envelope interface.
func GetWakuEnvelopeFrom(f types.Envelope) *waku.Envelope {
	return f.(*gethEnvelopeWrapper).wakuEnvelope
}

func (w *gethEnvelopeWrapper) Hash() types.Hash {
	switch {
	case w.shhEnvelope != nil:
		return types.Hash(w.shhEnvelope.Hash())
	case w.wakuEnvelope != nil:
		return types.Hash(w.wakuEnvelope.Hash())
	default:
		return types.Hash{}
	}
}

func (w *gethEnvelopeWrapper) Bloom() []byte {
	switch {
	case w.shhEnvelope != nil:
		return w.shhEnvelope.Bloom()
	case w.wakuEnvelope != nil:
		return w.wakuEnvelope.Bloom()
	default:
		return nil
	}
}

func (w *gethEnvelopeWrapper) PoW() float64 {
	switch {
	case w.shhEnvelope != nil:
		return w.shhEnvelope.PoW()
	case w.wakuEnvelope != nil:
		return w.wakuEnvelope.PoW()
	default:
		return 0
	}
}

func (w *gethEnvelopeWrapper) Expiry() uint32 {
	switch {
	case w.shhEnvelope != nil:
		return w.shhEnvelope.Expiry
	case w.wakuEnvelope != nil:
		return w.wakuEnvelope.Expiry
	default:
		return 0
	}
}

func (w *gethEnvelopeWrapper) TTL() uint32 {
	switch {
	case w.shhEnvelope != nil:
		return w.shhEnvelope.TTL
	case w.wakuEnvelope != nil:
		return w.wakuEnvelope.TTL
	default:
		return 0
	}
}

func (w *gethEnvelopeWrapper) Topic() types.TopicType {
	switch {
	case w.shhEnvelope != nil:
		return types.TopicType(w.shhEnvelope.Topic)
	case w.wakuEnvelope != nil:
		return types.TopicType(w.wakuEnvelope.Topic)
	default:
		return types.TopicType{}
	}
}

func (w *gethEnvelopeWrapper) Size() int {
	switch {
	case w.shhEnvelope != nil:
		return len(w.shhEnvelope.Data)
	case w.wakuEnvelope != nil:
		return len(w.wakuEnvelope.Data)
	default:
		return 0
	}
}
