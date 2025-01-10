package gethbridge

import (
	"github.com/status-im/status-go/eth-node/types"
	wakuv1common "github.com/status-im/status-go/wakuv1/common"
	wakuv2common "github.com/status-im/status-go/wakuv2/common"
)

// NewWakuEnvelopeErrorWrapper returns a types.EnvelopeError object that mimics Geth's EnvelopeError
func NewWakuEnvelopeErrorWrapper(envelopeError *wakuv1common.EnvelopeError) *types.EnvelopeError {
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
func NewWakuV2EnvelopeErrorWrapper(envelopeError *wakuv2common.EnvelopeError) *types.EnvelopeError {
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
	case wakuv1common.EnvelopeTimeNotSynced:
		return types.EnvelopeTimeNotSynced
	case wakuv1common.EnvelopeOtherError:
		return types.EnvelopeOtherError
	}
	return types.EnvelopeOtherError
}
