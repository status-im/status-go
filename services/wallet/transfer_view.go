package wallet

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

func castToTransferViews(transfers []Transfer) []TransferView {
	views := make([]TransferView, len(transfers))
	for i := range transfers {
		views[i] = castToTransferView(transfers[i])
	}
	return views
}

func castToTransferView(t Transfer) TransferView {
	view := TransferView{}
	view.ID = t.ID
	view.Type = t.Type
	view.Address = t.Address
	view.BlockNumber = (*hexutil.Big)(t.BlockNumber)
	view.BlockHash = t.BlockHash
	view.Timestamp = hexutil.Uint64(t.Timestamp)
	view.GasPrice = (*hexutil.Big)(t.Transaction.GasPrice())
	view.GasLimit = hexutil.Uint64(t.Transaction.Gas())
	view.GasUsed = hexutil.Uint64(t.Receipt.GasUsed)
	view.Nonce = hexutil.Uint64(t.Transaction.Nonce())
	view.TxStatus = hexutil.Uint64(t.Receipt.Status)
	view.Input = hexutil.Bytes(t.Transaction.Data())
	view.TxHash = t.Transaction.Hash()
	switch t.Type {
	case ethTransfer:
		view.From = t.From
		if t.Transaction.To() != nil {
			view.To = *t.Transaction.To()
		}
		view.Value = (*hexutil.Big)(t.Transaction.Value())
		view.Contract = t.Receipt.ContractAddress
	case erc20Transfer:
		view.Contract = t.Log.Address
		from, to, amount := parseLog(t.Log)
		view.From, view.To, view.Value = from, to, (*hexutil.Big)(amount)
	}
	return view
}

func parseLog(ethlog *types.Log) (from, to common.Address, amount *big.Int) {
	if len(ethlog.Topics) < 3 {
		log.Warn("not enough topics for erc20 transfer", "topics", ethlog.Topics)
		return
	}
	if len(ethlog.Topics[1]) != 32 {
		log.Warn("second topic is not padded to 32 byte address", "topic", ethlog.Topics[1])
		return
	}
	if len(ethlog.Topics[2]) != 32 {
		log.Warn("third topic is not padded to 32 byte address", "topic", ethlog.Topics[2])
		return
	}
	copy(from[:], ethlog.Topics[1][12:])
	copy(to[:], ethlog.Topics[2][12:])
	if len(ethlog.Data) != 32 {
		log.Warn("data is not padded to 32 byts big int", "data", ethlog.Data)
		return
	}
	amount = new(big.Int).SetBytes(ethlog.Data)
	return
}

// TransferView stores only fields used by a client and ensures that all relevant fields are
// encoded in hex.
type TransferView struct {
	ID          common.Hash    `json:"id"`
	Type        TransferType   `json:"type"`
	Address     common.Address `json:"address"`
	BlockNumber *hexutil.Big   `json:"blockNumber"`
	BlockHash   common.Hash    `json:"blockhash"`
	Timestamp   hexutil.Uint64 `json:"timestamp"`
	GasPrice    *hexutil.Big   `json:"gasPrice"`
	GasLimit    hexutil.Uint64 `json:"gasLimit"`
	GasUsed     hexutil.Uint64 `json:"gasUsed"`
	Nonce       hexutil.Uint64 `json:"nonce"`
	TxStatus    hexutil.Uint64 `json:"txStatus"`
	Input       hexutil.Bytes  `json:"input"`
	TxHash      common.Hash    `json:"txHash"`
	Value       *hexutil.Big   `json:"value"`
	From        common.Address `json:"from"`
	To          common.Address `json:"to"`
	Contract    common.Address `json:"contract"`
}
