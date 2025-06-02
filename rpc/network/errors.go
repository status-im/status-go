package network

import (
	"github.com/status-im/status-go/errors"
)

// Abbreviation `NET` for the error code stands for Networks
var (
	ErrNetworkNotDeactivatable    = &errors.ErrorResponse{Code: errors.ErrorCode("NET-001"), Details: "network is not deactivatable"}
	ErrActiveNetworksLimitReached = &errors.ErrorResponse{Code: errors.ErrorCode("NET-002"), Details: "maximum number of active networks reached"}
	ErrUnsupportedChainId         = &errors.ErrorResponse{Code: errors.ErrorCode("NET-003"), Details: "chainID is not supported"}
)
