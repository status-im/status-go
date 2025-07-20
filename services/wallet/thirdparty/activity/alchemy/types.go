package alchemy

import (
	"fmt"
	"strings"
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
		fmt.Println("--------------")
		fmt.Println("Input:")
		for _, transfer := range txTransfers {
			fmt.Printf("\tTransfer: %v\n", transfer)
		}

		// newEntries := processTxTransfers(txTransfers, chainID, accountAddress)
		newEntries := processTxTransfersV2(txTransfers, chainID, accountAddress)
		entries = append(entries, newEntries...)

		for _, entry := range newEntries {
			fmt.Println("Output:")
			fmt.Printf("\tEntry: %v \n", entry)
		}
	}

	return entries
}

type TransferAnalytics struct {
	transfersCount           int
	test                     bool
	externalTransfers        []Transfer
	tokenTransfers           []Transfer
	outboundNonZeroTransfers []Transfer
	inboundNonZeroTransfers  []Transfer
}

func (a TransferAnalytics) isContractDeployment() bool {
	return len(a.externalTransfers) == 1 && a.externalTransfers[0].IsContractDeployment()
}

func (a TransferAnalytics) isSwapTransfer() bool {

	if len(a.externalTransfers) == 1 {
		to := a.externalTransfers[0].ToAddress.Hex()
		_, known := dexRouters[strings.ToLower(to)]
		return known
	}
	return false
}

func (a TransferAnalytics) isErc20Transfer() bool {
	if len(a.externalTransfers) == 1 && len(a.tokenTransfers) == 1 {
		et := a.externalTransfers[0]
		tt := a.tokenTransfers[0]

		return *et.ToAddress == *tt.RawContract.Address
	}
	return false
}

func analyzeTransfers(txTransfers []Transfer, chainID uint64, accountAddress common.Address) TransferAnalytics {

	analytics := TransferAnalytics{test: true}
	analytics.test = true

	for _, transfer := range txTransfers {
		analytics.transfersCount += 1
		// Transfer(s) or contract interaction
		if transfer.Category == TransferCategoryExternal {
			analytics.externalTransfers = append(analytics.externalTransfers, transfer)
		} else {
			analytics.tokenTransfers = append(analytics.tokenTransfers, transfer)
		}

		if transfer.Value > 0 {
			if transfer.IsIncoming(accountAddress) {
				analytics.inboundNonZeroTransfers = append(analytics.inboundNonZeroTransfers, transfer)
			} else {
				analytics.outboundNonZeroTransfers = append(analytics.outboundNonZeroTransfers, transfer)
			}
		}
	}

	return analytics
}

func processTxTransfersV2(txTransfers []Transfer, chainID uint64, accountAddress common.Address) []thirdparty.ActivityEntry {

	// Simple transaction, just return the entry
	if len(txTransfers) == 1 {
		return transferToEntries(txTransfers[0], chainID, accountAddress)
	}

	entries := make([]thirdparty.ActivityEntry, 0)
	analytics := analyzeTransfers(txTransfers, chainID, accountAddress)

	if analytics.isContractDeployment() {
		entries = transferToEntries(analytics.externalTransfers[0], chainID, accountAddress)
	} else if analytics.isSwapTransfer() {
		fmt.Println("SwapDetected!! ", len(analytics.inboundNonZeroTransfers), "-", len(analytics.outboundNonZeroTransfers))
		if len(analytics.inboundNonZeroTransfers) == 1 && len(analytics.outboundNonZeroTransfers) == 1 {
			entries = append(entries, makeSwapEntry(analytics.outboundNonZeroTransfers[0], analytics.inboundNonZeroTransfers[0], chainID))
		}
	} else if analytics.isErc20Transfer() {
		entries = transferToEntries(analytics.tokenTransfers[0], chainID, accountAddress)
	}

	return entries
}

func processTxTransfers(txTransfers []Transfer, chainID uint64, accountAddress common.Address) []thirdparty.ActivityEntry {
	// fmt.Println("processing tx transfers", len(txTransfers), txTransfers[0].Hash.Hex())

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

func extractTransfersData(t Transfer, chainID uint64) []transferData {

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
	return transfersData
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

	transfersData := extractTransfersData(t, chainID)

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

func makeSwapEntry(outbound Transfer, inbound Transfer, chainID uint64) thirdparty.ActivityEntry {

	outboundTd := extractTransfersData(outbound, chainID)[0]
	inboundTd := extractTransfersData(inbound, chainID)[0]

	return thirdparty.ActivityEntry{
		Timestamp:       outbound.Metadata.BlockTimestamp.Unix(),
		ActivityType:    ac.SwapAT,
		AmountOut:       outboundTd.Value,
		AmountIn:        inboundTd.Value,
		TokenOut:        &outboundTd.Token,
		TokenIn:         &inboundTd.Token,
		Sender:          outbound.FromAddress,
		Recipient:       inbound.ToAddress,
		ChainIDOut:      &chainID,
		ChainIDIn:       &chainID,
		ContractAddress: outbound.RawContract.Address,
		TxHash:          outbound.Hash,
		BlockNumber:     (*hexutil.Big)(outbound.BlockNum.Int),
		TxStatus:        ac.Success,
	}

	// TxHash          common.Hash     `json:"txHash"`
	// BlockNumber     *hexutil.Big    `json:"blockNumber"`
	// TxStatus        ac.TxStatus     `json:"txStatus"`
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
	Value           float64              `json:"value,omitempty"`
	Erc1155Metadata []Erc1155Metadata    `json:"erc1155Metadata,omitempty"`
	TokenID         *bigint.VarHexBigInt `json:"tokenId"`
	Asset           string               `json:"asset"`
	UniqueID        string               `json:"uniqueId"`
	Hash            common.Hash          `json:"hash"`
	RawContract     RawContract          `json:"rawContract"`
	Metadata        Metadata             `json:"metadata"`
}

var dexRouters = map[string]string{
	"0x7a250d5630b4cf539739df2c5dacb4c659f2488d": "Uniswap V2",
	"0xe592427a0aece92de3edee1f18e0157c05861564": "Uniswap V3",
	"0x68b3465833fb72a70ecdf485e0e4c7bd8665fc45": "Uniswap V3",
	"0xd9e1ce17f2641f24ae83637ab66a2cca9c378b9f": "SushiSwap",
	"0x1111111254eeb25477b68fb85ed929f73a960582": "1inch",
	"0x1111111254fb6c44bac0bed2854e76f90643097d": "1inch",
	"0xba12222222228d8ba445958a75a0704d566bf2c8": "Balancer V2",
	"0x99a58482bef6a0b58cf15a81c4b666b64ab56abc": "Curve",
	"0x881d40237659c251811cec9c364ef91dc08d300c": "MetaMask Swap",
}

func addressToKnownDex(address string) string {
	router, ok := dexRouters[strings.ToLower(address)]
	if !ok {
		return address
	}
	return router
}

func (t Transfer) String() string {
	return fmt.Sprintf("%s %8s %5s %13.7f %s->%s, %s", t.Hash.TerminalString(), t.Category, t.Asset, t.Value, t.FromAddress.Hex(), addressToKnownDex(t.ToAddress.Hex()), t.RawContract.Address)
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
