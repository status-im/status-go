package router

import (
	"github.com/status-im/status-go/errors"
)

// Abbreviation `WR` for the error code stands for Wallet Router
var (
	ErrUsernameAndPubKeyRequiredForENSRegister   = &errors.ErrorResponse{Code: errors.ErrorCode("WR-001"), Details: "username and public key are required for ENSRegister"}
	ErrOnlySTTSupportedForENSRegisterOnTestnet   = &errors.ErrorResponse{Code: errors.ErrorCode("WR-002"), Details: "only STT is supported for ENSRegister on testnet"}
	ErrOnlySTTSupportedForENSReleaseOnTestnet    = &errors.ErrorResponse{Code: errors.ErrorCode("WR-003"), Details: "only STT is supported for ENSRelease on testnet"}
	ErrUsernameRequiredForENSRelease             = &errors.ErrorResponse{Code: errors.ErrorCode("WR-004"), Details: "username is required for ENSRelease"}
	ErrUsernameAndPubKeyRequiredForENSSetPubKey  = &errors.ErrorResponse{Code: errors.ErrorCode("WR-005"), Details: "username and public key are required for ENSSetPubKey"}
	ErrPackIDRequiredForStickersBuy              = &errors.ErrorResponse{Code: errors.ErrorCode("WR-006"), Details: "packID is required for StickersBuy"}
	ErrToTokenIDRequiredForSwap                  = &errors.ErrorResponse{Code: errors.ErrorCode("WR-007"), Details: "toTokenID is required for Swap"}
	ErrTokenIDAndToTokenIDDifferent              = &errors.ErrorResponse{Code: errors.ErrorCode("WR-008"), Details: "tokenID and toTokenID must be different"}
	ErrOnlyOneOfAmountInOrOutSet                 = &errors.ErrorResponse{Code: errors.ErrorCode("WR-009"), Details: "only one of amountIn or amountOut can be set"}
	ErrAmountInMustBePositive                    = &errors.ErrorResponse{Code: errors.ErrorCode("WR-010"), Details: "amountIn must be positive"}
	ErrAmountOutMustBePositive                   = &errors.ErrorResponse{Code: errors.ErrorCode("WR-011"), Details: "amountOut must be positive"}
	ErrLockedAmountNotSupportedForNetwork        = &errors.ErrorResponse{Code: errors.ErrorCode("WR-012"), Details: "locked amount is not supported for the selected network"}
	ErrLockedAmountMustBePositive                = &errors.ErrorResponse{Code: errors.ErrorCode("WR-013"), Details: "locked amount must be positive"}
	ErrLockedAmountExceedsTotalSendAmount        = &errors.ErrorResponse{Code: errors.ErrorCode("WR-014"), Details: "locked amount exceeds the total amount to send"}
	ErrLockedAmountLessThanSendAmountAllNetworks = &errors.ErrorResponse{Code: errors.ErrorCode("WR-015"), Details: "locked amount is less than the total amount to send, but all networks are locked"}
	ErrNotEnoughTokenBalance                     = &errors.ErrorResponse{Code: errors.ErrorCode("WR-016"), Details: "not enough token balance"}
	ErrNotEnoughNativeBalance                    = &errors.ErrorResponse{Code: errors.ErrorCode("WR-017"), Details: "not enough native balance"}
	ErrNativeTokenNotFound                       = &errors.ErrorResponse{Code: errors.ErrorCode("WR-018"), Details: "native token not found"}
)
