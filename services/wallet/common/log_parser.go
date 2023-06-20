// Moved here because transactions package depends on accounts package which
// depends on appdatabase where this functionality is needed
package common

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
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
	UniswapV2Swap      Type = "uniswapV2Swap"
	UniswapV3Swap      Type = "uniswapV3Swap"
	unknownTransaction Type = "unknown"

	// Event types
	Erc20TransferEventType  EventType = "erc20Event"
	Erc721TransferEventType EventType = "erc721Event"
	UniswapV2SwapEventType  EventType = "uniswapV2SwapEvent"
	UniswapV3SwapEventType  EventType = "uniswapV3SwapEvent"
	UnknownEventType        EventType = "unknownEvent"

	Erc20_721TransferEventSignature = "Transfer(address,address,uint256)"

	erc20TransferEventIndexedParameters  = 3 // signature, from, to
	erc721TransferEventIndexedParameters = 4 // signature, from, to, tokenId

	uniswapV2SwapEventSignature = "Swap(address,uint256,uint256,uint256,uint256,address)" // also used by SushiSwap
	uniswapV3SwapEventSignature = "Swap(address,address,int256,int256,uint160,uint128,int24)"
)

var (
	// MaxUint256 is the maximum value that can be represented by a uint256.
	MaxUint256 = new(big.Int).Sub(new(big.Int).Lsh(common.Big1, 256), common.Big1)
)

// Detect event type for a cetain item from the Events Log
func GetEventType(log *types.Log) EventType {
	erc20_721TransferEventSignatureHash := GetEventSignatureHash(Erc20_721TransferEventSignature)
	uniswapV2SwapEventSignatureHash := GetEventSignatureHash(uniswapV2SwapEventSignature)
	uniswapV3SwapEventSignatureHash := GetEventSignatureHash(uniswapV3SwapEventSignature)

	if len(log.Topics) > 0 {
		switch log.Topics[0] {
		case erc20_721TransferEventSignatureHash:
			switch len(log.Topics) {
			case erc20TransferEventIndexedParameters:
				return Erc20TransferEventType
			case erc721TransferEventIndexedParameters:
				return Erc721TransferEventType
			}
		case uniswapV2SwapEventSignatureHash:
			return UniswapV2SwapEventType
		case uniswapV3SwapEventSignatureHash:
			return UniswapV3SwapEventType
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
	case UniswapV2SwapEventType:
		return UniswapV2Swap
	case UniswapV3SwapEventType:
		return UniswapV3Swap
	}

	return unknownTransaction
}

func GetFirstEvent(logs []*types.Log) (EventType, *types.Log) {
	for _, log := range logs {
		eventType := GetEventType(log)
		if eventType != UnknownEventType {
			return eventType, log
		}
	}

	return UnknownEventType, nil
}

func IsTokenTransfer(logs []*types.Log) bool {
	eventType, _ := GetFirstEvent(logs)
	return eventType == Erc20TransferEventType
}

func ParseErc20TransferLog(ethlog *types.Log) (from, to common.Address, amount *big.Int) {
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

func ParseErc721TransferLog(ethlog *types.Log) (from, to common.Address, tokenID *big.Int) {
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

func ParseUniswapV2Log(ethlog *types.Log) (pairAddress common.Address, from common.Address, to common.Address, amount0In *big.Int, amount1In *big.Int, amount0Out *big.Int, amount1Out *big.Int, err error) {
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

func readInt256(b []byte) *big.Int {
	// big.SetBytes can't tell if a number is negative or positive in itself.
	// On EVM, if the returned number > max int256, it is negative.
	// A number is > max int256 if the bit at position 255 is set.
	ret := new(big.Int).SetBytes(b)
	if ret.Bit(255) == 1 {
		ret.Add(MaxUint256, new(big.Int).Neg(ret))
		ret.Add(ret, common.Big1)
		ret.Neg(ret)
	}
	return ret
}

func ParseUniswapV3Log(ethlog *types.Log) (poolAddress common.Address, sender common.Address, recipient common.Address, amount0 *big.Int, amount1 *big.Int, err error) {
	amount0 = new(big.Int)
	amount1 = new(big.Int)

	if len(ethlog.Topics) < 3 {
		err = fmt.Errorf("not enough topics for uniswapV3 swap %s, %v", "topics", ethlog.Topics)
		return
	}

	poolAddress = ethlog.Address

	if len(ethlog.Topics[1]) != 32 {
		err = fmt.Errorf("second topic is not padded to 32 byte address %s, %v", "topic", ethlog.Topics[1])
		return
	}
	if len(ethlog.Topics[2]) != 32 {
		err = fmt.Errorf("third topic is not padded to 32 byte address %s, %v", "topic", ethlog.Topics[2])
		return
	}
	copy(sender[:], ethlog.Topics[1][12:])
	copy(recipient[:], ethlog.Topics[2][12:])
	if len(ethlog.Data) != 32*5 {
		err = fmt.Errorf("data is not padded to 5 * 32 bytes big int %s, %v", "data", ethlog.Data)
		return
	}
	amount0 = readInt256(ethlog.Data[0:32])
	amount1 = readInt256(ethlog.Data[32:64])

	return
}

func GetEventSignatureHash(signature string) common.Hash {
	return crypto.Keccak256Hash([]byte(signature))
}

func ExtractTokenIdentity(dbEntryType Type, log *types.Log, tx *types.Transaction) (correctType Type, tokenAddress *common.Address, txTokenID *big.Int, txValue *big.Int, txFrom *common.Address, txTo *common.Address) {
	// erc721 transfers share signature with erc20 ones, so they both used to be categorized as erc20
	// by the Downloader. We fix this here since they might be mis-categorized in the db.
	if dbEntryType == Erc20Transfer {
		eventType := GetEventType(log)
		correctType = EventTypeToSubtransactionType(eventType)
	} else {
		correctType = dbEntryType
	}

	switch correctType {
	case EthTransfer:
		if tx != nil {
			txValue = new(big.Int).Set(tx.Value())
		}
	case Erc20Transfer:
		tokenAddress = new(common.Address)
		*tokenAddress = log.Address
		from, to, value := ParseErc20TransferLog(log)
		txValue = value
		txFrom = &from
		txTo = &to
	case Erc721Transfer:
		tokenAddress = new(common.Address)
		*tokenAddress = log.Address
		from, to, tokenID := ParseErc721TransferLog(log)
		txTokenID = tokenID
		txFrom = &from
		txTo = &to
	}

	return
}
