package requests

import (
	"math/big"
	"reflect"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/errors"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/router/fees"
	"github.com/status-im/status-go/services/wallet/router/sendtype"
	tokenTypes "github.com/status-im/status-go/services/wallet/token/types"
)

var (
	ErrENSRegisterRequiresUsernameAndPubKey  = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-001"), Details: "username and public key are required for ENSRegister"}
	ErrENSRegisterTestnetSTTOnly             = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-002"), Details: "only STT is supported for ENSRegister on testnet"}
	ErrENSRegisterMainnetSNTOnly             = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-003"), Details: "only SNT is supported for ENSRegister on mainnet"}
	ErrENSReleaseRequiresUsername            = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-004"), Details: "username is required for ENSRelease"}
	ErrENSSetPubKeyRequiresUsernameAndPubKey = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-005"), Details: "username and public key are required for ENSSetPubKey"}
	ErrStickersBuyRequiresPackID             = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-006"), Details: "packID is required for StickersBuy"}
	ErrSwapRequiresToTokenID                 = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-007"), Details: "toTokenID is required for Swap"}
	ErrSwapTokenIDMustBeDifferent            = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-008"), Details: "tokenID and toTokenID must be different"}
	ErrSwapAmountInAmountOutMustBeExclusive  = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-009"), Details: "only one of amountIn or amountOut can be set"}
	ErrSwapAmountInMustBePositive            = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-010"), Details: "amountIn must be positive"}
	ErrSwapAmountOutMustBePositive           = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-011"), Details: "amountOut must be positive"}
	// ErrLockedAmountNotSupportedForNetwork        = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-012"), Details: "locked amount is not supported for the selected network"}
	// ErrLockedAmountNotNegative                   = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-013"), Details: "locked amount must not be negative"}
	// ErrLockedAmountExceedsTotalSendAmount        = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-014"), Details: "locked amount exceeds the total amount to send"}
	// ErrLockedAmountLessThanSendAmountAllNetworks = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-015"), Details: "locked amount is less than the total amount to send, but all networks are locked"}
	// ErrDisabledChainFoundAmongLockedNetworks     = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-016"), Details: "disabled chain found among locked networks"}
	ErrENSSetPubKeyInvalidUsername = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-017"), Details: "a valid username, ending in '.eth', is required for ENSSetPubKey"}
	// ErrLockedAmountExcludesAllSupported = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-018"), Details: "all supported chains are excluded, routing impossible"}
	// ErrCannotCheckLockedAmounts         = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-019"), Details: "cannot check locked amounts"}
	ErrNoCommunityParametersProvided = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-020"), Details: "no community parameters provided"}
	ErrNoFromChainProvided           = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-021"), Details: "from chain not provided"}
	ErrNoToChainProvided             = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-022"), Details: "to chain not provided"}
	ErrFromAndToChainMustBeTheSame   = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-023"), Details: "from and to chain IDs must be the same"}
)

type RouteInputParams struct {
	Uuid                 string            `json:"uuid"`
	SendType             sendtype.SendType `json:"sendType" validate:"required"`
	AddrFrom             common.Address    `json:"addrFrom" validate:"required"`
	AddrTo               common.Address    `json:"addrTo" validate:"required"`
	AmountIn             *hexutil.Big      `json:"amountIn" validate:"required"`
	AmountOut            *hexutil.Big      `json:"amountOut"`
	TokenID              string            `json:"tokenID" validate:"required"`
	TokenIDIsOwnerToken  bool              `json:"tokenIDIsOwnerToken"`
	ToTokenID            string            `json:"toTokenID"`
	DisabledFromChainIDs []uint64          `json:"disabledFromChainIDs"`
	DisabledToChainIDs   []uint64          `json:"disabledToChainIDs"`
	GasFeeMode           fees.GasFeeMode   `json:"gasFeeMode" validate:"required"`
	TestnetMode          bool

	// For send types like EnsRegister, EnsRelease, EnsSetPubKey, StickersBuy
	Username  string       `json:"username"`
	PublicKey string       `json:"publicKey"`
	PackID    *hexutil.Big `json:"packID"`

	// Used internally
	PathTxCustomParams map[string]*PathTxCustomParams `json:"-"`

	// Community related params
	CommunityRouteInputParams *CommunityRouteInputParams `json:"communityRouteInputParams"`

	// TODO: Remove two fields below once we implement a better solution for tests
	// Currently used for tests only
	TestsMode  bool
	TestParams *RouterTestParams
}

type RouterTestParams struct {
	TokenFrom             *tokenTypes.Token
	TokenPrices           map[string]float64
	EstimationMap         map[string]Estimation // [processor-name, estimation]
	BonderFeeMap          map[string]*big.Int   // [token-symbol, bonder-fee]
	SuggestedFees         *fees.SuggestedFees
	BaseFee               *big.Int
	BalanceMap            map[string]*big.Int // [token-symbol, balance]
	ApprovalGasEstimation uint64
	ApprovalL1Fee         uint64
}

type Estimation struct {
	Value uint64
	Err   error
}

func slicesEqual(a, b []uint64) bool {
	if len(a) != len(b) {
		return false
	}

	aCopy := make([]uint64, len(a))
	bCopy := make([]uint64, len(b))
	copy(aCopy, a)
	copy(bCopy, b)

	sort.Slice(aCopy, func(i, j int) bool { return aCopy[i] < aCopy[j] })
	sort.Slice(bCopy, func(i, j int) bool { return bCopy[i] < bCopy[j] })

	return reflect.DeepEqual(aCopy, bCopy)
}

func (i *RouteInputParams) UseCommunityTransferDetails() bool {
	if !i.SendType.IsCommunityRelatedTransfer() || i.CommunityRouteInputParams == nil {
		return false
	}
	return i.CommunityRouteInputParams.UseTransferDetails()
}

func (i *RouteInputParams) Validate() error {
	if i.SendType == sendtype.ENSRegister {
		if i.Username == "" || i.PublicKey == "" {
			return ErrENSRegisterRequiresUsernameAndPubKey
		}
		if i.TestnetMode {
			if i.TokenID != walletCommon.SttSymbol {
				return ErrENSRegisterTestnetSTTOnly
			}
		} else {
			if i.TokenID != walletCommon.SntSymbol {
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

	if i.SendType.IsCommunityRelatedTransfer() {
		if i.DisabledFromChainIDs == nil || len(i.DisabledFromChainIDs) == 0 {
			return ErrNoFromChainProvided
		}
		if i.DisabledToChainIDs == nil || len(i.DisabledToChainIDs) == 0 {
			return ErrNoToChainProvided
		}
		if !slicesEqual(i.DisabledFromChainIDs, i.DisabledToChainIDs) {
			return ErrFromAndToChainMustBeTheSame
		}

		if i.CommunityRouteInputParams == nil {
			return ErrNoCommunityParametersProvided
		}
		return i.CommunityRouteInputParams.validateCommunityRelatedInputs(i.SendType)
	}

	return nil
}
