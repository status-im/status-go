package bridge

import (
	"github.com/status-im/status-go/eth-node/types"
	wakutypes "github.com/status-im/status-go/waku/types"

	wakuv1common "github.com/status-im/status-go/wakuv1/common"
	wakuv2common "github.com/status-im/status-go/wakuv2/common"
)

// NewWakuEnvelopeErrorWrapper returns a wakutypes.EnvelopeError object that mimics Geth's EnvelopeError
func NewWakuEnvelopeErrorWrapper(envelopeError *wakuv1common.EnvelopeError) *wakutypes.EnvelopeError {
	if envelopeError == nil {
		panic("envelopeError should not be nil")
	}

	return &wakutypes.EnvelopeError{
		Hash:        types.Hash(envelopeError.Hash),
		Code:        mapGethErrorCode(envelopeError.Code),
		Description: envelopeError.Description,
	}
}

// NewWakuEnvelopeErrorWrapper returns a wakutypes.EnvelopeError object that mimics Geth's EnvelopeError
func NewWakuV2EnvelopeErrorWrapper(envelopeError *wakuv2common.EnvelopeError) *wakutypes.EnvelopeError {
	if envelopeError == nil {
		panic("envelopeError should not be nil")
	}

	return &wakutypes.EnvelopeError{
		Hash:        types.Hash(envelopeError.Hash),
		Code:        mapGethErrorCode(envelopeError.Code),
		Description: envelopeError.Description,
	}
}

func mapGethErrorCode(code uint) uint {
	switch code {
	case wakuv1common.EnvelopeTimeNotSynced:
		return wakutypes.EnvelopeTimeNotSynced
	case wakuv1common.EnvelopeOtherError:
		return wakutypes.EnvelopeOtherError
	}
	return wakutypes.EnvelopeOtherError
}
