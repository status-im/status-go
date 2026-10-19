package multistandardbalance

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/assert"
)

func TestDeleteAccountsNotInList(t *testing.T) {
	tests := []struct {
		name           string
		initialMap     map[BalancesKey]string
		accountsToKeep []common.Address
		expectedKeys   []BalancesKey
	}{
		{
			name: "keep all accounts",
			initialMap: map[BalancesKey]string{
				{Account: common.HexToAddress("0x1111111111111111111111111111111111111111"), ChainID: 1}: "value1",
				{Account: common.HexToAddress("0x2222222222222222222222222222222222222222"), ChainID: 1}: "value2",
			},
			accountsToKeep: []common.Address{
				common.HexToAddress("0x1111111111111111111111111111111111111111"),
				common.HexToAddress("0x2222222222222222222222222222222222222222"),
			},
			expectedKeys: []BalancesKey{
				{Account: common.HexToAddress("0x1111111111111111111111111111111111111111"), ChainID: 1},
				{Account: common.HexToAddress("0x2222222222222222222222222222222222222222"), ChainID: 1},
			},
		},
		{
			name: "remove some accounts",
			initialMap: map[BalancesKey]string{
				{Account: common.HexToAddress("0x1111111111111111111111111111111111111111"), ChainID: 1}: "value1",
				{Account: common.HexToAddress("0x2222222222222222222222222222222222222222"), ChainID: 1}: "value2",
				{Account: common.HexToAddress("0x3333333333333333333333333333333333333333"), ChainID: 1}: "value3",
			},
			accountsToKeep: []common.Address{
				common.HexToAddress("0x1111111111111111111111111111111111111111"),
				common.HexToAddress("0x2222222222222222222222222222222222222222"),
			},
			expectedKeys: []BalancesKey{
				{Account: common.HexToAddress("0x1111111111111111111111111111111111111111"), ChainID: 1},
				{Account: common.HexToAddress("0x2222222222222222222222222222222222222222"), ChainID: 1},
			},
		},
		{
			name: "remove all accounts",
			initialMap: map[BalancesKey]string{
				{Account: common.HexToAddress("0x1111111111111111111111111111111111111111"), ChainID: 1}: "value1",
				{Account: common.HexToAddress("0x2222222222222222222222222222222222222222"), ChainID: 1}: "value2",
			},
			accountsToKeep: []common.Address{},
			expectedKeys:   []BalancesKey{},
		},
		{
			name:       "empty map",
			initialMap: map[BalancesKey]string{},
			accountsToKeep: []common.Address{
				common.HexToAddress("0x1111111111111111111111111111111111111111"),
			},
			expectedKeys: []BalancesKey{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := make(map[BalancesKey]string)
			for k, v := range tt.initialMap {
				m[k] = v
			}

			deleteAccountsNotInList(m, tt.accountsToKeep)

			assert.Equal(t, len(tt.expectedKeys), len(m))
			for _, expectedKey := range tt.expectedKeys {
				_, exists := m[expectedKey]
				assert.True(t, exists, "Expected key %v to exist", expectedKey)
			}
		})
	}
}

func TestDeleteChainsNotInList(t *testing.T) {
	tests := []struct {
		name         string
		initialMap   map[BalancesKey]string
		chainsToKeep []uint64
		expectedKeys []BalancesKey
	}{
		{
			name: "keep all chains",
			initialMap: map[BalancesKey]string{
				{Account: common.HexToAddress("0x1111111111111111111111111111111111111111"), ChainID: 1}: "value1",
				{Account: common.HexToAddress("0x1111111111111111111111111111111111111111"), ChainID: 2}: "value2",
			},
			chainsToKeep: []uint64{1, 2},
			expectedKeys: []BalancesKey{
				{Account: common.HexToAddress("0x1111111111111111111111111111111111111111"), ChainID: 1},
				{Account: common.HexToAddress("0x1111111111111111111111111111111111111111"), ChainID: 2},
			},
		},
		{
			name: "remove some chains",
			initialMap: map[BalancesKey]string{
				{Account: common.HexToAddress("0x1111111111111111111111111111111111111111"), ChainID: 1}: "value1",
				{Account: common.HexToAddress("0x1111111111111111111111111111111111111111"), ChainID: 2}: "value2",
				{Account: common.HexToAddress("0x1111111111111111111111111111111111111111"), ChainID: 3}: "value3",
			},
			chainsToKeep: []uint64{1, 2},
			expectedKeys: []BalancesKey{
				{Account: common.HexToAddress("0x1111111111111111111111111111111111111111"), ChainID: 1},
				{Account: common.HexToAddress("0x1111111111111111111111111111111111111111"), ChainID: 2},
			},
		},
		{
			name: "remove all chains",
			initialMap: map[BalancesKey]string{
				{Account: common.HexToAddress("0x1111111111111111111111111111111111111111"), ChainID: 1}: "value1",
				{Account: common.HexToAddress("0x1111111111111111111111111111111111111111"), ChainID: 2}: "value2",
			},
			chainsToKeep: []uint64{},
			expectedKeys: []BalancesKey{},
		},
		{
			name:         "empty map",
			initialMap:   map[BalancesKey]string{},
			chainsToKeep: []uint64{1, 2},
			expectedKeys: []BalancesKey{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := make(map[BalancesKey]string)
			for k, v := range tt.initialMap {
				m[k] = v
			}

			deleteChainsNotInList(m, tt.chainsToKeep)

			assert.Equal(t, len(tt.expectedKeys), len(m))
			for _, expectedKey := range tt.expectedKeys {
				_, exists := m[expectedKey]
				assert.True(t, exists, "Expected key %v to exist", expectedKey)
			}
		})
	}
}

func TestCountMissing(t *testing.T) {
	a := common.HexToAddress("0x1")
	b := common.HexToAddress("0x2")

	assert.Equal(t, 1, countMissing(map[common.Address]*big.Int{a: big.NewInt(10), b: big.NewInt(20)}, map[common.Address]*big.Int{a: big.NewInt(0)}))
	assert.Equal(t, 0, countMissing(nil, map[common.Address]*big.Int{a: big.NewInt(5)}))
	assert.Equal(t, 1, countMissing(map[common.Address]*big.Int{a: big.NewInt(10)}, nil))
}
