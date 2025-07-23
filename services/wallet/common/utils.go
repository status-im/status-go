package common

import (
	"context"
	"fmt"
	"math/big"
	"reflect"
	"slices"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethParams "github.com/ethereum/go-ethereum/params"
	"github.com/status-im/status-go/params"
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

func NetworksToChainIDs(networks []*params.Network) []uint64 {
	chainIDs := make([]uint64, 0)
	for _, network := range networks {
		chainIDs = append(chainIDs, network.ChainID)
	}

	return chainIDs
}

func ArrayContainsElement[T comparable](el T, arr []T) bool {
	return slices.Contains(arr, el)
}

func IsSingleChainOperation(fromChain *params.Network, toChain *params.Network) bool {
	return fromChain.ChainID == toChain.ChainID
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
	return new(big.Float).Quo(val, big.NewFloat(1000000000))
}

func WeiToGwei(val *big.Int) *big.Float {
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

// Special functions to hardcode the nature of some special chains (eg. Status Network), where we cannot deduce EIP-1559 compatibility in a generic way

// IsPartiallyOrFullyGaslessChain returns true if the chain is fully or partially (no base or no priority fee) gasless
func IsPartiallyOrFullyGaslessChain(chainID uint64) bool {
	return chainID == StatusNetworkSepolia
}

// IsPartiallyOrFullyGaslessChainEIP1559Compatible throws an error if the chain is not partially or fully gasless, if it is, returns true if the chain is EIP-1559 compatible
func IsPartiallyOrFullyGaslessChainEIP1559Compatible(chainID uint64) (bool, error) {
	if !IsPartiallyOrFullyGaslessChain(chainID) {
		return false, fmt.Errorf("chain %d is not supposed to be gasless", chainID) // for non-gasless chains, we should not use this function
	}
	return chainID == StatusNetworkSepolia, nil
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
