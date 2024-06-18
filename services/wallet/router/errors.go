package router

import "errors"

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
