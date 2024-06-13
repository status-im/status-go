package testutils

import (
	"reflect"
	"sort"

	"github.com/golang/mock/gomock"

	"github.com/ethereum/go-ethereum/common"
)

const EthSymbol = "ETH"
const SntSymbol = "SNT"
const DaiSymbol = "DAI"

func SliceContains[T comparable](slice []T, value T) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}
func StructExistsInSlice[T any](target T, slice []T) bool {
	for _, item := range slice {
		if reflect.DeepEqual(target, item) {
			return true
		}
	}
	return false
}

func Filter[T any](ss []T, test func(T) bool) (ret []T) {
	for _, s := range ss {
		if test(s) {
			ret = append(ret, s)
		}
	}
	return
}

// AddressSliceMatcher is a custom matcher for comparing common.Address slices regardless of order.
type AddressSliceMatcher struct {
	expected []common.Address
}

func NewAddressSliceMatcher(expected []common.Address) gomock.Matcher {
	return &AddressSliceMatcher{expected: expected}
}

func (m *AddressSliceMatcher) Matches(x interface{}) bool {
	actual, ok := x.([]common.Address)
	if !ok {
		return false
	}

	if len(m.expected) != len(actual) {
		return false
	}

	// Create copies of the slices to sort them
	expectedCopy := make([]common.Address, len(m.expected))
	actualCopy := make([]common.Address, len(actual))
	copy(expectedCopy, m.expected)
	copy(actualCopy, actual)

	sort.Slice(expectedCopy, func(i, j int) bool { return expectedCopy[i].Hex() < expectedCopy[j].Hex() })
	sort.Slice(actualCopy, func(i, j int) bool { return actualCopy[i].Hex() < actualCopy[j].Hex() })

	for i := range expectedCopy {
		if expectedCopy[i] != actualCopy[i] {
			return false
		}
	}

	return true
}

func (m *AddressSliceMatcher) String() string {
	return "matches Address slice regardless of order"
}

// Uint64SliceMatcher is a custom matcher for comparing uint64 slices regardless of order.
type Uint64SliceMatcher struct {
	expected []uint64
}

func NewUint64SliceMatcher(expected []uint64) gomock.Matcher {
	return &Uint64SliceMatcher{expected: expected}
}

func (m *Uint64SliceMatcher) Matches(x interface{}) bool {
	actual, ok := x.([]uint64)
	if !ok {
		return false
	}

	if len(m.expected) != len(actual) {
		return false
	}

	// Create copies of the slices to sort them
	expectedCopy := make([]uint64, len(m.expected))
	actualCopy := make([]uint64, len(actual))
	copy(expectedCopy, m.expected)
	copy(actualCopy, actual)

	sort.Slice(expectedCopy, func(i, j int) bool { return expectedCopy[i] < expectedCopy[j] })
	sort.Slice(actualCopy, func(i, j int) bool { return actualCopy[i] < actualCopy[j] })

	for i := range expectedCopy {
		if expectedCopy[i] != actualCopy[i] {
			return false
		}
	}

	return true
}

func (m *Uint64SliceMatcher) String() string {
	return "matches uint64 slice regardless of order"
}
