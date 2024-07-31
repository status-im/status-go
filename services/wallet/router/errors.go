package router

import (
	"github.com/status-im/status-go/errors"
)

// Abbreviation `WR` for the error code stands for Wallet Router
var (
	ErrENSRegisterRequiresUsernameAndPubKey      = &errors.ErrorResponse{Code: errors.ErrorCode("WR-001"), Details: "username and public key are required for ENSRegister"}
	ErrENSRegisterTestnetSTTOnly                 = &errors.ErrorResponse{Code: errors.ErrorCode("WR-002"), Details: "only STT is supported for ENSRegister on testnet"}
	ErrENSRegisterMainnetSNTOnly                 = &errors.ErrorResponse{Code: errors.ErrorCode("WR-003"), Details: "only SNT is supported for ENSRegister on mainnet"}
	ErrENSReleaseRequiresUsername                = &errors.ErrorResponse{Code: errors.ErrorCode("WR-004"), Details: "username is required for ENSRelease"}
	ErrENSSetPubKeyRequiresUsernameAndPubKey     = &errors.ErrorResponse{Code: errors.ErrorCode("WR-005"), Details: "username and public key are required for ENSSetPubKey"}
	ErrStickersBuyRequiresPackID                 = &errors.ErrorResponse{Code: errors.ErrorCode("WR-006"), Details: "packID is required for StickersBuy"}
	ErrSwapRequiresToTokenID                     = &errors.ErrorResponse{Code: errors.ErrorCode("WR-007"), Details: "toTokenID is required for Swap"}
	ErrSwapTokenIDMustBeDifferent                = &errors.ErrorResponse{Code: errors.ErrorCode("WR-008"), Details: "tokenID and toTokenID must be different"}
	ErrSwapAmountInAmountOutMustBeExclusive      = &errors.ErrorResponse{Code: errors.ErrorCode("WR-009"), Details: "only one of amountIn or amountOut can be set"}
	ErrSwapAmountInMustBePositive                = &errors.ErrorResponse{Code: errors.ErrorCode("WR-010"), Details: "amountIn must be positive"}
	ErrSwapAmountOutMustBePositive               = &errors.ErrorResponse{Code: errors.ErrorCode("WR-011"), Details: "amountOut must be positive"}
	ErrLockedAmountNotSupportedForNetwork        = &errors.ErrorResponse{Code: errors.ErrorCode("WR-012"), Details: "locked amount is not supported for the selected network"}
	ErrLockedAmountNotNegative                   = &errors.ErrorResponse{Code: errors.ErrorCode("WR-013"), Details: "locked amount must not be negative"}
	ErrLockedAmountExceedsTotalSendAmount        = &errors.ErrorResponse{Code: errors.ErrorCode("WR-014"), Details: "locked amount exceeds the total amount to send"}
	ErrLockedAmountLessThanSendAmountAllNetworks = &errors.ErrorResponse{Code: errors.ErrorCode("WR-015"), Details: "locked amount is less than the total amount to send, but all networks are locked"}
	ErrNotEnoughTokenBalance                     = &errors.ErrorResponse{Code: errors.ErrorCode("WR-016"), Details: "{\"token\": \"%s\", \"chainId\": %d}"}
	ErrNotEnoughNativeBalance                    = &errors.ErrorResponse{Code: errors.ErrorCode("WR-017"), Details: "{\"token\": \"%s\", \"chainId\": %d}"}
	ErrNativeTokenNotFound                       = &errors.ErrorResponse{Code: errors.ErrorCode("WR-018"), Details: "native token not found"}
	ErrDisabledChainFoundAmongLockedNetworks     = &errors.ErrorResponse{Code: errors.ErrorCode("WR-019"), Details: "disabled chain found among locked networks"}
	ErrENSSetPubKeyInvalidUsername               = &errors.ErrorResponse{Code: errors.ErrorCode("WR-020"), Details: "a valid username, ending in '.eth', is required for ENSSetPubKey"}
	ErrLockedAmountExcludesAllSupported          = &errors.ErrorResponse{Code: errors.ErrorCode("WR-021"), Details: "all supported chains are excluded, routing impossible"}
	ErrTokenNotFound                             = &errors.ErrorResponse{Code: errors.ErrorCode("WR-022"), Details: "token not found"}
	ErrNoBestRouteFound                          = &errors.ErrorResponse{Code: errors.ErrorCode("WR-023"), Details: "no best route found"}
	ErrCannotCheckBalance                        = &errors.ErrorResponse{Code: errors.ErrorCode("WR-024"), Details: "cannot check balance"}
	ErrCannotCheckLockedAmounts                  = &errors.ErrorResponse{Code: errors.ErrorCode("WR-025"), Details: "cannot check locked amounts"}
	ErrLowAmountInForHopBridge                   = &errors.ErrorResponse{Code: errors.ErrorCode("WR-026"), Details: "bonder fee greater than estimated received, a higher amount is needed to cover fees"}
)
