// Moved here because transactions package depends on accounts package which
// depends on appdatabase where this functionality is needed
package common

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/status-im/status-go/internal/logutils"
)

// Type type of transaction
type Type string

// Log Event type
type EventType string

const (
	// Transaction types
	EthTransfer        Type = "eth"
	Erc20Transfer      Type = "erc20"
	Erc721Transfer     Type = "erc721"
	Erc1155Transfer    Type = "erc1155"
	UniswapV2Swap      Type = "uniswapV2Swap"
	UniswapV3Swap      Type = "uniswapV3Swap"
	HopBridgeFrom      Type = "HopBridgeFrom"
	HopBridgeTo        Type = "HopBridgeTo"
	unknownTransaction Type = "unknown"

	// Event types
	WETHDepositEventType                      EventType = "wethDepositEvent"
	WETHWithdrawalEventType                   EventType = "wethWithdrawalEvent"
	Erc20TransferEventType                    EventType = "erc20Event"
	Erc721TransferEventType                   EventType = "erc721Event"
	Erc1155TransferSingleEventType            EventType = "erc1155SingleEvent"
	Erc1155TransferBatchEventType             EventType = "erc1155BatchEvent"
	UniswapV2SwapEventType                    EventType = "uniswapV2SwapEvent"
	UniswapV3SwapEventType                    EventType = "uniswapV3SwapEvent"
	HopBridgeTransferSentToL2EventType        EventType = "hopBridgeTransferSentToL2Event"
	HopBridgeTransferFromL1CompletedEventType EventType = "hopBridgeTransferFromL1CompletedEvent"
	HopBridgeWithdrawalBondedEventType        EventType = "hopBridgeWithdrawalBondedEvent"
	HopBridgeTransferSentEventType            EventType = "hopBridgeTransferSentEvent"
	UnknownEventType                          EventType = "unknownEvent"

	// Deposit (index_topic_1 address dst, uint256 wad)
	wethDepositEventSignature = "Deposit(address,uint256)"
	// Withdrawal (index_topic_1 address src, uint256 wad)
	wethWithdrawalEventSignature = "Withdrawal(address,uint256)"

	// Transfer (index_topic_1 address from, index_topic_2 address to, uint256 value)
	// Transfer (index_topic_1 address from, index_topic_2 address to, index_topic_3 uint256 tokenId)
	Erc20_721TransferEventSignature     = "Transfer(address,address,uint256)"
	Erc1155TransferSingleEventSignature = "TransferSingle(address,address,address,uint256,uint256)"    // operator, from, to, id, value
	Erc1155TransferBatchEventSignature  = "TransferBatch(address,address,address,uint256[],uint256[])" // operator, from, to, ids, values

	erc20TransferEventIndexedParameters   = 3 // signature, from, to
	erc721TransferEventIndexedParameters  = 4 // signature, from, to, tokenId
	erc1155TransferEventIndexedParameters = 4 // signature, operator, from, to (id, value are not indexed)

	// Swap (index_topic_1 address sender, uint256 amount0In, uint256 amount1In, uint256 amount0Out, uint256 amount1Out, index_topic_2 address to)
	uniswapV2SwapEventSignature = "Swap(address,uint256,uint256,uint256,uint256,address)" // also used by SushiSwap
	// Swap (index_topic_1 address sender, index_topic_2 address recipient, int256 amount0, int256 amount1, uint160 sqrtPriceX96, uint128 liquidity, int24 tick)
	uniswapV3SwapEventSignature = "Swap(address,address,int256,int256,uint160,uint128,int24)"

	// TransferSentToL2 (index_topic_1 uint256 chainId, index_topic_2 address recipient, uint256 amount, uint256 amountOutMin, uint256 deadline, index_topic_3 address relayer, uint256 relayerFee)
	hopBridgeTransferSentToL2EventSignature = "TransferSentToL2(uint256,address,uint256,uint256,uint256,address,uint256)"
	// TransferFromL1Completed (index_topic_1 address recipient, uint256 amount, uint256 amountOutMin, uint256 deadline, index_topic_2 address relayer, uint256 relayerFee)
	HopBridgeTransferFromL1CompletedEventSignature = "TransferFromL1Completed(address,uint256,uint256,uint256,address,uint256)"
	// WithdrawalBonded (index_topic_1 bytes32 transferID, uint256 amount)
	hopBridgeWithdrawalBondedEventSignature = "WithdrawalBonded(bytes32,uint256)"
	// TransferSent (index_topic_1 bytes32 transferID, index_topic_2 uint256 chainId, index_topic_3 address recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee, uint256 index, uint256 amountOutMin, uint256 deadline)
	hopBridgeTransferSentEventSignature = "TransferSent(bytes32,uint256,address,uint256,bytes32,uint256,uint256,uint256,uint256)"
)

var (
	// MaxUint256 is the maximum value that can be represented by a uint256.
	MaxUint256 = new(big.Int).Sub(new(big.Int).Lsh(common.Big1, 256), common.Big1)
)

// Detect event type for a cetain item from the Events Log
func GetEventType(log *types.Log) EventType {
	wethDepositEventSignatureHash := GetEventSignatureHash(wethDepositEventSignature)
	wethWithdrawalEventSignatureHash := GetEventSignatureHash(wethWithdrawalEventSignature)
	erc20_721TransferEventSignatureHash := GetEventSignatureHash(Erc20_721TransferEventSignature)
	erc1155TransferSingleEventSignatureHash := GetEventSignatureHash(Erc1155TransferSingleEventSignature)
	erc1155TransferBatchEventSignatureHash := GetEventSignatureHash(Erc1155TransferBatchEventSignature)
	uniswapV2SwapEventSignatureHash := GetEventSignatureHash(uniswapV2SwapEventSignature)
	uniswapV3SwapEventSignatureHash := GetEventSignatureHash(uniswapV3SwapEventSignature)
	hopBridgeTransferSentToL2EventSignatureHash := GetEventSignatureHash(hopBridgeTransferSentToL2EventSignature)
	hopBridgeTransferFromL1CompletedEventSignatureHash := GetEventSignatureHash(HopBridgeTransferFromL1CompletedEventSignature)
	hopBridgeWithdrawalBondedEventSignatureHash := GetEventSignatureHash(hopBridgeWithdrawalBondedEventSignature)
	hopBridgeTransferSentEventSignatureHash := GetEventSignatureHash(hopBridgeTransferSentEventSignature)

	if len(log.Topics) > 0 {
		switch log.Topics[0] {
		case wethDepositEventSignatureHash:
			return WETHDepositEventType
		case wethWithdrawalEventSignatureHash:
			return WETHWithdrawalEventType
		case erc20_721TransferEventSignatureHash:
			switch len(log.Topics) {
			case erc20TransferEventIndexedParameters:
				return Erc20TransferEventType
			case erc721TransferEventIndexedParameters:
				return Erc721TransferEventType
			}
		case erc1155TransferSingleEventSignatureHash:
			return Erc1155TransferSingleEventType
		case erc1155TransferBatchEventSignatureHash:
			return Erc1155TransferBatchEventType
		case uniswapV2SwapEventSignatureHash:
			return UniswapV2SwapEventType
		case uniswapV3SwapEventSignatureHash:
			return UniswapV3SwapEventType
		case hopBridgeTransferSentToL2EventSignatureHash:
			return HopBridgeTransferSentToL2EventType
		case hopBridgeTransferFromL1CompletedEventSignatureHash:
			return HopBridgeTransferFromL1CompletedEventType
		case hopBridgeWithdrawalBondedEventSignatureHash:
			return HopBridgeWithdrawalBondedEventType
		case hopBridgeTransferSentEventSignatureHash:
			return HopBridgeTransferSentEventType
		}
	}

	return UnknownEventType
}

func EventTypeToSubtransactionType(eventType EventType) Type {
	switch eventType {
	case Erc20TransferEventType:
		return Erc20Transfer
	case Erc721TransferEventType:
		return Erc721Transfer
	case Erc1155TransferSingleEventType, Erc1155TransferBatchEventType:
		return Erc1155Transfer
	case UniswapV2SwapEventType:
		return UniswapV2Swap
	case UniswapV3SwapEventType:
		return UniswapV3Swap
	case HopBridgeTransferSentToL2EventType, HopBridgeTransferSentEventType:
		return HopBridgeFrom
	case HopBridgeTransferFromL1CompletedEventType, HopBridgeWithdrawalBondedEventType:
		return HopBridgeTo
	}

	return unknownTransaction
}

func ParseErc20TransferLog(ethlog *types.Log) (from, to common.Address, amount *big.Int) {
	amount = new(big.Int)
	if len(ethlog.Topics) < erc20TransferEventIndexedParameters {
		logutils.ZapLogger().Warn("not enough topics for erc20 transfer", zap.Stringers("topics", ethlog.Topics))
		return
	}
	var err error
	from, to, err = getFromToAddresses(*ethlog)
	if err != nil {
		logutils.ZapLogger().Error("log_parser::ParseErc20TransferLog", zap.Error(err))
		return
	}

	if len(ethlog.Data) != 32 {
		logutils.ZapLogger().Warn("data is not padded to 32 byts big int", zap.Binary("data", ethlog.Data))
		return
	}
	amount.SetBytes(ethlog.Data)

	return
}

func ParseErc721TransferLog(ethlog *types.Log) (from, to common.Address, tokenID *big.Int) {
	tokenID = new(big.Int)
	if len(ethlog.Topics) < erc721TransferEventIndexedParameters {
		logutils.ZapLogger().Warn("not enough topics for erc721 transfer", zap.Stringers("topics", ethlog.Topics))
		return
	}

	var err error
	from, to, err = getFromToAddresses(*ethlog)
	if err != nil {
		logutils.ZapLogger().Error("log_parser::ParseErc721TransferLog", zap.Error(err))
		return
	}
	tokenID.SetBytes(ethlog.Topics[3][:])

	return
}

func GetLogSubTxID(log types.Log) common.Hash {
	// Get unique ID by using TxHash and log index
	index := [4]byte{}
	binary.BigEndian.PutUint32(index[:], uint32(log.Index))
	return crypto.Keccak256Hash(log.TxHash.Bytes(), index[:])
}

func getLogSubTxIDWithTokenIDIndex(log types.Log, tokenIDIdx uint16) common.Hash {
	// Get unique ID by using TxHash, log index and extra bytes (token id index for ERC1155 TransferBatch)
	index := [4]byte{}
	value := uint32(log.Index&0x0000FFFF) | (uint32(tokenIDIdx) << 16) // log index should not exceed uint16 max value
	binary.BigEndian.PutUint32(index[:], value)
	return crypto.Keccak256Hash(log.TxHash.Bytes(), index[:])
}

func checkTopicsLength(ethlog types.Log, startIdx, endIdx int) (err error) {
	for i := startIdx; i < endIdx; i++ {
		if len(ethlog.Topics[i]) != common.HashLength {
			err = fmt.Errorf("topic %d is not padded to %d byte address, topic=%s", i, common.HashLength, ethlog.Topics[i])
			logutils.ZapLogger().Error("log_parser::checkTopicsLength", zap.Error(err))
			return
		}
	}
	return
}

func getFromToAddresses(ethlog types.Log) (from, to common.Address, err error) {
	eventType := GetEventType(&ethlog)
	addressIdx := common.HashLength - common.AddressLength
	switch eventType {
	case Erc1155TransferSingleEventType, Erc1155TransferBatchEventType:
		err = checkTopicsLength(ethlog, 2, 4)
		if err != nil {
			return
		}
		copy(from[:], ethlog.Topics[2][addressIdx:])
		copy(to[:], ethlog.Topics[3][addressIdx:])
		return

	case Erc20TransferEventType, Erc721TransferEventType, UniswapV2SwapEventType, UniswapV3SwapEventType, HopBridgeTransferFromL1CompletedEventType:
		err = checkTopicsLength(ethlog, 1, 3)
		if err != nil {
			return
		}
		copy(from[:], ethlog.Topics[1][addressIdx:])
		copy(to[:], ethlog.Topics[2][addressIdx:])
		return
	}

	return from, to, fmt.Errorf("unsupported event type to get from/to adddresses %s", eventType)
}
func ParseTransferLog(ethlog types.Log) (from, to common.Address, txIDs []common.Hash, tokenIDs, values []*big.Int, err error) {
	eventType := GetEventType(&ethlog)

	switch eventType {
	case Erc20TransferEventType:
		var amount *big.Int
		from, to, amount = ParseErc20TransferLog(&ethlog)
		txIDs = append(txIDs, GetLogSubTxID(ethlog))
		values = append(values, amount)
		return
	case Erc721TransferEventType:
		var tokenID *big.Int
		from, to, tokenID = ParseErc721TransferLog(&ethlog)
		txIDs = append(txIDs, GetLogSubTxID(ethlog))
		tokenIDs = append(tokenIDs, tokenID)
		values = append(values, big.NewInt(1))
		return
	case Erc1155TransferSingleEventType, Erc1155TransferBatchEventType:
		_, from, to, tokenIDs, values, err = ParseErc1155TransferLog(&ethlog, eventType)
		for i := range tokenIDs {
			txIDs = append(txIDs, getLogSubTxIDWithTokenIDIndex(ethlog, uint16(i)))
		}
		return
	}

	return from, to, txIDs, tokenIDs, values, fmt.Errorf("unsupported event type in log_parser::ParseTransferLogs %s", eventType)
}

func ParseErc1155TransferLog(ethlog *types.Log, evType EventType) (operator, from, to common.Address, ids, amounts []*big.Int, err error) {
	if len(ethlog.Topics) < erc1155TransferEventIndexedParameters {
		err = fmt.Errorf("not enough topics for erc1155 transfer %s, %v", "topics", ethlog.Topics)
		logutils.ZapLogger().Error("log_parser::ParseErc1155TransferLog", zap.Error(err))
		return
	}

	err = checkTopicsLength(*ethlog, 1, erc1155TransferEventIndexedParameters)
	if err != nil {
		return
	}

	addressIdx := common.HashLength - common.AddressLength
	copy(operator[:], ethlog.Topics[1][addressIdx:])
	from, to, err = getFromToAddresses(*ethlog)
	if err != nil {
		logutils.ZapLogger().Error("log_parser::ParseErc1155TransferLog", zap.Error(err))
		return
	}

	if len(ethlog.Data) == 0 || len(ethlog.Data)%(common.HashLength*2) != 0 {
		err = fmt.Errorf("data is not padded to 64 bytes %s, %v", "data", ethlog.Data)
		logutils.ZapLogger().Error("log_parser::ParseErc1155TransferLog", zap.Error(err))
		return
	}

	if evType == Erc1155TransferSingleEventType {
		ids = append(ids, new(big.Int).SetBytes(ethlog.Data[:common.HashLength]))
		amounts = append(amounts, new(big.Int).SetBytes(ethlog.Data[common.HashLength:]))
		logutils.ZapLogger().Debug("log_parser::ParseErc1155TransferSingleLog",
			zap.Any("ids", ids),
			zap.Any("amounts", amounts),
		)
	} else {
		// idTypeSize := new(big.Int).SetBytes(ethlog.Data[:common.HashLength]).Uint64() // Left for knowledge
		// valueTypeSize := new(big.Int).SetBytes(ethlog.Data[common.HashLength : common.HashLength*2]).Uint64() // Left for knowledge
		idsArraySize := new(big.Int).SetBytes(ethlog.Data[common.HashLength*2 : common.HashLength*2+common.HashLength]).Uint64()

		initialOffset := common.HashLength*2 + common.HashLength
		for i := 0; i < int(idsArraySize); i++ {
			ids = append(ids, new(big.Int).SetBytes(ethlog.Data[initialOffset+i*common.HashLength:initialOffset+(i+1)*common.HashLength]))
		}
		valuesArraySize := new(big.Int).SetBytes(ethlog.Data[initialOffset+int(idsArraySize)*common.HashLength : initialOffset+int(idsArraySize+1)*common.HashLength]).Uint64()

		if idsArraySize != valuesArraySize {
			err = fmt.Errorf("ids and values sizes don't match %d, %d", idsArraySize, valuesArraySize)
			logutils.ZapLogger().Error("log_parser::ParseErc1155TransferBatchLog", zap.Error(err))
			return
		}

		initialOffset = initialOffset + int(idsArraySize+1)*common.HashLength
		for i := 0; i < int(valuesArraySize); i++ {
			amounts = append(amounts, new(big.Int).SetBytes(ethlog.Data[initialOffset+i*common.HashLength:initialOffset+(i+1)*common.HashLength]))
			logutils.ZapLogger().Debug("log_parser::ParseErc1155TransferBatchLog",
				zap.Any("id", ids[i]),
				zap.Any("amount", amounts[i]),
			)
		}
	}

	return
}

func GetEventSignatureHash(signature string) common.Hash {
	return crypto.Keccak256Hash([]byte(signature))
}

func ExtractTokenTransferData(dbEntryType Type, log *types.Log, tx *types.Transaction) (correctType Type, tokenAddress *common.Address, txFrom *common.Address, txTo *common.Address) {
	// erc721 transfers share signature with erc20 ones, so they both used to be categorized as erc20
	// by the Downloader. We fix this here since they might be mis-categorized in the db.
	if dbEntryType == Erc20Transfer {
		eventType := GetEventType(log)
		correctType = EventTypeToSubtransactionType(eventType)
	} else {
		correctType = dbEntryType
	}

	switch correctType {
	case Erc20Transfer:
		tokenAddress = new(common.Address)
		*tokenAddress = log.Address
		from, to, _ := ParseErc20TransferLog(log)
		txFrom = &from
		txTo = &to
	case Erc721Transfer:
		tokenAddress = new(common.Address)
		*tokenAddress = log.Address
		from, to, _ := ParseErc721TransferLog(log)
		txFrom = &from
		txTo = &to
	case Erc1155Transfer:
		tokenAddress = new(common.Address)
		*tokenAddress = log.Address
		_, from, to, _, _, err := ParseErc1155TransferLog(log, Erc1155TransferSingleEventType) // from/to extraction is the same for single and batch
		if err != nil {
			return
		}
		txFrom = &from
		txTo = &to
	}

	return
}
