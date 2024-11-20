package pathprocessor

import (
	"context"

	"github.com/status-im/status-go/errors"

	pathProcessorCommon "github.com/status-im/status-go/services/wallet/router/pathprocessor/common"
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
	ErrTransferCustomError            = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-025"), Details: "Transfer custom error"}
	ErrERC721TransferCustomError      = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-026"), Details: "ERC721Transfer custom error"}
	ErrERC1155TransferCustomError     = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-027"), Details: "ERC1155Transfer custom error"}
	ErrBridgeHopCustomError           = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-028"), Details: "Hop custom error"}
	ErrBridgeCellerCustomError        = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-029"), Details: "CBridge custom error"}
	ErrSwapParaswapCustomError        = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-030"), Details: "Paraswap custom error"}
	ErrENSRegisterCustomError         = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-031"), Details: "ENSRegister custom error"}
	ErrENSReleaseCustomError          = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-032"), Details: "ENSRelease custom error"}
	ErrENSPublicKeyCustomError        = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-033"), Details: "ENSPublicKey custom error"}
	ErrStickersBuyCustomError         = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-034"), Details: "StickersBuy custom error"}
	ErrContextCancelled               = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-035"), Details: "context cancelled"}
	ErrContextDeadlineExceeded        = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-036"), Details: "context deadline exceeded"}
	ErrPriceTimeout                   = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-037"), Details: "price timeout"}
	ErrNotEnoughLiquidity             = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-038"), Details: "not enough liquidity"}
	ErrPriceImpactTooHigh             = &errors.ErrorResponse{Code: errors.ErrorCode("WPP-039"), Details: "price impact too high"}
)

func createErrorResponse(processorName string, err error) error {
	if err == nil {
		return nil
	}

	genericErrResp := errors.CreateErrorResponseFromError(err).(*errors.ErrorResponse)

	if genericErrResp.Code != errors.GenericErrorCode {
		return genericErrResp
	}

	switch genericErrResp.Details {
	case context.Canceled.Error():
		return ErrContextCancelled
	case context.DeadlineExceeded.Error():
		return ErrContextDeadlineExceeded
	}

	var customErrResp *errors.ErrorResponse
	switch processorName {
	case pathProcessorCommon.ProcessorTransferName:
		customErrResp = ErrTransferCustomError
	case pathProcessorCommon.ProcessorERC721Name:
		customErrResp = ErrERC721TransferCustomError
	case pathProcessorCommon.ProcessorERC1155Name:
		customErrResp = ErrERC1155TransferCustomError
	case pathProcessorCommon.ProcessorBridgeHopName:
		customErrResp = ErrBridgeHopCustomError
	case pathProcessorCommon.ProcessorBridgeCelerName:
		customErrResp = ErrBridgeCellerCustomError
	case pathProcessorCommon.ProcessorSwapParaswapName:
		customErrResp = ErrSwapParaswapCustomError
	case pathProcessorCommon.ProcessorENSRegisterName:
		customErrResp = ErrENSRegisterCustomError
	case pathProcessorCommon.ProcessorENSReleaseName:
		customErrResp = ErrENSReleaseCustomError
	case pathProcessorCommon.ProcessorENSPublicKeyName:
		customErrResp = ErrENSPublicKeyCustomError
	case pathProcessorCommon.ProcessorStickersBuyName:
		customErrResp = ErrStickersBuyCustomError
	default:
		return genericErrResp
	}

	customErrResp.Details = genericErrResp.Details
	return customErrResp
}

func IsCustomError(err error) bool {
	if err == nil {
		return false
	}

	errResp, ok := err.(*errors.ErrorResponse)
	if !ok {
		return false
	}

	switch errResp {
	case ErrTransferCustomError,
		ErrERC721TransferCustomError,
		ErrERC1155TransferCustomError,
		ErrBridgeHopCustomError,
		ErrBridgeCellerCustomError,
		ErrSwapParaswapCustomError,
		ErrENSRegisterCustomError,
		ErrENSReleaseCustomError,
		ErrENSPublicKeyCustomError,
		ErrStickersBuyCustomError:
		return true
	default:
		return false
	}
}
