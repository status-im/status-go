package router

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/stretchr/testify/assert"
)

func TestSetupRouteValidationMapsV2(t *testing.T) {
	tests := []struct {
		name                 string
		fromLockedAmount     map[uint64]*hexutil.Big
		expectedFromIncluded map[uint64]bool
		expectedFromExcluded map[uint64]bool
	}{
		{
			name: "Mixed locked amounts",
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(100)),
				2: (*hexutil.Big)(big.NewInt(0)),
				3: (*hexutil.Big)(big.NewInt(50)),
			},
			expectedFromIncluded: map[uint64]bool{1: false, 3: false},
			expectedFromExcluded: map[uint64]bool{2: true},
		},
		{
			name: "All amounts locked",
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(100)),
				2: (*hexutil.Big)(big.NewInt(50)),
			},
			expectedFromIncluded: map[uint64]bool{1: false, 2: false},
			expectedFromExcluded: map[uint64]bool{},
		},
		{
			name: "No amounts locked",
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(0)),
				2: (*hexutil.Big)(big.NewInt(0)),
			},
			expectedFromIncluded: map[uint64]bool{},
			expectedFromExcluded: map[uint64]bool{1: true, 2: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fromIncluded, fromExcluded := setupRouteValidationMapsV2(tt.fromLockedAmount)
			assert.Equal(t, tt.expectedFromIncluded, fromIncluded)
			assert.Equal(t, tt.expectedFromExcluded, fromExcluded)
		})
	}
}

func TestCalculateTotalRestAmountV2(t *testing.T) {
	tests := []struct {
		name          string
		route         []*PathV2
		expectedTotal *big.Int
	}{
		{
			name: "Multiple paths with varying amounts",
			route: []*PathV2{
				{AmountIn: (*hexutil.Big)(big.NewInt(100))},
				{AmountIn: (*hexutil.Big)(big.NewInt(200))},
				{AmountIn: (*hexutil.Big)(big.NewInt(300))},
			},
			expectedTotal: big.NewInt(600),
		},
		{
			name: "Single path",
			route: []*PathV2{
				{AmountIn: (*hexutil.Big)(big.NewInt(500))},
			},
			expectedTotal: big.NewInt(500),
		},
		{
			name:          "No paths",
			route:         []*PathV2{},
			expectedTotal: big.NewInt(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			total := calculateTotalRestAmountV2(tt.route)
			assert.Equal(t, tt.expectedTotal, total)
		})
	}
}
