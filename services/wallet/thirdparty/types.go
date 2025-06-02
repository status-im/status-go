package thirdparty

//go:generate mockgen -package=mock_thirdparty -source=types.go -destination=mock/types.go

import (
	"errors"

	w_common "github.com/status-im/status-go/services/wallet/common"
)

var (
	ErrChainIDNotSupported  = errors.New("chainID not supported")
	ErrEndpointNotSupported = errors.New("endpoint not supported")
)

const FetchNoLimit = 0
const FetchFromStartCursor = ""
const FetchFromAnyProvider = ""

type Provider interface {
	ID() string
	IsConnected() bool
}

type ChainProvider interface {
	Provider
	IsChainSupported(chainID w_common.ChainID) bool
}
