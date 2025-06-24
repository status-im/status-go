package alchemy

import (
	"fmt"
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

func TransfersToCommon(tt []Transfer, chainID uint64, accountAddress common.Address) []thirdparty.ActivityEntry {
	entries := make([]thirdparty.ActivityEntry, 0, len(tt))

	// Group transfers by tx hash
	transfersPerHash := make(map[common.Hash][]Transfer)
	for _, t := range tt {
		transfersPerHash[t.Hash] = append(transfersPerHash[t.Hash], t)
	}

	for _, txTransfers := range transfersPerHash {
		entries = append(entries, processTxTransfers(txTransfers, chainID, accountAddress)...)
	}

	return entries
}

func processTxTransfers(txTransfers []Transfer, chainID uint64, accountAddress common.Address) []thirdparty.ActivityEntry {
	fmt.Println("processing tx transfers", len(txTransfers), txTransfers[0].Hash.Hex())
	// Notes about what Alchemy gives us:
	// - For contract deployments, we should get a single External transfer with toAddress nil.
	// - For native transfers, we should get a single External transfer with toAddress equal to the recipient
	// - For token transfers (simple transfer, mint, burn), we should get 1/many Token transfers with the same token contract address and
	// an External transfer with toAddress equal to the token contract address.
	// - For other contract interactions (swap, bridge, others), we should get 1/many Token transfers and an External transfer with
	// toAddress different from the token contract address

	// Simple transaction, just return the entry
	if len(txTransfers) == 1 {
		return transferToEntries(txTransfers[0], chainID, accountAddress)
	}

	externalTransfers := make([]Transfer, 0, len(txTransfers))
	tokenTransfers := make([]Transfer, 0, len(txTransfers))
	for _, t := range txTransfers {
		// Transfer(s) or contract interaction
		if t.Category == TransferCategoryExternal {
			externalTransfers = append(externalTransfers, t)
		} else {
			tokenTransfers = append(tokenTransfers, t)
		}
	}

	entries := make([]thirdparty.ActivityEntry, 0, len(txTransfers))
	if len(externalTransfers) == 0 {
		// No external transfers, so we can just return the token transfers
		for _, t := range tokenTransfers {
			entries = append(entries, transferToEntries(t, chainID, accountAddress)...)
		}
	} else if len(externalTransfers) == 1 {
		// One external transfer with at least one token transfer
		// If the external transfer points to the token contract, it's probably a token transfer
		// If the external transfer points to a different address, it's probably a contract interaction
		externalTransfer := externalTransfers[0]
		for _, t := range tokenTransfers {
			if externalTransfer.IsContractDeployment() {
				// Contract deployment
				entries = transferToEntries(externalTransfer, chainID, accountAddress)
				break
			} else if *externalTransfer.ToAddress != *t.RawContract.Address {
				// Contract interaction, could be a swap/bridge/etc
				contractInteractionEntry := transferToEntries(externalTransfer, chainID, accountAddress)
				contractInteractionEntry[0].ActivityType = ac.ContractInteractionAT
				contractInteractionEntry[0].ContractAddress = contractInteractionEntry[0].Recipient
				contractInteractionEntry[0].Recipient = nil
				entries = contractInteractionEntry
				break
			}
			// External transfer points to the token contract. Could be a transfer/mint/burn, we just return the token transfers
			entries = append(entries, transferToEntries(t, chainID, accountAddress)...)
		}
	} else {
		// Multiple external transfers in a single transaction is not a normal scenario
		// Don't add these to the list of entries
		fmt.Println("multiple external transfers in a single transaction", len(externalTransfers))
		fmt.Printf("external transfers %v\n", externalTransfers)
		fmt.Printf("token transfers %v\n", tokenTransfers)
	}

	return entries
}

type transferData struct {
	Token ac.Token
	Value *hexutil.Big
}

func transferToEntries(t Transfer, chainID uint64, accountAddress common.Address) []thirdparty.ActivityEntry {
	baseEntry := thirdparty.ActivityEntry{
		Timestamp:   t.Metadata.BlockTimestamp.Unix(),
		Sender:      t.FromAddress,
		Recipient:   t.ToAddress,
		TxHash:      t.Hash,
		BlockNumber: (*hexutil.Big)(t.BlockNum.Int),
		TxStatus:    ac.Success,
	}

	if t.ToAddress == nil {
		entry := baseEntry
		entry.ActivityType = ac.ContractDeploymentAT
		entry.ContractAddress = t.RawContract.Address
		entry.ChainIDOut = &chainID
		return []thirdparty.ActivityEntry{entry}
	}

	transfersData := make([]transferData, 0, 1)
	switch t.Category {
	case TransferCategoryExternal:
		transfersData = append(transfersData, transferData{
			Token: ac.Token{
				TokenType: ac.Native,
				ChainID:   wCommon.ChainID(chainID),
			},
			Value: (*hexutil.Big)(t.RawContract.Value.Int),
		})
	case TransferCategoryErc20:
		transfersData = append(transfersData, transferData{
			Token: ac.Token{
				TokenType: ac.Erc20,
				ChainID:   wCommon.ChainID(chainID),
				Address:   *t.RawContract.Address,
			},
			Value: (*hexutil.Big)(t.RawContract.Value.Int),
		})
	case TransferCategoryErc721:
		transfersData = append(transfersData, transferData{
			Token: ac.Token{
				TokenType: ac.Erc721,
				ChainID:   wCommon.ChainID(chainID),
				Address:   *t.RawContract.Address,
				TokenID:   (*hexutil.Big)(t.TokenID.Int),
			},
		})
	case TransferCategoryErc1155:
		for _, m := range t.Erc1155Metadata {
			transfersData = append(transfersData, transferData{
				Token: ac.Token{
					TokenType: ac.Erc1155,
					ChainID:   wCommon.ChainID(chainID),
					Address:   *t.RawContract.Address,
					TokenID:   (*hexutil.Big)(m.TokenID.Int),
				},
				Value: (*hexutil.Big)(m.Value.Int),
			})
		}
	}

	entries := make([]thirdparty.ActivityEntry, 0, len(transfersData))
	for _, td := range transfersData {
		entry := baseEntry

		if t.IsIncoming(accountAddress) {
			entry.ActivityType = ac.ReceiveAT
			entry.ChainIDIn = &chainID
			entry.TokenIn = &td.Token
			entry.AmountIn = td.Value
		} else {
			entry.ActivityType = ac.SendAT
			entry.ChainIDOut = &chainID
			entry.TokenOut = &td.Token
			entry.AmountOut = td.Value
		}
		entries = append(entries, entry)
	}
	return entries
}

func transferToContractInteractionEntry(t Transfer, chainID uint64) thirdparty.ActivityEntry {
	entry := thirdparty.ActivityEntry{
		Timestamp:       t.Metadata.BlockTimestamp.Unix(),
		ActivityType:    ac.ContractInteractionAT,
		Sender:          t.FromAddress,
		ChainIDOut:      &chainID,
		ContractAddress: t.ToAddress,
		TxHash:          t.Hash,
		BlockNumber:     (*hexutil.Big)(t.BlockNum.Int),
		TxStatus:        ac.Success,
	}

	return entry
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

func (t Transfer) IsContractDeployment() bool {
	return t.Category == TransferCategoryExternal && t.ToAddress == nil
}

func (t Transfer) IsIncoming(accountAddress common.Address) bool {
	return t.FromAddress != accountAddress
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
