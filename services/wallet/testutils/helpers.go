package testutils

import (
	"fmt"

	"go.uber.org/mock/gomock"

	"github.com/ethereum/go-ethereum/common"
)

const EthSymbol = "ETH"
const SntSymbol = "SNT"
const DaiSymbol = "DAI"

// AddressSliceMatcher is a custom matcher for comparing common.Address slices regardless of order.
type AddressSliceMatcher struct {
	expected []common.Address
}

// Uint64SliceMatcher is a custom matcher for comparing uint64 slices regardless of order.
type Uint64SliceMatcher struct {
	expected []uint64
}

// StringSliceElementsMatcher is a custom matcher for comparing string slices regardless of order.
type StringSliceElementsMatcher struct {
	expected []string
}

func NewStringSliceElementsMatcher(expected []string) gomock.Matcher {
	return &StringSliceElementsMatcher{expected: expected}
}

func (m *StringSliceElementsMatcher) Matches(x interface{}) bool {
	actual, ok := x.([]string)
	if !ok {
		return false
	}
	if len(actual) != len(m.expected) {
		return false
	}
	expectedSet := make(map[string]struct{}, len(m.expected))
	for _, k := range m.expected {
		expectedSet[k] = struct{}{}
	}
	for _, k := range actual {
		if _, ok := expectedSet[k]; !ok {
			return false
		}
	}
	return true
}

func (m *StringSliceElementsMatcher) String() string {
	return fmt.Sprintf("contains elements %v", m.expected)
}
