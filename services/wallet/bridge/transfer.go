package bridge

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/transactions"
)

type TransferBridge struct {
	transactor *transactions.Transactor
}

func NewTransferBridge(transactor *transactions.Transactor) *TransferBridge {
	return &TransferBridge{transactor: transactor}
}

func (s *TransferBridge) Name() string {
	return "Transfer"
}

func (s *TransferBridge) Can(from, to *params.Network, token *token.Token, balance *big.Int) (bool, error) {
	return from.ChainID == to.ChainID, nil
}

func (s *TransferBridge) CalculateFees(from, to *params.Network, token *token.Token, amountIn *big.Int, nativeTokenPrice, tokenPrice float64, gasPrice *big.Float) (*big.Int, *big.Int, error) {
	return big.NewInt(0), big.NewInt(0), nil
}

func (s *TransferBridge) EstimateGas(from, to *params.Network, account common.Address, token *token.Token, amountIn *big.Int) (uint64, error) {
	// TODO: replace by estimate function
	if token.IsNative() {
		return 22000, nil // default gas limit for eth transaction
	}

	return 200000, nil //default gas limit for erc20 transaction
}

func (s *TransferBridge) Send(sendArgs *TransactionBridge, verifiedAccount *account.SelectedExtKey) (types.Hash, error) {
	return s.transactor.SendTransactionWithChainID(sendArgs.ChainID, *sendArgs.TransferTx, verifiedAccount)
}

func (s *TransferBridge) BuildTransaction(sendArgs *TransactionBridge) (*ethTypes.Transaction, error) {
	return s.transactor.ValidateAndBuildTransaction(sendArgs.ChainID, *sendArgs.TransferTx)
}

func (s *TransferBridge) CalculateAmountOut(from, to *params.Network, amountIn *big.Int, symbol string) (*big.Int, error) {
	return amountIn, nil
}

func (s *TransferBridge) GetContractAddress(network *params.Network, token *token.Token) *common.Address {
	return nil
}
