package pathprocessor

import (
	"math/big"

	ethTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/wallet/token"
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
	EstimateGas(params ProcessorInputParams) (uint64, error)
	// CalculateAmountOut calculates the amount out
	CalculateAmountOut(params ProcessorInputParams) (*big.Int, error)
	// Send sends the tx
	Send(sendArgs *MultipathProcessorTxArgs, verifiedAccount *account.SelectedExtKey) (types.Hash, error)
	// GetContractAddress returns the contract address
	GetContractAddress(params ProcessorInputParams) (common.Address, error)
	// BuildTransaction builds the transaction based on MultipathProcessorTxArgs
	BuildTransaction(sendArgs *MultipathProcessorTxArgs) (*ethTypes.Transaction, error)
}

type PathProcessorClearable interface {
	// Clear clears the local cache
	Clear()
}

type ProcessorInputParams struct {
	FromChain *params.Network
	ToChain   *params.Network
	FromAddr  common.Address
	ToAddr    common.Address
	FromToken *token.Token
	ToToken   *token.Token
	AmountIn  *big.Int
	AmountOut *big.Int

	// extra params
	BonderFee *big.Int
	Username  string
	PublicKey string
	PackID    *big.Int

	// for testing purposes
	TestsMode                 bool
	TestEstimationMap         map[string]uint64   // [brifge-name, estimated-value]
	TestBonderFeeMap          map[string]*big.Int // [token-symbol, bonder-fee]
	TestApprovalGasEstimation uint64
	TestApprovalL1Fee         uint64
}
