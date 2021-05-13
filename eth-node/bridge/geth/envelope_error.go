package gethbridge

import (
	"github.com/status-im/status-go/eth-node/types"
	waku "github.com/status-im/status-go/waku/common"
)

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
	case waku.EnvelopeTimeNotSynced:
		return types.EnvelopeTimeNotSynced
	case waku.EnvelopeOtherError:
		return types.EnvelopeOtherError
	}
	return types.EnvelopeOtherError
}
