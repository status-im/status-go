package bridge

import (
	"math/big"

	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/transactions"
)

type TransactionBridge struct {
	BridgeName string
	ChainID    uint64
	SimpleTx   *transactions.SendTxArgs
	HopTx      *HopTxArgs
}

func (t *TransactionBridge) Value() *big.Int {
	if t.SimpleTx != nil && t.SimpleTx.To != nil {
		return t.SimpleTx.Value.ToInt()
	} else if t.HopTx != nil {
		return t.HopTx.Amount.ToInt()
	}

	return big.NewInt(0)
}

func (t *TransactionBridge) From() types.Address {
	if t.SimpleTx != nil && t.SimpleTx.To != nil {
		return t.SimpleTx.From
	} else if t.HopTx != nil {
		return t.HopTx.From
	}

	return types.HexToAddress("0x0")
}

func (t *TransactionBridge) To() types.Address {
	if t.SimpleTx != nil && t.SimpleTx.To != nil {
		return *t.SimpleTx.To
	} else if t.HopTx != nil {
		return types.Address(t.HopTx.Recipient)
	}

	return types.HexToAddress("0x0")
}

func (t *TransactionBridge) Data() types.HexBytes {
	if t.SimpleTx != nil && t.SimpleTx.To != nil {
		return t.SimpleTx.Data
	} else if t.HopTx != nil {
		return types.HexBytes("")
	}

	return types.HexBytes("")
}

type Bridge interface {
	Name() string
	Can(from *params.Network, to *params.Network, token *token.Token, balance *big.Int) (bool, error)
	CalculateFees(from, to *params.Network, token *token.Token, amountIn *big.Int, nativeTokenPrice, tokenPrice float64, gasPrice *big.Float) (*big.Int, *big.Int, error)
	EstimateGas(from *params.Network, to *params.Network, token *token.Token, amountIn *big.Int) (uint64, error)
	CalculateAmountOut(from, to *params.Network, amountIn *big.Int, symbol string) (*big.Int, error)
	Send(sendArgs *TransactionBridge, verifiedAccount *account.SelectedExtKey) (types.Hash, error)
}
