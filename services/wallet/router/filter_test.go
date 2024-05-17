package router

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/status-im/status-go/params"

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

func TestIsValidForNetworkComplianceV2(t *testing.T) {
	tests := []struct {
		name             string
		route            []*PathV2
		fromLockedAmount map[uint64]*hexutil.Big
		expectedResult   bool
	}{
		{
			name: "Valid route with required chain IDs included",
			route: []*PathV2{
				{From: &params.Network{ChainID: 1}},
				{From: &params.Network{ChainID: 3}},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(100)),
				2: (*hexutil.Big)(big.NewInt(0)),
			},
			expectedResult: true,
		},
		{
			name: "Invalid route with excluded chain ID",
			route: []*PathV2{
				{From: &params.Network{ChainID: 2}},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(100)),
				2: (*hexutil.Big)(big.NewInt(0)),
			},
			expectedResult: false,
		},
		{
			name: "Route missing required chain ID",
			route: []*PathV2{
				{From: &params.Network{ChainID: 3}},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(100)),
				2: (*hexutil.Big)(big.NewInt(50)),
			},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidForNetworkComplianceV2(tt.route, tt.fromLockedAmount)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestHasSufficientCapacityV2(t *testing.T) {
	tests := []struct {
		name             string
		route            []*PathV2
		amountIn         *big.Int
		fromLockedAmount map[uint64]*hexutil.Big
		expectedResult   bool
	}{
		{
			name: "Sufficient capacity with multiple paths",
			route: []*PathV2{
				{From: &params.Network{ChainID: 1}, AmountIn: (*hexutil.Big)(big.NewInt(100))},
				{From: &params.Network{ChainID: 2}, AmountIn: (*hexutil.Big)(big.NewInt(200))},
			},
			amountIn: big.NewInt(150),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(50)),
				2: (*hexutil.Big)(big.NewInt(100)),
			},
			expectedResult: true,
		},
		{
			name: "Insufficient capacity",
			route: []*PathV2{
				{From: &params.Network{ChainID: 1}, AmountIn: (*hexutil.Big)(big.NewInt(100))},
				{From: &params.Network{ChainID: 2}, AmountIn: (*hexutil.Big)(big.NewInt(50))},
			},
			amountIn: big.NewInt(200),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(50)),
				2: (*hexutil.Big)(big.NewInt(50)),
			},
			expectedResult: false,
		},
		{
			name: "Exact capacity match",
			route: []*PathV2{
				{From: &params.Network{ChainID: 1}, AmountIn: (*hexutil.Big)(big.NewInt(100))},
				{From: &params.Network{ChainID: 2}, AmountIn: (*hexutil.Big)(big.NewInt(50))},
			},
			amountIn: big.NewInt(150),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(100)),
				2: (*hexutil.Big)(big.NewInt(50)),
			},
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasSufficientCapacityV2(tt.route, tt.amountIn, tt.fromLockedAmount)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestFilterNetworkComplianceV2(t *testing.T) {
	tests := []struct {
		name             string
		routes           [][]*PathV2
		fromLockedAmount map[uint64]*hexutil.Big
		expectedRoutes   [][]*PathV2
	}{
		{
			name: "Mixed routes with valid and invalid paths",
			routes: [][]*PathV2{
				{
					{From: &params.Network{ChainID: 1}},
					{From: &params.Network{ChainID: 3}},
				},
				{
					{From: &params.Network{ChainID: 2}},
					{From: &params.Network{ChainID: 3}},
				},
				{
					{From: &params.Network{ChainID: 1}},
					{From: &params.Network{ChainID: 2}},
					{From: &params.Network{ChainID: 3}},
				},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(100)),
				2: (*hexutil.Big)(big.NewInt(0)),
			},
			expectedRoutes: [][]*PathV2{
				{
					{From: &params.Network{ChainID: 1}},
					{From: &params.Network{ChainID: 3}},
				},
				{
					{From: &params.Network{ChainID: 1}},
					{From: &params.Network{ChainID: 2}},
					{From: &params.Network{ChainID: 3}},
				},
			},
		},
		{
			name: "All valid routes",
			routes: [][]*PathV2{
				{
					{From: &params.Network{ChainID: 1}},
					{From: &params.Network{ChainID: 3}},
				},
				{
					{From: &params.Network{ChainID: 1}},
					{From: &params.Network{ChainID: 4}},
				},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(100)),
			},
			expectedRoutes: [][]*PathV2{
				{
					{From: &params.Network{ChainID: 1}},
					{From: &params.Network{ChainID: 3}},
				},
				{
					{From: &params.Network{ChainID: 1}},
					{From: &params.Network{ChainID: 4}},
				},
			},
		},
		{
			name: "All invalid routes",
			routes: [][]*PathV2{
				{
					{From: &params.Network{ChainID: 2}},
					{From: &params.Network{ChainID: 3}},
				},
				{
					{From: &params.Network{ChainID: 4}},
					{From: &params.Network{ChainID: 5}},
				},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(100)),
				2: (*hexutil.Big)(big.NewInt(0)),
			},
			expectedRoutes: [][]*PathV2{},
		},
		{
			name:   "Empty routes",
			routes: [][]*PathV2{},
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(100)),
			},
			expectedRoutes: [][]*PathV2{},
		},
		{
			name: "No locked amounts",
			routes: [][]*PathV2{
				{
					{From: &params.Network{ChainID: 1}},
					{From: &params.Network{ChainID: 2}},
				},
				{
					{From: &params.Network{ChainID: 3}},
					{From: &params.Network{ChainID: 4}},
				},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{},
			expectedRoutes: [][]*PathV2{
				{
					{From: &params.Network{ChainID: 1}},
					{From: &params.Network{ChainID: 2}},
				},
				{
					{From: &params.Network{ChainID: 3}},
					{From: &params.Network{ChainID: 4}},
				},
			},
		},
		{
			name: "Single route with mixed valid and invalid paths",
			routes: [][]*PathV2{
				{
					{From: &params.Network{ChainID: 1}},
					{From: &params.Network{ChainID: 2}},
					{From: &params.Network{ChainID: 3}},
				},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(100)),
				2: (*hexutil.Big)(big.NewInt(0)),
			},
			expectedRoutes: [][]*PathV2{
				{
					{From: &params.Network{ChainID: 1}},
					{From: &params.Network{ChainID: 3}},
				},
			},
		},
		{
			name: "Routes with duplicate chain IDs",
			routes: [][]*PathV2{
				{
					{From: &params.Network{ChainID: 1}},
					{From: &params.Network{ChainID: 1}},
					{From: &params.Network{ChainID: 2}},
				},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(100)),
			},
			expectedRoutes: [][]*PathV2{
				{
					{From: &params.Network{ChainID: 1}},
					{From: &params.Network{ChainID: 1}},
					{From: &params.Network{ChainID: 2}},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filteredRoutes := filterNetworkComplianceV2(tt.routes, tt.fromLockedAmount)
			assert.Equal(t, tt.expectedRoutes, filteredRoutes)
		})
	}
}

func TestFilterCapacityValidationV2(t *testing.T) {
	network1 := &params.Network{ChainID: 1}
	network2 := &params.Network{ChainID: 2}
	network3 := &params.Network{ChainID: 3}

	tests := []struct {
		name             string
		routes           [][]*PathV2
		amountIn         *big.Int
		fromLockedAmount map[uint64]*hexutil.Big
		expectedRoutes   [][]*PathV2
	}{
		{
			name: "Sufficient capacity with multiple paths",
			routes: [][]*PathV2{
				{
					{From: network1, AmountIn: (*hexutil.Big)(big.NewInt(100))},
					{From: network2, AmountIn: (*hexutil.Big)(big.NewInt(200))},
				},
				{
					{From: network1, AmountIn: (*hexutil.Big)(big.NewInt(50))},
					{From: network2, AmountIn: (*hexutil.Big)(big.NewInt(100))},
				},
			},
			amountIn: big.NewInt(150),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(50)),
				2: (*hexutil.Big)(big.NewInt(100)),
			},
			expectedRoutes: [][]*PathV2{
				{
					{From: network1, AmountIn: (*hexutil.Big)(big.NewInt(100)), AmountInLocked: false},
					{From: network2, AmountIn: (*hexutil.Big)(big.NewInt(200)), AmountInLocked: false},
				},
			},
		},
		{
			name: "Insufficient capacity",
			routes: [][]*PathV2{
				{
					{From: network1, AmountIn: (*hexutil.Big)(big.NewInt(100))},
					{From: network2, AmountIn: (*hexutil.Big)(big.NewInt(50))},
				},
			},
			amountIn: big.NewInt(200),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(50)),
				2: (*hexutil.Big)(big.NewInt(50)),
			},
			expectedRoutes: [][]*PathV2{},
		},
		{
			name: "Exact capacity match",
			routes: [][]*PathV2{
				{
					{From: network1, AmountIn: (*hexutil.Big)(big.NewInt(100))},
					{From: network2, AmountIn: (*hexutil.Big)(big.NewInt(50))},
				},
			},
			amountIn: big.NewInt(150),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(100)),
				2: (*hexutil.Big)(big.NewInt(50)),
			},
			expectedRoutes: [][]*PathV2{
				{
					{From: network1, AmountIn: (*hexutil.Big)(big.NewInt(100)), AmountInLocked: true},
					{From: network2, AmountIn: (*hexutil.Big)(big.NewInt(50)), AmountInLocked: true},
				},
			},
		},
		{
			name: "No locked amounts",
			routes: [][]*PathV2{
				{
					{From: network1, AmountIn: (*hexutil.Big)(big.NewInt(100))},
					{From: network2, AmountIn: (*hexutil.Big)(big.NewInt(50))},
				},
			},
			amountIn:         big.NewInt(150),
			fromLockedAmount: map[uint64]*hexutil.Big{},
			expectedRoutes: [][]*PathV2{
				{
					{From: network1, AmountIn: (*hexutil.Big)(big.NewInt(100)), AmountInLocked: false},
					{From: network2, AmountIn: (*hexutil.Big)(big.NewInt(50)), AmountInLocked: false},
				},
			},
		},
		{
			name: "Single route with sufficient capacity",
			routes: [][]*PathV2{
				{
					{From: network1, AmountIn: (*hexutil.Big)(big.NewInt(200))},
				},
			},
			amountIn: big.NewInt(150),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(50)),
			},
			expectedRoutes: [][]*PathV2{
				{
					{From: network1, AmountIn: (*hexutil.Big)(big.NewInt(200)), AmountInLocked: false},
				},
			},
		},
		{
			name: "Single route with insufficient capacity",
			routes: [][]*PathV2{
				{
					{From: network1, AmountIn: (*hexutil.Big)(big.NewInt(100))},
				},
			},
			amountIn: big.NewInt(150),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(50)),
			},
			expectedRoutes: [][]*PathV2{},
		},
		{
			name:     "Empty routes",
			routes:   [][]*PathV2{},
			amountIn: big.NewInt(150),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(50)),
			},
			expectedRoutes: [][]*PathV2{},
		},
		{
			name: "Routes with duplicate chain IDs",
			routes: [][]*PathV2{
				{
					{From: network1, AmountIn: (*hexutil.Big)(big.NewInt(100))},
					{From: network1, AmountIn: (*hexutil.Big)(big.NewInt(100))},
				},
			},
			amountIn: big.NewInt(150),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(50)),
			},
			expectedRoutes: [][]*PathV2{
				{
					{From: network1, AmountIn: (*hexutil.Big)(big.NewInt(100)), AmountInLocked: false},
					{From: network1, AmountIn: (*hexutil.Big)(big.NewInt(100)), AmountInLocked: false},
				},
			},
		},
		{
			name: "Partial locked amounts",
			routes: [][]*PathV2{
				{
					{From: network1, AmountIn: (*hexutil.Big)(big.NewInt(100))},
					{From: network2, AmountIn: (*hexutil.Big)(big.NewInt(100))},
					{From: network3, AmountIn: (*hexutil.Big)(big.NewInt(200))},
				},
			},
			amountIn: big.NewInt(250),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(50)),
				2: (*hexutil.Big)(big.NewInt(0)),
				3: (*hexutil.Big)(big.NewInt(100)),
			},
			expectedRoutes: [][]*PathV2{
				{
					{From: network1, AmountIn: (*hexutil.Big)(big.NewInt(100)), AmountInLocked: true},
					{From: network2, AmountIn: (*hexutil.Big)(big.NewInt(100)), AmountInLocked: false},
					{From: network3, AmountIn: (*hexutil.Big)(big.NewInt(200)), AmountInLocked: true},
				},
			},
		},
		{
			name: "Mixed networks with sufficient capacity",
			routes: [][]*PathV2{
				{
					{From: network1, AmountIn: (*hexutil.Big)(big.NewInt(100))},
					{From: network3, AmountIn: (*hexutil.Big)(big.NewInt(200))},
				},
			},
			amountIn: big.NewInt(250),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(100)),
				3: (*hexutil.Big)(big.NewInt(200)),
			},
			expectedRoutes: [][]*PathV2{
				{
					{From: network1, AmountIn: (*hexutil.Big)(big.NewInt(100)), AmountInLocked: true},
					{From: network3, AmountIn: (*hexutil.Big)(big.NewInt(200)), AmountInLocked: true},
				},
			},
		},
		{
			name: "Mixed networks with insufficient capacity",
			routes: [][]*PathV2{
				{
					{From: network1, AmountIn: (*hexutil.Big)(big.NewInt(100))},
					{From: network3, AmountIn: (*hexutil.Big)(big.NewInt(100))},
				},
			},
			amountIn: big.NewInt(250),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(50)),
				3: (*hexutil.Big)(big.NewInt(100)),
			},
			expectedRoutes: [][]*PathV2{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filteredRoutes := filterCapacityValidationV2(tt.routes, tt.amountIn, tt.fromLockedAmount)
			assert.Equal(t, tt.expectedRoutes, filteredRoutes)
		})
	}
}

func TestFilterRoutesV2(t *testing.T) {
	fromLockedAmount := map[uint64]*hexutil.Big{
		1: (*hexutil.Big)(big.NewInt(50)),
		2: (*hexutil.Big)(big.NewInt(100)),
	}

	routes := [][]*PathV2{
		{
			{From: &params.Network{ChainID: 1}, AmountIn: (*hexutil.Big)(big.NewInt(100))},
			{From: &params.Network{ChainID: 2}, AmountIn: (*hexutil.Big)(big.NewInt(200))},
		},
		{
			{From: &params.Network{ChainID: 3}, AmountIn: (*hexutil.Big)(big.NewInt(100))},
			{From: &params.Network{ChainID: 4}, AmountIn: (*hexutil.Big)(big.NewInt(50))},
		},
	}

	amountIn := big.NewInt(120)

	expectedRoutes := [][]*PathV2{
		{
			{From: &params.Network{ChainID: 1}, AmountIn: (*hexutil.Big)(big.NewInt(50)), AmountInLocked: true},
			{From: &params.Network{ChainID: 2}, AmountIn: (*hexutil.Big)(big.NewInt(200))},
		},
	}

	filteredRoutes := filterRoutesV2(routes, amountIn, fromLockedAmount)
	assert.Equal(t, expectedRoutes, filteredRoutes)
}
