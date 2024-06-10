package pathprocessor

import (
	"math/big"

	ethTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/wallet/router/bridge"
	"github.com/status-im/status-go/services/wallet/token"
)

type PathProcessor interface {
	// returns the name of the bridge
	Name() string
	// checks if the bridge is available for the given networks/tokens
	AvailableFor(params ProcessorInputParams) (bool, error)
	// calculates the fees for the bridge and returns the amount BonderFee and TokenFee (used for bridges)
	CalculateFees(params ProcessorInputParams) (*big.Int, *big.Int, error)
	// Pack the method for sending tx and method call's data
	PackTxInputData(params ProcessorInputParams) ([]byte, error)
	EstimateGas(params ProcessorInputParams) (uint64, error)
	CalculateAmountOut(params ProcessorInputParams) (*big.Int, error)
	Send(sendArgs *bridge.TransactionBridge, verifiedAccount *account.SelectedExtKey) (types.Hash, error)
	GetContractAddress(params ProcessorInputParams) (common.Address, error)
	BuildTransaction(sendArgs *bridge.TransactionBridge) (*ethTypes.Transaction, error)
	BuildTx(params ProcessorInputParams) (*ethTypes.Transaction, error)
}

type ProcessorInputParams struct {
	FromChain *params.Network
	ToChain   *params.Network
	FromAddr  common.Address
	ToAddr    common.Address
	FromToken *token.Token
	ToToken   *token.Token
	AmountIn  *big.Int

	// extra params
	BonderFee *big.Int
	Username  string
	PublicKey string
}
