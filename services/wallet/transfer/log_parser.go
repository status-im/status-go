package transfer

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// Type type of transaction
type Type string

// Log Event type
type EventType string

const (
	// Transaction types
	ethTransfer        Type = "eth"
	erc20Transfer      Type = "erc20"
	erc721Transfer     Type = "erc721"
	uniswapV2Swap      Type = "uniswapV2Swap"
	unknownTransaction Type = "unknown"

	// Event types
	erc20TransferEventType  EventType = "erc20Event"
	erc721TransferEventType EventType = "erc721Event"
	uniswapV2SwapEventType  EventType = "uniswapV2SwapEvent"
	unknownEventType        EventType = "unknownEvent"

	erc20_721TransferEventSignature = "Transfer(address,address,uint256)"

	erc20TransferEventIndexedParameters  = 3 // signature, from, to
	erc721TransferEventIndexedParameters = 4 // signature, from, to, tokenId

	uniswapV2SwapEventSignature = "Swap(address,uint256,uint256,uint256,uint256,address)" // also used by SushiSwap
)

var (
	// MaxUint256 is the maximum value that can be represented by a uint256.
	MaxUint256 = new(big.Int).Sub(new(big.Int).Lsh(common.Big1, 256), common.Big1)
)

// Detect event type for a cetain item from the Events Log
func GetEventType(log *types.Log) EventType {
	erc20_721TransferEventSignatureHash := getEventSignatureHash(erc20_721TransferEventSignature)
	uniswapV2SwapEventSignatureHash := getEventSignatureHash(uniswapV2SwapEventSignature)

	if len(log.Topics) > 0 {
		switch log.Topics[0] {
		case erc20_721TransferEventSignatureHash:
			switch len(log.Topics) {
			case erc20TransferEventIndexedParameters:
				return erc20TransferEventType
			case erc721TransferEventIndexedParameters:
				return erc721TransferEventType
			}
		case uniswapV2SwapEventSignatureHash:
			return uniswapV2SwapEventType
		}
	}

	return unknownEventType
}

func EventTypeToSubtransactionType(eventType EventType) Type {
	switch eventType {
	case erc20TransferEventType:
		return erc20Transfer
	case erc721TransferEventType:
		return erc721Transfer
	case uniswapV2SwapEventType:
		return uniswapV2Swap
	}

	return unknownTransaction
}

func GetFirstEvent(logs []*types.Log) (EventType, *types.Log) {
	for _, log := range logs {
		eventType := GetEventType(log)
		if eventType != unknownEventType {
			return eventType, log
		}
	}

	return unknownEventType, nil
}

func IsTokenTransfer(logs []*types.Log) bool {
	eventType, _ := GetFirstEvent(logs)
	return eventType == erc20TransferEventType
}

func parseErc20TransferLog(ethlog *types.Log) (from, to common.Address, amount *big.Int) {
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

func parseErc721TransferLog(ethlog *types.Log) (from, to common.Address, tokenID *big.Int) {
	tokenID = new(big.Int)
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
	tokenID.SetBytes(ethlog.Topics[3][:])

	return
}

func parseUniswapV2Log(ethlog *types.Log) (pairAddress common.Address, from common.Address, to common.Address, amount0In *big.Int, amount1In *big.Int, amount0Out *big.Int, amount1Out *big.Int, err error) {
	amount0In = new(big.Int)
	amount1In = new(big.Int)
	amount0Out = new(big.Int)
	amount1Out = new(big.Int)

	if len(ethlog.Topics) < 3 {
		err = fmt.Errorf("not enough topics for uniswapV2 swap %s, %v", "topics", ethlog.Topics)
		return
	}

	pairAddress = ethlog.Address

	if len(ethlog.Topics[1]) != 32 {
		err = fmt.Errorf("second topic is not padded to 32 byte address %s, %v", "topic", ethlog.Topics[1])
		return
	}
	if len(ethlog.Topics[2]) != 32 {
		err = fmt.Errorf("third topic is not padded to 32 byte address %s, %v", "topic", ethlog.Topics[2])
		return
	}
	copy(from[:], ethlog.Topics[1][12:])
	copy(to[:], ethlog.Topics[2][12:])
	if len(ethlog.Data) != 32*4 {
		err = fmt.Errorf("data is not padded to 4 * 32 bytes big int %s, %v", "data", ethlog.Data)
		return
	}
	amount0In.SetBytes(ethlog.Data[0:32])
	amount1In.SetBytes(ethlog.Data[32:64])
	amount0Out.SetBytes(ethlog.Data[64:96])
	amount1Out.SetBytes(ethlog.Data[96:128])

	return
}
