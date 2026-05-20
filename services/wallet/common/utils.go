package common

import (
	"context"
	"math/big"
	"reflect"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethParams "github.com/ethereum/go-ethereum/params"
)

// ShouldCancel returns true if the context has been cancelled and task should be aborted
func ShouldCancel(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
	}
	return false
}

func ArrayContainsElement[T comparable](el T, arr []T) bool {
	for _, e := range arr {
		if e == el {
			return true
		}
	}
	return false
}

func IsSingleChainOperation(fromChainID, toChainID uint64) bool {
	return fromChainID == toChainID
}

// CopyMapGeneric creates a copy of any map, if the deepCopyValue function is provided, it will be used to copy values.
func CopyMapGeneric(original interface{}, deepCopyValueFn func(interface{}) interface{}) interface{} {
	originalVal := reflect.ValueOf(original)
	if originalVal.Kind() != reflect.Map {
		return nil
	}

	newMap := reflect.MakeMap(originalVal.Type())
	for iter := originalVal.MapRange(); iter.Next(); {
		if deepCopyValueFn != nil {
			newMap.SetMapIndex(iter.Key(), reflect.ValueOf(deepCopyValueFn(iter.Value().Interface())))
		} else {
			newMap.SetMapIndex(iter.Key(), iter.Value())
		}
	}

	return newMap.Interface()
}

func GweiToEth(val *big.Float) *big.Float {
	if val == nil {
		return nil
	}
	return new(big.Float).Quo(val, big.NewFloat(1000000000))
}

func WeiToGwei(val *big.Int) *big.Float {
	if val == nil {
		return nil
	}
	result := new(big.Float)
	result.SetInt(val)

	unit := new(big.Int)
	unit.SetInt64(gethParams.GWei)

	return result.Quo(result, new(big.Float).SetInt(unit))
}

func GetBlockCreationTimeForChain(chainID uint64) time.Duration {
	blockDuration, found := AverageBlockDurationForChain[ChainID(chainID)]
	if !found {
		blockDuration = AverageBlockDurationForChain[ChainID(UnknownChainID)]
	}
	return blockDuration
}

func ToCallArg(msg ethereum.CallMsg) interface{} {
	arg := map[string]interface{}{
		"from": msg.From,
		"to":   msg.To,
	}
	if len(msg.Data) > 0 {
		arg["data"] = hexutil.Bytes(msg.Data)
	}
	if msg.Value != nil {
		arg["value"] = (*hexutil.Big)(msg.Value)
	}
	if msg.Gas != 0 {
		arg["gas"] = hexutil.Uint64(msg.Gas)
	}
	if msg.GasPrice != nil {
		arg["gasPrice"] = (*hexutil.Big)(msg.GasPrice)
	}
	return arg
}
