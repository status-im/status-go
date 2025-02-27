package alchemy

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"

	ac "github.com/status-im/status-go/services/wallet/activity/common"
	"github.com/status-im/status-go/services/wallet/bigint"
	wCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
)

const getAssetTransfersMethod = "alchemy_getAssetTransfers"
const MaxAssetTransfersCount = 1000

type TransferCategory string

const (
	TransferCategoryExternal   TransferCategory = "external"
	TransferCategoryInternal   TransferCategory = "internal"
	TransferCategoryErc20      TransferCategory = "erc20"
	TransferCategoryErc721     TransferCategory = "erc721"
	TransferCategoryErc1155    TransferCategory = "erc1155"
	TransferCategorySpecialNft TransferCategory = "specialnft"
)

type TransferOrder string

const (
	TransferOrderOldToNew TransferOrder = "asc"
	TransferOrderNewToOld TransferOrder = "desc"
)

type GetAssetTransfersParams struct {
	FromBlock         *rpc.BlockNumber   `json:"fromBlock,omitempty"` // Defaults to "0x0"
	ToBlock           *rpc.BlockNumber   `json:"toBlock,omitempty"`   // Defaults to "latest"
	FromAddress       *common.Address    `json:"fromAddress,omitempty"`
	ToAddress         *common.Address    `json:"toAddress,omitempty"`
	ContractAddresses []common.Address   `json:"contractAddresses,omitempty"`
	Category          []TransferCategory `json:"category"`
	Order             TransferOrder      `json:"order"`
	WithMetadata      bool               `json:"withMetadata"`
	ExcludeZeroValue  bool               `json:"excludeZeroValue"`
	MaxCount          *hexutil.Big       `json:"maxCount"`
	PageKey           string             `json:"pageKey,omitempty"`
}

type GetAssetTranfersResponse struct {
	Transfers []Transfer `json:"transfers"`
	PageKey   string     `json:"pageKey"`
}

type transferData struct {
	Token ac.Token
	Value *hexutil.Big
}

func TransfersToCommon(tt []Transfer, isIncoming bool, chainID uint64) []thirdparty.ActivityEntry {
	entries := make([]thirdparty.ActivityEntry, 0, len(tt))

	cChainID := wCommon.ChainID(chainID)
	for _, t := range tt {
		if t.ToAddress == nil {
			entry := thirdparty.ActivityEntry{
				ActivityType: ac.ContractDeploymentAT,
				ChainIDOut:   &chainID,
				Timestamp:    t.Metadata.BlockTimestamp.Unix(),
				Sender:       t.FromAddress,
				Recipient:    t.ToAddress,
				TxHash:       t.Hash,
				BlockNumber:  (*hexutil.Big)(t.BlockNum.Int),
			}
			// Alchemy doesn't provide the contract address for contract deployment
			entries = append(entries, entry)
		} else {
			transfers := make([]transferData, 0, 1)
			switch t.Category {
			case TransferCategoryErc20:
				transfers = append(transfers, transferData{
					Token: ac.Token{
						ChainID:   cChainID,
						TokenType: ac.Erc20,
						Address:   *t.RawContract.Address,
					},
					Value: (*hexutil.Big)(t.RawContract.Value.Int),
				})
			case TransferCategoryErc721, TransferCategorySpecialNft:
				transfers = append(transfers, transferData{
					Token: ac.Token{
						ChainID:   cChainID,
						TokenType: ac.Erc721,
						Address:   *t.RawContract.Address,
						TokenID:   (*hexutil.Big)(t.TokenID.Int),
					},
					Value: (*hexutil.Big)(big.NewInt(1)),
				})
			case TransferCategoryErc1155:
				for _, m := range t.Erc1155Metadata {
					transfers = append(transfers, transferData{
						Token: ac.Token{
							ChainID:   cChainID,
							TokenType: ac.Erc1155,
							Address:   *t.RawContract.Address,
							TokenID:   (*hexutil.Big)(m.TokenID.Int),
						},
						Value: (*hexutil.Big)(m.Value.Int),
					})
				}
			default:
				transfers = append(transfers, transferData{
					Token: ac.Token{
						ChainID:   cChainID,
						TokenType: ac.Native,
					},
					Value: (*hexutil.Big)(t.RawContract.Value.Int),
				})
			}

			for _, transfer := range transfers {
				entry := thirdparty.ActivityEntry{
					Timestamp:   t.Metadata.BlockTimestamp.Unix(),
					Sender:      t.FromAddress,
					Recipient:   t.ToAddress,
					TxHash:      t.Hash,
					BlockNumber: (*hexutil.Big)(t.BlockNum.Int),
				}
				if isIncoming {
					entry.ActivityType = ac.ReceiveAT
					entry.ChainIDIn = &chainID
					entry.TokenIn = &transfer.Token
					entry.AmountIn = transfer.Value
				} else {
					entry.ActivityType = ac.SendAT
					entry.ChainIDOut = &chainID
					entry.TokenOut = &transfer.Token
					entry.AmountOut = transfer.Value
				}
				entries = append(entries, entry)
			}
		}
	}
	return entries
}

type Transfer struct {
	Category        TransferCategory     `json:"category"`
	BlockNum        *bigint.VarHexBigInt `json:"blockNum"`
	FromAddress     common.Address       `json:"from"`
	ToAddress       *common.Address      `json:"to,omitempty"`
	Erc1155Metadata []Erc1155Metadata    `json:"erc1155Metadata,omitempty"`
	TokenID         *bigint.VarHexBigInt `json:"tokenId"`
	Asset           string               `json:"asset"`
	UniqueID        string               `json:"uniqueId"`
	Hash            common.Hash          `json:"hash"`
	RawContract     RawContract          `json:"rawContract"`
	Metadata        Metadata             `json:"metadata"`
}

type Erc1155Metadata struct {
	TokenID *bigint.VarHexBigInt `json:"tokenId"`
	Value   *bigint.VarHexBigInt `json:"value"`
}

type RawContract struct {
	Value   *bigint.VarHexBigInt `json:"value"`   // nil if ERC721 or ERC1155 transfer
	Address *common.Address      `json:"address"` // nil if external or internal transfer
	Decimal *bigint.VarHexBigInt `json:"decimal"` // nil if not available in the contract
}

type Metadata struct {
	BlockTimestamp time.Time `json:"blockTimestamp"`
}
