package router

import (
	"github.com/status-im/status-go/params"
)

func arrayContainsElement[T comparable](el T, arr []T) bool {
	for _, e := range arr {
		if e == el {
			return true
		}
	}
	return false
}

func arraysWithSameElements[T comparable](ar1 []T, ar2 []T, isEqual func(T, T) bool) bool {
	if len(ar1) != len(ar2) {
		return false
	}
	for _, el := range ar1 {
		if !arrayContainsElement(el, ar2) {
			return false
		}
	}
	return true
}

func isSingleChainOperation(fromChains []*params.Network, toChains []*params.Network) bool {
	return len(fromChains) == 1 &&
		len(toChains) == 1 &&
		fromChains[0].ChainID == toChains[0].ChainID
}
