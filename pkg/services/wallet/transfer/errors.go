package transfer

import (
	stderrors "errors"

	"github.com/status-im/status-go/internal/errors"
)

// Abbreviation `WT` for the error code stands for Wallet Transfer
var (
	ErrNoRoute                   = &errors.ErrorResponse{Code: errors.ErrorCode("WT-001"), Details: "no generated route"}
	ErrNoTrsansactionsBeingBuilt = &errors.ErrorResponse{Code: errors.ErrorCode("WT-002"), Details: "no transactions being built"}
	ErrMissingSignatureForTx     = &errors.ErrorResponse{Code: errors.ErrorCode("WT-003"), Details: "missing signature for transaction %s"}
	// WT-004 is ErrInvalidSignatureDetails in pkg/services/wallet/requests
	ErrRouteGasEstimateExpired = &errors.ErrorResponse{Code: errors.ErrorCode("WT-005"), Details: "the route's gas estimate is out of date, request the route again"}
	ErrGasBelowEstimate        = &errors.ErrorResponse{Code: errors.ErrorCode("WT-006"), Details: "the transaction's gas limit is below the current gas estimate, request the route again"}
)

// RouteNeedsReview reports a route refused at build (WT-005, WT-006); nothing was sent. Building the
// same route again fails the same way: request it again (SuggestedRoutes, or ReevaluateRouterPath for
// a swap's main tx after its approval), so its gas and fees are suggested and shown anew. A custom
// gas amount below the new estimate is refused until the user raises it.
func RouteNeedsReview(err error) bool {
	return stderrors.Is(err, ErrRouteGasEstimateExpired) || stderrors.Is(err, ErrGasBelowEstimate)
}
