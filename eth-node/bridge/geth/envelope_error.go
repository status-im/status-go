package gethbridge

import (
	"github.com/status-im/status-go/eth-node/types"
	waku "github.com/status-im/status-go/waku/common"
	"github.com/status-im/status-go/whisper/v6"
)

// NewWhisperEnvelopeErrorWrapper returns a types.EnvelopeError object that mimics Geth's EnvelopeError
func NewWhisperEnvelopeErrorWrapper(envelopeError *whisper.EnvelopeError) *types.EnvelopeError {
	if envelopeError == nil {
		panic("envelopeError should not be nil")
	}

	return &types.EnvelopeError{
		Hash:        types.Hash(envelopeError.Hash),
		Code:        mapGethErrorCode(envelopeError.Code),
		Description: envelopeError.Description,
	}
}

// NewWakuEnvelopeErrorWrapper returns a types.EnvelopeError object that mimics Geth's EnvelopeError
func NewWakuEnvelopeErrorWrapper(envelopeError *waku.EnvelopeError) *types.EnvelopeError {
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
	case waku.EnvelopeTimeNotSynced:
		return types.EnvelopeTimeNotSynced
	case whisper.EnvelopeOtherError:
	case waku.EnvelopeOtherError:
		return types.EnvelopeOtherError
	}
	return types.EnvelopeOtherError
}
