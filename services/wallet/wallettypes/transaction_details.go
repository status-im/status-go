package wallettypes

import (
	ethTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/status-im/status-go/eth-node/types"
	wallet_common "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/router/routes"
)

type TransactionData struct {
	TxArgs     *SendTxArgs
	Tx         *ethTypes.Transaction
	HashToSign types.Hash
	Signature  []byte
	SentHash   types.Hash
}

func (txd *TransactionData) IsTxPlaced() bool {
	return txd.SentHash != types.Hash(wallet_common.ZeroHash())
}

type RouterTransactionDetails struct {
	RouterPath     *routes.Path
	TxData         *TransactionData
	ApprovalTxData *TransactionData
}

func (rtd *RouterTransactionDetails) IsTxPlaced() bool {
	return rtd.TxData != nil && rtd.TxData.IsTxPlaced()
}

func (rtd *RouterTransactionDetails) IsApprovalPlaced() bool {
	return rtd.ApprovalTxData != nil && rtd.ApprovalTxData.IsTxPlaced()
}
