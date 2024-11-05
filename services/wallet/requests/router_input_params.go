package requests

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/errors"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/router/fees"
	"github.com/status-im/status-go/services/wallet/router/pathprocessor"
	"github.com/status-im/status-go/services/wallet/router/sendtype"
	"github.com/status-im/status-go/services/wallet/token"
)

var (
	ErrENSRegisterRequiresUsernameAndPubKey      = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-001"), Details: "username and public key are required for ENSRegister"}
	ErrENSRegisterTestnetSTTOnly                 = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-002"), Details: "only STT is supported for ENSRegister on testnet"}
	ErrENSRegisterMainnetSNTOnly                 = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-003"), Details: "only SNT is supported for ENSRegister on mainnet"}
	ErrENSReleaseRequiresUsername                = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-004"), Details: "username is required for ENSRelease"}
	ErrENSSetPubKeyRequiresUsernameAndPubKey     = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-005"), Details: "username and public key are required for ENSSetPubKey"}
	ErrStickersBuyRequiresPackID                 = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-006"), Details: "packID is required for StickersBuy"}
	ErrSwapRequiresToTokenID                     = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-007"), Details: "toTokenID is required for Swap"}
	ErrSwapTokenIDMustBeDifferent                = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-008"), Details: "tokenID and toTokenID must be different"}
	ErrSwapAmountInAmountOutMustBeExclusive      = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-009"), Details: "only one of amountIn or amountOut can be set"}
	ErrSwapAmountInMustBePositive                = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-010"), Details: "amountIn must be positive"}
	ErrSwapAmountOutMustBePositive               = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-011"), Details: "amountOut must be positive"}
	ErrLockedAmountNotSupportedForNetwork        = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-012"), Details: "locked amount is not supported for the selected network"}
	ErrLockedAmountNotNegative                   = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-013"), Details: "locked amount must not be negative"}
	ErrLockedAmountExceedsTotalSendAmount        = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-014"), Details: "locked amount exceeds the total amount to send"}
	ErrLockedAmountLessThanSendAmountAllNetworks = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-015"), Details: "locked amount is less than the total amount to send, but all networks are locked"}
	ErrDisabledChainFoundAmongLockedNetworks     = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-016"), Details: "disabled chain found among locked networks"}
	ErrENSSetPubKeyInvalidUsername               = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-017"), Details: "a valid username, ending in '.eth', is required for ENSSetPubKey"}
	ErrLockedAmountExcludesAllSupported          = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-018"), Details: "all supported chains are excluded, routing impossible"}
	ErrCannotCheckLockedAmounts                  = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-019"), Details: "cannot check locked amounts"}
)

type RouteInputParams struct {
	Uuid                 string                  `json:"uuid"`
	SendType             sendtype.SendType       `json:"sendType" validate:"required"`
	AddrFrom             common.Address          `json:"addrFrom" validate:"required"`
	AddrTo               common.Address          `json:"addrTo" validate:"required"`
	AmountIn             *hexutil.Big            `json:"amountIn" validate:"required"`
	AmountOut            *hexutil.Big            `json:"amountOut"`
	TokenID              string                  `json:"tokenID" validate:"required"`
	TokenIDIsOwnerToken  bool                    `json:"tokenIDIsOwnerToken"`
	ToTokenID            string                  `json:"toTokenID"`
	DisabledFromChainIDs []uint64                `json:"disabledFromChainIDs"`
	DisabledToChainIDs   []uint64                `json:"disabledToChainIDs"`
	GasFeeMode           fees.GasFeeMode         `json:"gasFeeMode" validate:"required"`
	FromLockedAmount     map[uint64]*hexutil.Big `json:"fromLockedAmount"`
	TestnetMode          bool

	// For send types like EnsRegister, EnsRelease, EnsSetPubKey, StickersBuy
	Username  string       `json:"username"`
	PublicKey string       `json:"publicKey"`
	PackID    *hexutil.Big `json:"packID"`

	// TODO: Remove two fields below once we implement a better solution for tests
	// Currently used for tests only
	TestsMode  bool
	TestParams *RouterTestParams
}

type RouterTestParams struct {
	TokenFrom             *token.Token
	TokenPrices           map[string]float64
	EstimationMap         map[string]pathprocessor.Estimation // [processor-name, estimation]
	BonderFeeMap          map[string]*big.Int                 // [token-symbol, bonder-fee]
	SuggestedFees         *fees.SuggestedFees
	BaseFee               *big.Int
	BalanceMap            map[string]*big.Int // [token-symbol, balance]
	ApprovalGasEstimation uint64
	ApprovalL1Fee         uint64
}

func (i *RouteInputParams) Validate() error {
	if i.SendType == sendtype.ENSRegister {
		if i.Username == "" || i.PublicKey == "" {
			return ErrENSRegisterRequiresUsernameAndPubKey
		}
		if i.TestnetMode {
			if i.TokenID != pathprocessor.SttSymbol {
				return ErrENSRegisterTestnetSTTOnly
			}
		} else {
			if i.TokenID != pathprocessor.SntSymbol {
				return ErrENSRegisterMainnetSNTOnly
			}
		}
		return nil
	}

	if i.SendType == sendtype.ENSRelease {
		if i.Username == "" {
			return ErrENSReleaseRequiresUsername
		}
	}

	if i.SendType == sendtype.ENSSetPubKey {
		if i.Username == "" || i.PublicKey == "" {
			return ErrENSSetPubKeyRequiresUsernameAndPubKey
		}

		if walletCommon.ValidateENSUsername(i.Username) != nil {
			return ErrENSSetPubKeyInvalidUsername
		}
	}

	if i.SendType == sendtype.StickersBuy {
		if i.PackID == nil {
			return ErrStickersBuyRequiresPackID
		}
	}

	if i.SendType == sendtype.Swap {
		if i.ToTokenID == "" {
			return ErrSwapRequiresToTokenID
		}
		if i.TokenID == i.ToTokenID {
			return ErrSwapTokenIDMustBeDifferent
		}

		if i.AmountIn != nil &&
			i.AmountOut != nil &&
			i.AmountIn.ToInt().Cmp(walletCommon.ZeroBigIntValue()) > 0 &&
			i.AmountOut.ToInt().Cmp(walletCommon.ZeroBigIntValue()) > 0 {
			return ErrSwapAmountInAmountOutMustBeExclusive
		}

		if i.AmountIn != nil && i.AmountIn.ToInt().Sign() < 0 {
			return ErrSwapAmountInMustBePositive
		}

		if i.AmountOut != nil && i.AmountOut.ToInt().Sign() < 0 {
			return ErrSwapAmountOutMustBePositive
		}
	}

	return i.validateFromLockedAmount()
}

func (i *RouteInputParams) validateFromLockedAmount() error {
	if i.FromLockedAmount == nil || len(i.FromLockedAmount) == 0 {
		return nil
	}

	var suppNetworks map[uint64]bool
	if i.TestnetMode {
		suppNetworks = walletCommon.CopyMapGeneric(walletCommon.SupportedTestNetworks, nil).(map[uint64]bool)
	} else {
		suppNetworks = walletCommon.CopyMapGeneric(walletCommon.SupportedNetworks, nil).(map[uint64]bool)
	}

	if suppNetworks == nil {
		return ErrCannotCheckLockedAmounts
	}

	totalLockedAmount := big.NewInt(0)
	excludedChainCount := 0

	for chainID, amount := range i.FromLockedAmount {
		if walletCommon.ArrayContainsElement(chainID, i.DisabledFromChainIDs) {
			return ErrDisabledChainFoundAmongLockedNetworks
		}

		if i.TestnetMode {
			if !walletCommon.SupportedTestNetworks[chainID] {
				return ErrLockedAmountNotSupportedForNetwork
			}
		} else {
			if !walletCommon.SupportedNetworks[chainID] {
				return ErrLockedAmountNotSupportedForNetwork
			}
		}

		if amount == nil || amount.ToInt().Sign() < 0 {
			return ErrLockedAmountNotNegative
		}

		if !(amount.ToInt().Sign() > 0) {
			excludedChainCount++
		}
		delete(suppNetworks, chainID)
		totalLockedAmount = new(big.Int).Add(totalLockedAmount, amount.ToInt())
	}

	if (!i.TestnetMode && excludedChainCount == len(walletCommon.SupportedNetworks)) ||
		(i.TestnetMode && excludedChainCount == len(walletCommon.SupportedTestNetworks)) {
		return ErrLockedAmountExcludesAllSupported
	}

	if totalLockedAmount.Cmp(i.AmountIn.ToInt()) > 0 {
		return ErrLockedAmountExceedsTotalSendAmount
	}
	if totalLockedAmount.Cmp(i.AmountIn.ToInt()) < 0 && len(suppNetworks) == 0 {
		return ErrLockedAmountLessThanSendAmountAllNetworks
	}
	return nil
}
