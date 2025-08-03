package alchemy

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/services/wallet/activity"
	ac "github.com/status-im/status-go/services/wallet/activity/common"
	wCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
)

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

var bridgeRouters = map[string]string{
	"0x0439e60f02a8900a951603950d8d4527f400c3f1": "MetaMask Bridge",
	"0x2796317b0ff8538f253012862c06787adfb8ceb6": "Synapse Bridge",
	"0x3666f603cc264fc1bea8f960f27b4bba7f960c58": "Hop ETH Bridge",
	"0x3ee18b2214aff97000d974cf647e7c347e8fa585": "Wormhole Token Bridge",
	"0x401f6c983ea34274ec46f84d70b31c151321188b": "Polygon PoS Bridge",
	"0x5c7bcd6e7de5423a257d81b442b9608c8ba9488d": "Across V2 HubPool",
	"0x66a71dcef29a0f5cb9584e92ed36966c3a3f437e": "LayerZero Endpoint",
	"0x72ce9c846789fdb6fc1f34ac4ad25dd9ef7031ef": "Arbitrum L1 Gateway Router",
	"0x8731d54e9d02c286767d56ac03e8037c07e01e98": "Stargate Router",
	"0x8898b472c54c31894e3b9bb83cea802a5d0e63c6": "Connext Bridge",
	"0x98f3c9e6e3face36baad05fe09d375ef1464288b": "Wormhole Core Bridge",
	"0x99c9fc46f92e8a1c7aa79fd23f77c7222dd3c98b": "Optimism L1 Standard Bridge",
	"0xa3a7b6f88361f48403514059f1f16c8e78d60eec": "Arbitrum L1 ERC20 Gateway",
	"0xb8901acb165ed027e32754e0ffe830802919727f": "Hop USDC Bridge",
}

func addressToKnown(address string) string {
	router, ok := dexRouters[strings.ToLower(address)]
	if ok {
		return router
	}

	router, ok = bridgeRouters[strings.ToLower(address)]
	if ok {
		return router
	}
	return address
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

func (a TransferAnalytics) isBridgeTransfer() bool {
	if len(a.externalTransfers) == 1 {
		to := a.externalTransfers[0].ToAddress.Hex()
		_, known := bridgeRouters[strings.ToLower(to)]
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

func sameHashTransfersToThirdpartyActivityEntries(txTransfers []Transfer, chainID uint64, accountAddress common.Address) []thirdparty.ActivityEntry {

	entries := make([]thirdparty.ActivityEntry, 0)

	analytics := analyzeTransfers(txTransfers, chainID, accountAddress)

	if analytics.isContractDeployment() {
		fmt.Println("ContractDeploymentDetected!! ")
		entries = transferToThirdpartyActivityEntries(analytics.externalTransfers[0], chainID, accountAddress)
	} else if analytics.isSwapTransfer() {
		fmt.Println("SwapDetected!! ", len(analytics.inboundNonZeroTransfers), "-", len(analytics.outboundNonZeroTransfers))
		if len(analytics.inboundNonZeroTransfers) == 1 && len(analytics.outboundNonZeroTransfers) == 1 {
			entries = append(entries, makeSwapThirdpartyActivityEntry(analytics.outboundNonZeroTransfers[0], analytics.inboundNonZeroTransfers[0], chainID))
		}
	} else if analytics.isBridgeTransfer() {
		fmt.Println("BridgeDetected!! ")
		if len(analytics.inboundNonZeroTransfers) == 0 && len(analytics.outboundNonZeroTransfers) == 1 {
			entries = append(entries, makeBridgeThirdpartyActivityEntry(analytics.outboundNonZeroTransfers[0], chainID))
		}
	} else if analytics.isErc20Transfer() {
		fmt.Println("ERC20Transfer!! ")
		entries = transferToThirdpartyActivityEntries(analytics.tokenTransfers[0], chainID, accountAddress)
	} else {
		fmt.Println("SimpleTransfer!! ")
		entries = transferToThirdpartyActivityEntries(txTransfers[0], chainID, accountAddress)
	}

	return entries
}

func sameHashTransfersToWalletActivityEntries(txTransfers []Transfer, chainID uint64, accountAddress common.Address) []activity.Entry {

	entries := make([]activity.Entry, 0)

	analytics := analyzeTransfers(txTransfers, chainID, accountAddress)

	if analytics.isContractDeployment() {
		fmt.Println("ContractDeploymentDetected!! ")
		entries = transferToWalletActivityEntries(analytics.externalTransfers[0], chainID, accountAddress)
	} else if analytics.isSwapTransfer() {
		fmt.Println("SwapDetected!! ", len(analytics.inboundNonZeroTransfers), "-", len(analytics.outboundNonZeroTransfers))
		if len(analytics.inboundNonZeroTransfers) == 1 && len(analytics.outboundNonZeroTransfers) == 1 {
			entries = append(entries, makeSwapThirdpartyActivityEntry(analytics.outboundNonZeroTransfers[0], analytics.inboundNonZeroTransfers[0], chainID))
		}
	} else if analytics.isBridgeTransfer() {
		fmt.Println("BridgeDetected!! ")
		if len(analytics.inboundNonZeroTransfers) == 0 && len(analytics.outboundNonZeroTransfers) == 1 {
			entries = append(entries, makeBridgeThirdpartyActivityEntry(analytics.outboundNonZeroTransfers[0], chainID))
		}
	} else if analytics.isErc20Transfer() {
		fmt.Println("ERC20Transfer!! ")
		entries = transferToThirdpartyActivityEntries(analytics.tokenTransfers[0], chainID, accountAddress)
	} else {
		fmt.Println("SimpleTransfer!! ")
		entries = transferToThirdpartyActivityEntries(txTransfers[0], chainID, accountAddress)
	}

	return entries
}

type transferTokenAndValue struct {
	Token ac.Token
	Value *hexutil.Big
}

func extractTransferTokenAndValue(t Transfer, chainID uint64) []transferTokenAndValue {

	transfersData := make([]transferTokenAndValue, 0, 1)

	switch t.Category {
	case TransferCategoryExternal:
		transfersData = append(transfersData, transferTokenAndValue{
			Token: ac.Token{
				TokenType: ac.Native,
				ChainID:   wCommon.ChainID(chainID),
			},
			Value: (*hexutil.Big)(t.RawContract.Value.Int),
		})
	case TransferCategoryErc20:
		transfersData = append(transfersData, transferTokenAndValue{
			Token: ac.Token{
				TokenType: ac.Erc20,
				ChainID:   wCommon.ChainID(chainID),
				Address:   *t.RawContract.Address,
			},
			Value: (*hexutil.Big)(t.RawContract.Value.Int),
		})
	case TransferCategoryErc721:
		transfersData = append(transfersData, transferTokenAndValue{
			Token: ac.Token{
				TokenType: ac.Erc721,
				ChainID:   wCommon.ChainID(chainID),
				Address:   *t.RawContract.Address,
				TokenID:   (*hexutil.Big)(t.TokenID.Int),
			},
		})
	case TransferCategoryErc1155:
		for _, m := range t.Erc1155Metadata {
			transfersData = append(transfersData, transferTokenAndValue{
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

func transferToThirdpartyActivityEntries(t Transfer, chainID uint64, accountAddress common.Address) []thirdparty.ActivityEntry {
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

	transfersTokenAndValue := extractTransferTokenAndValue(t, chainID)

	entries := make([]thirdparty.ActivityEntry, 0, len(transfersTokenAndValue))
	for _, td := range transfersTokenAndValue {
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

func transferToWalletctivityEntries(t Transfer, chainID uint64, accountAddress common.Address) []thirdparty.ActivityEntry {
	baseEntry := activity.Entry{
		timestamp: t.Metadata.BlockTimestamp.Unix(),
		sender:    t.FromAddress,
		recipient: t.ToAddress,
		// TxHash:      t.Hash,
		// BlockNumber: (*hexutil.Big)(t.BlockNum.Int),
		// TxStatus: ac.Success,
	}

	if t.ToAddress == nil {
		entry := baseEntry
		entry.ActivityType = ac.ContractDeploymentAT
		entry.ContractAddress = t.RawContract.Address
		entry.ChainIDOut = &chainID
		return []thirdparty.ActivityEntry{entry}
	}

	transfersTokenAndValue := extractTransferTokenAndValue(t, chainID)

	entries := make([]thirdparty.ActivityEntry, 0, len(transfersTokenAndValue))
	for _, td := range transfersTokenAndValue {
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

func makeSwapThirdpartyActivityEntry(outbound Transfer, inbound Transfer, chainID uint64) thirdparty.ActivityEntry {

	outboundTd := extractTransferTokenAndValue(outbound, chainID)[0]
	inboundTd := extractTransferTokenAndValue(inbound, chainID)[0]

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
}

func makeBridgeThirdpartyActivityEntry(outbound Transfer, chainID uint64) thirdparty.ActivityEntry {

	outboundTd := extractTransferTokenAndValue(outbound, chainID)[0]

	return thirdparty.ActivityEntry{
		Timestamp:       outbound.Metadata.BlockTimestamp.Unix(),
		ActivityType:    ac.BridgeAT,
		AmountOut:       outboundTd.Value,
		AmountIn:        outboundTd.Value,
		TokenOut:        &outboundTd.Token,
		TokenIn:         &outboundTd.Token,
		Sender:          outbound.FromAddress,
		Recipient:       outbound.ToAddress,
		ChainIDOut:      &chainID,
		ChainIDIn:       &chainID,
		ContractAddress: outbound.RawContract.Address,
		TxHash:          outbound.Hash,
		BlockNumber:     (*hexutil.Big)(outbound.BlockNum.Int),
		TxStatus:        ac.Success,
	}
}

func groupTransfersByHash(transfers []Transfer) map[common.Hash][]Transfer {
	// Group transfers by tx hash
	transfersPerHash := make(map[common.Hash][]Transfer)
	for _, t := range transfers {
		transfersPerHash[t.Hash] = append(transfersPerHash[t.Hash], t)
	}

	return transfersPerHash
}

func TransfersToThirdpartyActivityEntries(tt []Transfer, chainID uint64, accountAddress common.Address) []thirdparty.ActivityEntry {
	entries := make([]thirdparty.ActivityEntry, 0, len(tt))

	transfersPerHash := groupTransfersByHash(tt)

	for _, txTransfers := range transfersPerHash {
		newEntries := sameHashTransfersToThirdpartyActivityEntries(txTransfers, chainID, accountAddress)
		entries = append(entries, newEntries...)
	}

	return entries
}

func TransfersToWalletActivityEntries(tt []Transfer, chainID uint64, accountAddress common.Address) []activity.Entry {
	entries := make([]thirdparty.ActivityEntry, 0, len(tt))

	transfersPerHash := groupTransfersByHash(tt)

	for _, txTransfers := range transfersPerHash {
		newEntries := sameHashTransfersToWalletActivityEntries(txTransfers, chainID, accountAddress)
		entries = append(entries, newEntries...)
	}

	return entries
}
