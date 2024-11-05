package pathprocessor

import (
	"math/big"

	"github.com/status-im/status-go/eth-node/types"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/wallettypes"
)

type MultipathProcessorTxArgs struct {
	Name              string `json:"bridgeName"`
	ChainID           uint64
	TransferTx        *wallettypes.SendTxArgs
	HopTx             *HopBridgeTxArgs
	CbridgeTx         *CelerBridgeTxArgs
	ERC721TransferTx  *ERC721TxArgs
	ERC1155TransferTx *ERC1155TxArgs
	SwapTx            *SwapParaswapTxArgs
}

func (t *MultipathProcessorTxArgs) Value() *big.Int {
	if t.TransferTx != nil && t.TransferTx.To != nil {
		return t.TransferTx.Value.ToInt()
	} else if t.HopTx != nil {
		return t.HopTx.Amount.ToInt()
	} else if t.CbridgeTx != nil {
		return t.CbridgeTx.Amount.ToInt()
	} else if t.ERC721TransferTx != nil {
		return big.NewInt(1)
	} else if t.ERC1155TransferTx != nil {
		return t.ERC1155TransferTx.Amount.ToInt()
	}

	return walletCommon.ZeroBigIntValue()
}

func (t *MultipathProcessorTxArgs) From() types.Address {
	if t.TransferTx != nil && t.TransferTx.To != nil {
		return t.TransferTx.From
	} else if t.HopTx != nil {
		return t.HopTx.From
	} else if t.CbridgeTx != nil {
		return t.CbridgeTx.From
	} else if t.ERC721TransferTx != nil {
		return t.ERC721TransferTx.From
	} else if t.ERC1155TransferTx != nil {
		return t.ERC1155TransferTx.From
	}

	return types.HexToAddress("0x0")
}

func (t *MultipathProcessorTxArgs) To() types.Address {
	if t.TransferTx != nil && t.TransferTx.To != nil {
		return *t.TransferTx.To
	} else if t.HopTx != nil {
		return types.Address(t.HopTx.Recipient)
	} else if t.CbridgeTx != nil {
		return types.Address(t.HopTx.Recipient)
	} else if t.ERC721TransferTx != nil {
		return types.Address(t.ERC721TransferTx.Recipient)
	} else if t.ERC1155TransferTx != nil {
		return types.Address(t.ERC1155TransferTx.Recipient)
	}

	return types.HexToAddress("0x0")
}

func (t *MultipathProcessorTxArgs) Data() types.HexBytes {
	if t.TransferTx != nil && t.TransferTx.To != nil {
		return t.TransferTx.Data
	} else if t.HopTx != nil {
		return types.HexBytes("")
	} else if t.CbridgeTx != nil {
		return types.HexBytes("")
	} else if t.ERC721TransferTx != nil {
		return types.HexBytes("")
	} else if t.ERC1155TransferTx != nil {
		return types.HexBytes("")
	}

	return types.HexBytes("")
}
