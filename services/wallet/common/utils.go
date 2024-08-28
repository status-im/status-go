package common

import (
	"context"
	"math/big"
	"reflect"

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
