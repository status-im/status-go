package pathprocessor

//go:generate go tool mockgen -package=mock_pathprocessor -source=processor.go -destination=mock/processor.go

import (
	"context"
	"math/big"

	ethTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/wallet/permit2"
	"github.com/status-im/status-go/services/wallet/requests"
	tokentypes "github.com/status-im/status-go/services/wallet/token/types"
	"github.com/status-im/status-go/services/wallet/wallettypes"
)

type PathProcessor interface {
	// Name returns the name of the bridge
	Name() string
	// AvailableFor checks if the bridge is available for the given networks/tokens
	AvailableFor(params ProcessorInputParams) (bool, error)
	// CalculateFees calculates the fees for the bridge and returns the amount BonderFee and TokenFee (used for bridges)
	CalculateFees(params ProcessorInputParams) (*big.Int, *big.Int, error)
	// PackTxInputData packs tx for sending
	PackTxInputData(params ProcessorInputParams) ([]byte, error)
	// EstimateGas estimates the gas
	EstimateGas(params ProcessorInputParams, input []byte) (uint64, error)
	// CalculateAmountOut calculates the amount out
	CalculateAmountOut(params ProcessorInputParams) (*big.Int, error)
	// GetContractAddress returns the contract address
	GetContractAddress(params ProcessorInputParams) (common.Address, error)
	// BuildTransactionV2 builds the transaction based on SendTxArgs, returns the transaction and the used nonce (lastUsedNonce is -1 if it's the first tx)
	BuildTransactionV2(sendArgs *wallettypes.SendTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error)
}

type PathProcessorClearable interface {
	// Clear clears the local cache
	Clear()
}

// PermitResolver is implemented by processors that can pull the user's tokens with an
// off-chain permit instead of an approval tx. Optional: the router type-asserts for it
// and falls back to approve-then-swap when it isn't implemented.
type PermitResolver interface {
	ResolvePermit(ctx context.Context, params ProcessorInputParams) (*permit2.Plan, error)
}

type ProcessorCommunityTokenParams struct {
	Name               string
	Symbol             string
	TokenURI           string
	Transferable       bool
	RemoteSelfDestruct bool
	Supply             *big.Int
	OwnerTokenAddress  string
	MasterTokenAddress string
}

type ProcessorInputParams struct {
	FromChain *params.Network
	ToChain   *params.Network
	FromAddr  common.Address
	ToAddr    common.Address
	FromToken *tokentypes.Token
	ToToken   *tokentypes.Token
	AmountIn  *big.Int
	AmountOut *big.Int

	// extra params
	BonderFee          *big.Int
	Username           string
	PublicKey          string
	PackID             *big.Int
	SlippagePercentage float32

	// community related params
	CommunityParams *requests.CommunityRouteInputParams

	// PermitPlan is set by the router once it resolved whether this swap can use a permit.
	// Processors use it to account for the extra gas the Permit2Proxy call costs.
	PermitPlan *permit2.Plan

	// for testing purposes
	TestsMode                 bool
	TestEstimationMap         map[string]requests.Estimation // [bridge-name, estimation]
	TestBonderFeeMap          map[string]*big.Int            // [token-symbol, bonder-fee]
	TestApprovalGasEstimation uint64
	TestApprovalL1Fee         uint64
}
