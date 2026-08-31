package requests

import (
	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/status-im/status-go/internal/contracts/snt"
	"github.com/status-im/status-go/internal/errors"
	walletCommon "github.com/status-im/status-go/pkg/services/wallet/common"
	"github.com/status-im/status-go/pkg/services/wallet/router/fees"
	"github.com/status-im/status-go/pkg/services/wallet/router/sendtype"
	tokentypes "github.com/status-im/status-go/pkg/services/wallet/token/types"
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
	ErrFromChainNotSupported         = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-021"), Details: "from chain not supported"}
	ErrToChainNotSupported           = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-022"), Details: "to chain not supported"}
	// ErrFromAndToChainMustBeTheSame          = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-023"), Details: "from and to chain IDs must be the same"}
	ErrSwapSlippagePercentageMustBePositive = &errors.ErrorResponse{Code: errors.ErrorCode("WRR-024"), Details: "slippage percentage must be positive"}
)

type RouteInputParams struct {
	Uuid               string            `json:"uuid"`
	SendType           sendtype.SendType `json:"sendType" validate:"required"`
	AddrFrom           common.Address    `json:"addrFrom" validate:"required"`
	AddrTo             common.Address    `json:"addrTo" validate:"required"`
	AmountIn           *hexutil.Big      `json:"amountIn" validate:"required"`
	AmountOut          *hexutil.Big      `json:"amountOut"`
	TokenKey           string            `json:"tokenKey" validate:"required"`
	TokenIsOwnerToken  bool              `json:"tokenIDIsOwnerToken"`
	ToTokenKey         string            `json:"toTokenKey" validate:"required"`
	FromChainID        uint64            `json:"fromChainID"` // TODO: remove since it can be inferred from the token key
	ToChainID          uint64            `json:"toChainID"`   // TODO: remove since it can be inferred from the token key
	GasFeeMode         fees.GasFeeMode   `json:"gasFeeMode" validate:"required"`
	SlippagePercentage float32           `json:"slippagePercentage"`
	// RouteOrder is optional and only meaningful for processors that can rank
	// routes (currently LI.FI); empty means the provider's own default.
	RouteOrder  string `json:"routeOrder"`
	TestnetMode bool

	// For send types like EnsRegister, EnsRelease, EnsSetPubKey, StickersBuy
	Username  string       `json:"username"`
	PublicKey string       `json:"publicKey"`
	PackID    *hexutil.Big `json:"packID"`

	// Used internally
	PathTxCustomParams map[string]*PathTxCustomParams `json:"-"`

	// Community related params
	CommunityRouteInputParams *CommunityRouteInputParams `json:"communityRouteInputParams"`
}

func (i *RouteInputParams) UseCommunityTransferDetails() bool {
	if !i.SendType.IsCommunityRelatedTransfer() || i.CommunityRouteInputParams == nil {
		return false
	}
	return i.CommunityRouteInputParams.UseTransferDetails()
}

func (i *RouteInputParams) Validate() error {
	if !walletCommon.IsSupportedChainID(i.FromChainID) {
		return ErrFromChainNotSupported
	}

	if !walletCommon.IsSupportedChainID(i.ToChainID) {
		return ErrToChainNotSupported
	}

	if i.SendType == sendtype.ENSRegister {
		if i.Username == "" || i.PublicKey == "" {
			return ErrENSRegisterRequiresUsernameAndPubKey
		}
		if i.FromChainID == walletCommon.AnvilMainnet {
			// On Anvil, the token address is determined dynamically from the registrar contract
			return nil
		}
		if i.TestnetMode {
			// available only on sepolia testnet
			sntToken := tokentypes.Token{
				Token: &types.Token{
					ChainID: walletCommon.EthereumSepolia,
				},
			}
			var err error
			sntToken.Address, err = snt.ContractAddress(walletCommon.EthereumSepolia)
			if err != nil {
				return err
			}
			if i.TokenKey != sntToken.Key() {
				return ErrENSRegisterTestnetSTTOnly
			}
		} else {
			// available only on mainnet
			sntToken := tokentypes.Token{
				Token: &types.Token{
					ChainID: walletCommon.EthereumMainnet,
				},
			}
			var err error
			sntToken.Address, err = snt.ContractAddress(walletCommon.EthereumMainnet)
			if err != nil {
				return err
			}
			if i.TokenKey != sntToken.Key() {
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
		if i.ToTokenKey == "" {
			return ErrSwapRequiresToTokenID
		}
		if i.TokenKey == i.ToTokenKey {
			return ErrSwapTokenIDMustBeDifferent
		}

		if i.SlippagePercentage <= 0 {
			return ErrSwapSlippagePercentageMustBePositive
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
		if i.CommunityRouteInputParams == nil {
			return ErrNoCommunityParametersProvided
		}
		return i.CommunityRouteInputParams.validateCommunityRelatedInputs(i.SendType)
	}

	return nil
}
