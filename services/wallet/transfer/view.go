package transfer

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// View stores only fields used by a client and ensures that all relevant fields are
// encoded in hex.
type View struct {
	ID                   common.Hash    `json:"id"`
	Type                 Type           `json:"type"`
	Address              common.Address `json:"address"`
	BlockNumber          *hexutil.Big   `json:"blockNumber"`
	BlockHash            common.Hash    `json:"blockhash"`
	Timestamp            hexutil.Uint64 `json:"timestamp"`
	GasPrice             *hexutil.Big   `json:"gasPrice"`
	MaxFeePerGas         *hexutil.Big   `json:"maxFeePerGas"`
	MaxPriorityFeePerGas *hexutil.Big   `json:"maxPriorityFeePerGas"`
	EffectiveTip         *hexutil.Big   `json:"effectiveTip"`
	EffectiveGasPrice    *hexutil.Big   `json:"effectiveGasPrice"`
	GasLimit             hexutil.Uint64 `json:"gasLimit"`
	GasUsed              hexutil.Uint64 `json:"gasUsed"`
	Nonce                hexutil.Uint64 `json:"nonce"`
	TxStatus             hexutil.Uint64 `json:"txStatus"`
	Input                hexutil.Bytes  `json:"input"`
	TxHash               common.Hash    `json:"txHash"`
	Value                *hexutil.Big   `json:"value"`   // Only used for Type ethTransfer and erc20Transfer
	TokenID              *hexutil.Big   `json:"tokenId"` // Only used for Type erc721Transfer
	From                 common.Address `json:"from"`
	To                   common.Address `json:"to"`
	Contract             common.Address `json:"contract"`
	NetworkID            uint64         `json:"networkId"`
	MultiTransactionID   int64          `json:"multiTransactionID"`
	BaseGasFees          string         `json:"base_gas_fee"`
}

func castToTransferViews(transfers []Transfer) []View {
	views := make([]View, len(transfers))
	for i := range transfers {
		views[i] = CastToTransferView(transfers[i])
	}
	return views
}

func CastToTransferView(t Transfer) View {
	view := View{}
	view.ID = t.ID
	view.Type = getFixedTransferType(t)
	view.Address = t.Address
	view.BlockNumber = (*hexutil.Big)(t.BlockNumber)
	view.BlockHash = t.BlockHash
	view.Timestamp = hexutil.Uint64(t.Timestamp)
	view.GasPrice = (*hexutil.Big)(t.Transaction.GasPrice())
	if t.BaseGasFees != "" {
		baseFee := new(big.Int)
		baseFee.SetString(t.BaseGasFees[2:], 16)
		tip := t.Transaction.EffectiveGasTipValue(baseFee)

		view.EffectiveTip = (*hexutil.Big)(tip)
		price := new(big.Int).Add(baseFee, tip)
		view.EffectiveGasPrice = (*hexutil.Big)(price)
	}
	view.MaxFeePerGas = (*hexutil.Big)(t.Transaction.GasFeeCap())
	view.MaxPriorityFeePerGas = (*hexutil.Big)(t.Transaction.GasTipCap())
	view.GasLimit = hexutil.Uint64(t.Transaction.Gas())
	view.GasUsed = hexutil.Uint64(t.Receipt.GasUsed)
	view.BaseGasFees = t.BaseGasFees
	view.Nonce = hexutil.Uint64(t.Transaction.Nonce())
	view.TxStatus = hexutil.Uint64(t.Receipt.Status)
	view.Input = hexutil.Bytes(t.Transaction.Data())
	view.TxHash = t.Transaction.Hash()
	view.NetworkID = t.NetworkID

	value := new(hexutil.Big)
	tokenId := new(hexutil.Big)

	switch view.Type {
	case ethTransfer:
		view.From = t.From
		if t.Transaction.To() != nil {
			view.To = *t.Transaction.To()
		}
		value = (*hexutil.Big)(t.Transaction.Value())
		view.Contract = t.Receipt.ContractAddress
	case erc20Transfer:
		view.Contract = t.Log.Address
		from, to, valueInt := parseErc20Log(t.Log)
		view.From, view.To, value = from, to, (*hexutil.Big)(valueInt)
	case erc721Transfer:
		view.Contract = t.Log.Address
		from, to, tokenIdInt := parseErc721Log(t.Log)
		view.From, view.To, tokenId = from, to, (*hexutil.Big)(tokenIdInt)
	}

	view.MultiTransactionID = int64(t.MultiTransactionID)
	view.Value = value
	view.TokenID = tokenId

	return view
}

func getFixedTransferType(tx Transfer) Type {
	// erc721 transfers share signature with erc20 ones, so they are both (cached and new)
	// categorized as erc20 by the Downloader. We fix this on the fly for the moment, until
	// the Downloader gets refactored.
	if tx.Type == erc20Transfer {
		switch len(tx.Log.Topics) {
		case erc20TransferEventIndexedParameters:
			// do nothing
		case erc721TransferEventIndexedParameters:
			return erc721Transfer
		default:
			return unknownTokenTransfer
		}
	}
	return tx.Type
}

func parseErc20Log(ethlog *types.Log) (from, to common.Address, amount *big.Int) {
	amount = new(big.Int)
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
	amount.SetBytes(ethlog.Data)

	return
}

func parseErc721Log(ethlog *types.Log) (from, to common.Address, tokenId *big.Int) {
	tokenId = new(big.Int)
	if len(ethlog.Topics) < 4 {
		log.Warn("not enough topics for erc721 transfer", "topics", ethlog.Topics)
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
	if len(ethlog.Topics[3]) != 32 {
		log.Warn("fourth topic is not 32 byte tokenId", "topic", ethlog.Topics[3])
		return
	}
	copy(from[:], ethlog.Topics[1][12:])
	copy(to[:], ethlog.Topics[2][12:])
	tokenId.SetBytes(ethlog.Topics[3][:])

	return
}
