package gethbridge

import (
	"github.com/status-im/status-go/eth-node/types"
	whisper "github.com/status-im/whisper/whisperv6"
)

// NewGethEnvelopeErrorWrapper returns a types.EnvelopeError object that mimics Geth's EnvelopeError
func NewGethEnvelopeErrorWrapper(envelopeError *whisper.EnvelopeError) *types.EnvelopeError {
	if envelopeError == nil {
		panic("envelopeError should not be nil")
	}

	return &types.EnvelopeError{
		Hash:        types.Hash(envelopeError.Hash),
		Code:        mapGethErrorCode(envelopeError.Code),
		Description: envelopeError.Description,
	}
}

func mapGethErrorCode(code uint) uint {
	switch code {
	case whisper.EnvelopeTimeNotSynced:
		return types.EnvelopeTimeNotSynced
	case whisper.EnvelopeOtherError:
		return types.EnvelopeOtherError
	}
	return types.EnvelopeOtherError
}
