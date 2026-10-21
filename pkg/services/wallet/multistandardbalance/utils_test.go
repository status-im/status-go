package multistandardbalance

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/assert"
)

func TestIsBigIntMapEqual(t *testing.T) {
	tests := []struct {
		name     string
		m1       map[string]*big.Int
		m2       map[string]*big.Int
		expected bool
	}{
		{
			name:     "empty maps",
			m1:       map[string]*big.Int{},
			m2:       map[string]*big.Int{},
			expected: true,
		},
		{
			name: "identical maps",
			m1: map[string]*big.Int{
				"key1": big.NewInt(100),
				"key2": big.NewInt(200),
			},
			m2: map[string]*big.Int{
				"key1": big.NewInt(100),
				"key2": big.NewInt(200),
			},
			expected: true,
		},
		{
			name: "different values",
			m1: map[string]*big.Int{
				"key1": big.NewInt(100),
				"key2": big.NewInt(200),
			},
			m2: map[string]*big.Int{
				"key1": big.NewInt(100),
				"key2": big.NewInt(300),
			},
			expected: false,
		},
		{
			name: "different keys",
			m1: map[string]*big.Int{
				"key1": big.NewInt(100),
				"key2": big.NewInt(200),
			},
			m2: map[string]*big.Int{
				"key1": big.NewInt(100),
				"key3": big.NewInt(200),
			},
			expected: false,
		},
		{
			name: "different lengths",
			m1: map[string]*big.Int{
				"key1": big.NewInt(100),
			},
			m2: map[string]*big.Int{
				"key1": big.NewInt(100),
				"key2": big.NewInt(200),
			},
			expected: false,
		},
		{
			name: "nil values",
			m1: map[string]*big.Int{
				"key1": nil,
				"key2": big.NewInt(200),
			},
			m2: map[string]*big.Int{
				"key1": nil,
				"key2": big.NewInt(200),
			},
			expected: true,
		},
		{
			name: "zero values",
			m1: map[string]*big.Int{
				"key1": big.NewInt(0),
				"key2": big.NewInt(200),
			},
			m2: map[string]*big.Int{
				"key1": big.NewInt(0),
				"key2": big.NewInt(200),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isBigIntMapEqual(tt.m1, tt.m2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

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

func TestIsBigIntMapEqualWithDifferentTypes(t *testing.T) {
	// Test with ContractAddress type
	contractMap1 := map[ContractAddress]*big.Int{
		common.HexToAddress("0x1111111111111111111111111111111111111111"): big.NewInt(100),
		common.HexToAddress("0x2222222222222222222222222222222222222222"): big.NewInt(200),
	}

	contractMap2 := map[ContractAddress]*big.Int{
		common.HexToAddress("0x1111111111111111111111111111111111111111"): big.NewInt(100),
		common.HexToAddress("0x2222222222222222222222222222222222222222"): big.NewInt(200),
	}

	assert.True(t, isBigIntMapEqual(contractMap1, contractMap2))

	// Test with HashableCollectibleID type
	collectibleMap1 := map[HashableCollectibleID]*big.Int{
		HashableCollectibleID{
			ContractAddress: common.HexToAddress("0x1111111111111111111111111111111111111111"),
			TokenID:         [32]byte{1},
		}: big.NewInt(10),
	}

	collectibleMap2 := map[HashableCollectibleID]*big.Int{
		HashableCollectibleID{
			ContractAddress: common.HexToAddress("0x1111111111111111111111111111111111111111"),
			TokenID:         [32]byte{1},
		}: big.NewInt(10),
	}

	assert.True(t, isBigIntMapEqual(collectibleMap1, collectibleMap2))
}

func answeredSet(tokens ...common.Address) map[common.Address]struct{} {
	set := make(map[common.Address]struct{}, len(tokens))
	for _, token := range tokens {
		set[token] = struct{}{}
	}
	return set
}

func TestMergeERC20Balances(t *testing.T) {
	a := common.HexToAddress("0x1")
	b := common.HexToAddress("0x2")
	c := common.HexToAddress("0x3")
	d := common.HexToAddress("0x4")

	t.Run("asked and missing is an answered zero", func(t *testing.T) {
		previous := ERC20Balances{Balances: map[common.Address]*big.Int{a: big.NewInt(10)}, Answered: answeredSet(a)}
		merged := mergeERC20Balances(previous, []common.Address{a, b, c}, map[common.Address]*big.Int{c: big.NewInt(5)}, nil)
		assert.Equal(t, map[common.Address]*big.Int{c: big.NewInt(5)}, merged.Balances)
		assert.Equal(t, answeredSet(a, b, c), merged.Answered)
	})

	t.Run("a failed call keeps the stored balance and is not answered", func(t *testing.T) {
		previous := ERC20Balances{Balances: map[common.Address]*big.Int{a: big.NewInt(10)}, Answered: answeredSet(a)}
		merged := mergeERC20Balances(previous, []common.Address{a, b}, nil, []common.Address{a, b})
		assert.Equal(t, map[common.Address]*big.Int{a: big.NewInt(10)}, merged.Balances)
		assert.Equal(t, answeredSet(a), merged.Answered)
	})

	t.Run("tokens not asked keep their stored state", func(t *testing.T) {
		previous := ERC20Balances{Balances: map[common.Address]*big.Int{d: big.NewInt(40)}, Answered: answeredSet(c, d)}
		merged := mergeERC20Balances(previous, []common.Address{a}, map[common.Address]*big.Int{a: big.NewInt(1)}, nil)
		assert.Equal(t, map[common.Address]*big.Int{a: big.NewInt(1), d: big.NewInt(40)}, merged.Balances)
		assert.Equal(t, answeredSet(a, c, d), merged.Answered)
	})

	t.Run("the answered set is reused while it does not grow", func(t *testing.T) {
		previous := ERC20Balances{Balances: map[common.Address]*big.Int{}, Answered: answeredSet(a, b)}
		merged := mergeERC20Balances(previous, []common.Address{a, b}, nil, nil)
		assert.Equal(t, reflect.ValueOf(previous.Answered).Pointer(), reflect.ValueOf(merged.Answered).Pointer())
		assert.True(t, isERC20BalancesEqual(previous, merged))

		grown := mergeERC20Balances(merged, []common.Address{a, b, c}, nil, nil)
		assert.Equal(t, answeredSet(a, b), previous.Answered, "previous set untouched")
		assert.False(t, isERC20BalancesEqual(merged, grown))
	})
}
