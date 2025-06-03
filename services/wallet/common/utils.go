package common

import (
	"context"
	"math/big"
	"reflect"
	"time"

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
	for _, e := range arr {
		if e == el {
			return true
		}
	}
	return false
}

func IsSingleChainOperation(fromChains []*params.Network, toChains []*params.Network) bool {
	return len(fromChains) == 1 &&
		len(toChains) == 1 &&
		fromChains[0].ChainID == toChains[0].ChainID
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

// IsGaslessChainAndEIP1559Compatible returns true if the chain is gasless and EIP-1559 compatible
func IsGaslessChainAndEIP1559Compatible(chainID uint64) bool {
	return chainID == StatusNetworkSepolia
}

// HasNoBaseFee returns true if the chain has no base fee (eg. Status Network is (will be fully) gasless chain, but EIP-1559 compatible, but at the moment its base fee is 0)
func HasNoBaseFee(chainID uint64) bool {
	return chainID == StatusNetworkSepolia
}

// HasNoPriorityFee returns true if the chain has no priority fee (eg. Status Network is (will be fully) gasless chain, but EIP-1559 compatible, but at the moment it has priority fee greater than 0)
func HasNoPriorityFee(chainID uint64) bool {
	return false // At this moment Status Network has priority fee, but it will be 0 after the upgrade
}
