package gethbridge

import (
	whispertypes "github.com/status-im/status-go/protocol/transport/whisper/types"
	statusproto "github.com/status-im/status-go/protocol/types"
	whisper "github.com/status-im/whisper/whisperv6"
)

// NewGethEnvelopeErrorWrapper returns a whispertypes.EnvelopeError object that mimics Geth's EnvelopeError
func NewGethEnvelopeErrorWrapper(envelopeError *whisper.EnvelopeError) *whispertypes.EnvelopeError {
	if envelopeError == nil {
		panic("envelopeError should not be nil")
	}

	return &whispertypes.EnvelopeError{
		Hash:        statusproto.Hash(envelopeError.Hash),
		Code:        mapGethErrorCode(envelopeError.Code),
		Description: envelopeError.Description,
	}
}

func mapGethErrorCode(code uint) uint {
	switch code {
	case whisper.EnvelopeTimeNotSynced:
		return whispertypes.EnvelopeTimeNotSynced
	case whisper.EnvelopeOtherError:
		return whispertypes.EnvelopeOtherError
	}
	return whispertypes.EnvelopeOtherError
}
