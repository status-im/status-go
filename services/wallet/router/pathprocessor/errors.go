package pathprocessor

import (
	"github.com/status-im/status-go/errors"
)

// Abbreviartion `WPP` for the error code stands for `Wallet Path Processor`
var (
	ErrFailedToParseBaseFee           = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-001"), Details: "failed to parse base fee"}
	ErrFailedToParsePercentageFee     = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-002"), Details: "failed to parse percentage fee"}
	ErrContractNotFound               = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-003"), Details: "contract not found"}
	ErrNetworkNotFound                = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-004"), Details: "network not found"}
	ErrTokenNotFound                  = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-005"), Details: "token not found"}
	ErrNoEstimationFound              = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-006"), Details: "no estimation found"}
	ErrNotAvailableForContractType    = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-007"), Details: "not available for contract type"}
	ErrNoBonderFeeFound               = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-008"), Details: "no bonder fee found"}
	ErrContractTypeNotSupported       = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-009"), Details: "contract type not supported"}
	ErrFromChainNotSupported          = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-010"), Details: "from chain not supported"}
	ErrToChainNotSupported            = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-011"), Details: "to chain not supported"}
	ErrTxForChainNotSupported         = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-012"), Details: "tx for chain not supported"}
	ErrENSResolverNotFound            = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-013"), Details: "ENS resolver not found"}
	ErrENSRegistrarNotFound           = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-014"), Details: "ENS registrar not found"}
	ErrToAndFromTokensMustBeSet       = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-015"), Details: "to and from tokens must be set"}
	ErrCannotResolveTokens            = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-016"), Details: "cannot resolve tokens"}
	ErrPriceRouteNotFound             = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-017"), Details: "price route not found"}
	ErrConvertingAmountToBigInt       = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-018"), Details: "converting amount to big.Int"}
	ErrNoChainSet                     = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-019"), Details: "no chain set"}
	ErrNoTokenSet                     = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-020"), Details: "no token set"}
	ErrToTokenShouldNotBeSet          = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-021"), Details: "to token should not be set"}
	ErrFromAndToChainsMustBeDifferent = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-022"), Details: "from and to chains must be different"}
	ErrFromAndToChainsMustBeSame      = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-023"), Details: "from and to chains must be same"}
	ErrFromAndToTokensMustBeDifferent = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-024"), Details: "from and to tokens must be different"}
)
