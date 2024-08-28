package router

import (
	"github.com/status-im/status-go/errors"
)

// Abbreviation `WR` for the error code stands for Wallet Router
var (
	ErrNotEnoughTokenBalance   = &errors.ErrorResponse{Code: errors.ErrorCode("WR-001"), Details: "not enough token balance, token: %s, chainId: %d"}
	ErrNotEnoughNativeBalance  = &errors.ErrorResponse{Code: errors.ErrorCode("WR-002"), Details: "not enough native balance, token: %s, chainId: %d"}
	ErrNativeTokenNotFound     = &errors.ErrorResponse{Code: errors.ErrorCode("WR-003"), Details: "native token not found"}
	ErrTokenNotFound           = &errors.ErrorResponse{Code: errors.ErrorCode("WR-004"), Details: "token not found"}
	ErrNoBestRouteFound        = &errors.ErrorResponse{Code: errors.ErrorCode("WR-005"), Details: "no best route found"}
	ErrCannotCheckBalance      = &errors.ErrorResponse{Code: errors.ErrorCode("WR-006"), Details: "cannot check balance"}
	ErrLowAmountInForHopBridge = &errors.ErrorResponse{Code: errors.ErrorCode("WR-007"), Details: "bonder fee greater than estimated received, a higher amount is needed to cover fees"}
	ErrNoPositiveBalance       = &errors.ErrorResponse{Code: errors.ErrorCode("WR-008"), Details: "no positive balance"}
)
