package transfer

import (
	"github.com/status-im/status-go/internal/errors"
)

// Abbreviation `WT` for the error code stands for Wallet Transfer
var (
	ErrNoRoute                   = &errors.ErrorResponse{Code: errors.ErrorCode("WT-001"), Details: "no generated route"}
	ErrNoTrsansactionsBeingBuilt = &errors.ErrorResponse{Code: errors.ErrorCode("WT-002"), Details: "no transactions being built"}
	ErrMissingSignatureForTx     = &errors.ErrorResponse{Code: errors.ErrorCode("WT-003"), Details: "missing signature for transaction %s"}
	// WT-004 is ErrInvalidSignatureDetails in pkg/services/wallet/requests
	ErrRouteGasEstimateExpired = &errors.ErrorResponse{Code: errors.ErrorCode("WT-005"), Details: "the route's gas estimate is out of date, request the route again"}
)
