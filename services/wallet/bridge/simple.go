package bridge

import (
	"math/big"

	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/transactions"
)

type SimpleBridge struct {
	transactor *transactions.Transactor
}

func NewSimpleBridge(transactor *transactions.Transactor) *SimpleBridge {
	return &SimpleBridge{transactor: transactor}
}

func (s *SimpleBridge) Name() string {
	return "Simple"
}

func (s *SimpleBridge) Can(from, to *params.Network, token *token.Token, balance *big.Int) (bool, error) {
	return from.ChainID == to.ChainID, nil
}

func (s *SimpleBridge) CalculateFees(from, to *params.Network, token *token.Token, amountIn *big.Int, nativeTokenPrice, tokenPrice float64, gasPrice *big.Float) (*big.Int, *big.Int, error) {
	return big.NewInt(0), big.NewInt(0), nil
}

func (s *SimpleBridge) EstimateGas(from, to *params.Network, token *token.Token, amountIn *big.Int) (uint64, error) {
	// TODO: replace by estimate function
	if token.IsNative() {
		return 22000, nil // default gas limit for eth transaction
	}

	return 200000, nil //default gas limit for erc20 transaction
}

func (s *SimpleBridge) Send(sendArgs *TransactionBridge, verifiedAccount *account.SelectedExtKey) (types.Hash, error) {
	return s.transactor.SendTransactionWithChainID(sendArgs.ChainID, *sendArgs.SimpleTx, verifiedAccount)
}

func (s *SimpleBridge) CalculateAmountOut(from, to *params.Network, amountIn *big.Int, symbol string) (*big.Int, error) {
	return amountIn, nil
}
