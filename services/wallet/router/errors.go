package router

import (
	"errors"

	sErrors "github.com/status-im/status-go/errors"
)

// Abbreviation `WR` for the error code stands for Wallet Router
var (
	ErrUsernameAndPubKeyRequiredForENSRegister   = &sErrors.ErrorResponse{Code: sErrors.ErrorCode("WR-001"), Details: "username and public key are required for ENSRegister"}
	ErrOnlySTTSupportedForENSRegisterOnTestnet   = &sErrors.ErrorResponse{Code: sErrors.ErrorCode("WR-002"), Details: "only STT is supported for ENSRegister on testnet"}
	ErrOnlySTTSupportedForENSReleaseOnTestnet    = &sErrors.ErrorResponse{Code: sErrors.ErrorCode("WR-003"), Details: "only STT is supported for ENSRelease on testnet"}
	ErrUsernameRequiredForENSRelease             = &sErrors.ErrorResponse{Code: sErrors.ErrorCode("WR-004"), Details: "username is required for ENSRelease"}
	ErrUsernameAndPubKeyRequiredForENSSetPubKey  = &sErrors.ErrorResponse{Code: sErrors.ErrorCode("WR-005"), Details: "username and public key are required for ENSSetPubKey"}
	ErrPackIDRequiredForStickersBuy              = &sErrors.ErrorResponse{Code: sErrors.ErrorCode("WR-006"), Details: "packID is required for StickersBuy"}
	ErrToTokenIDRequiredForSwap                  = &sErrors.ErrorResponse{Code: sErrors.ErrorCode("WR-007"), Details: "toTokenID is required for Swap"}
	ErrTokenIDAndToTokenIDDifferent              = &sErrors.ErrorResponse{Code: sErrors.ErrorCode("WR-008"), Details: "tokenID and toTokenID must be different"}
	ErrOnlyOneOfAmountInOrOutSet                 = &sErrors.ErrorResponse{Code: sErrors.ErrorCode("WR-009"), Details: "only one of amountIn or amountOut can be set"}
	ErrAmountInMustBePositive                    = &sErrors.ErrorResponse{Code: sErrors.ErrorCode("WR-010"), Details: "amountIn must be positive"}
	ErrAmountOutMustBePositive                   = &sErrors.ErrorResponse{Code: sErrors.ErrorCode("WR-011"), Details: "amountOut must be positive"}
	ErrLockedAmountNotSupportedForNetwork        = &sErrors.ErrorResponse{Code: sErrors.ErrorCode("WR-012"), Details: "locked amount is not supported for the selected network"}
	ErrLockedAmountMustBePositive                = &sErrors.ErrorResponse{Code: sErrors.ErrorCode("WR-013"), Details: "locked amount must be positive"}
	ErrLockedAmountExceedsTotalSendAmount        = &sErrors.ErrorResponse{Code: sErrors.ErrorCode("WR-014"), Details: "locked amount exceeds the total amount to send"}
	ErrLockedAmountLessThanSendAmountAllNetworks = &sErrors.ErrorResponse{Code: sErrors.ErrorCode("WR-015"), Details: "locked amount is less than the total amount to send, but all networks are locked"}
	ErrNotEnoughTokenBalance                     = &sErrors.ErrorResponse{Code: sErrors.ErrorCode("WR-016"), Details: "not enough token balance"}
	ErrNotEnoughNativeBalance                    = &sErrors.ErrorResponse{Code: sErrors.ErrorCode("WR-017"), Details: "not enough native balance"}
	ErrNativeTokenNotFound                       = &sErrors.ErrorResponse{Code: sErrors.ErrorCode("WR-018"), Details: "native token not found"}
	ErrDisabledChainFoundAmongLockedNetworks     = &sErrors.ErrorResponse{Code: sErrors.ErrorCode("WR-019"), Details: "disabled chain found among locked networks"}
)

var (
	ErrorENSRegisterRequires                  = errors.New("username and public key are required for ENSRegister")
	ErrorENSRegisterTestNetSTTOnly            = errors.New("only STT is supported for ENSRegister on testnet")
	ErrorENSRegisterSNTOnly                   = errors.New("only SNT is supported for ENSRegister")
	ErrorENSReleaseRequires                   = errors.New("username is required for ENSRelease")
	ErrorENSSetPubKeyRequires                 = errors.New("username and public key are required for ENSSetPubKey")
	ErrorStickersBuyRequires                  = errors.New("packID is required for StickersBuy")
	ErrorSwapRequires                         = errors.New("toTokenID is required for Swap")
	ErrorSwapTokenIDMustBeDifferent           = errors.New("tokenID and toTokenID must be different")
	ErrorSwapAmountInAmountOutMustBeExclusive = errors.New("only one of amountIn or amountOut can be set")
	ErrorSwapAmountInMustBePositive           = errors.New("amountIn must be positive")
	ErrorSwapAmountOutMustBePositive          = errors.New("amountOut must be positive")
	ErrorLockedAmountNotSupportedNetwork      = errors.New("locked amount is not supported for the selected network")
	ErrorLockedAmountNotNegative              = errors.New("locked amount must not be negative")
	ErrorLockedAmountExcludesAllSupported     = errors.New("all supported chains are excluded, routing impossible")
)
