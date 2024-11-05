package transfer

import (
	"github.com/status-im/status-go/errors"
)

// Abbreviation `WT` for the error code stands for Wallet Transfer
var (
	ErrNoRoute                   = &errors.ErrorResponse{Code: errors.ErrorCode("WT-001"), Details: "no generated route"}
	ErrNoTrsansactionsBeingBuilt = &errors.ErrorResponse{Code: errors.ErrorCode("WT-002"), Details: "no transactions being built"}
	ErrMissingSignatureForTx     = &errors.ErrorResponse{Code: errors.ErrorCode("WT-003"), Details: "missing signature for transaction %s"}
)
