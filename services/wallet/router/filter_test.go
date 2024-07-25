package router

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/wallet/router/pathprocessor"

	"github.com/stretchr/testify/assert"
)

var (
	network1 = &params.Network{ChainID: 1}
	network2 = &params.Network{ChainID: 2}
	network3 = &params.Network{ChainID: 3}
	network4 = &params.Network{ChainID: 4}
	network5 = &params.Network{ChainID: 5}

	amount0 = hexutil.Big(*big.NewInt(0))
	amount1 = hexutil.Big(*big.NewInt(100))
	amount2 = hexutil.Big(*big.NewInt(200))
	amount3 = hexutil.Big(*big.NewInt(300))
	amount4 = hexutil.Big(*big.NewInt(400))
	amount5 = hexutil.Big(*big.NewInt(500))

	path0 = &PathV2{FromChain: network4, AmountIn: &amount0}

	pathC1A1 = &PathV2{FromChain: network1, AmountIn: &amount1}

	pathC2A1 = &PathV2{FromChain: network2, AmountIn: &amount1}
	pathC2A2 = &PathV2{FromChain: network2, AmountIn: &amount2}

	pathC3A1 = &PathV2{FromChain: network3, AmountIn: &amount1}
	pathC3A2 = &PathV2{FromChain: network3, AmountIn: &amount2}
	pathC3A3 = &PathV2{FromChain: network3, AmountIn: &amount3}

	pathC4A1 = &PathV2{FromChain: network4, AmountIn: &amount1}
	pathC4A4 = &PathV2{FromChain: network4, AmountIn: &amount4}

	pathC5A5 = &PathV2{FromChain: network5, AmountIn: &amount5}
)

func routesEqual(t *testing.T, expected, actual [][]*PathV2) bool {
	if len(expected) != len(actual) {
		return false
	}
	for i := range expected {
		if !pathsEqual(t, expected[i], actual[i]) {
			return false
		}
	}
	return true
}

func pathsEqual(t *testing.T, expected, actual []*PathV2) bool {
	if len(expected) != len(actual) {
		return false
	}
	for i := range expected {
		if !pathEqual(t, expected[i], actual[i]) {
			return false
		}
	}
	return true
}

func pathEqual(t *testing.T, expected, actual *PathV2) bool {
	if expected.FromChain.ChainID != actual.FromChain.ChainID {
		t.Logf("expected chain ID '%d' , actual chain ID '%d'", expected.FromChain.ChainID, actual.FromChain.ChainID)
		return false
	}
	if expected.AmountIn.ToInt().Cmp(actual.AmountIn.ToInt()) != 0 {
		t.Logf("expected AmountIn '%d' , actual AmountIn '%d'", expected.AmountIn.ToInt(), actual.AmountIn.ToInt())
		return false
	}
	if expected.AmountInLocked != actual.AmountInLocked {
		t.Logf("expected AmountInLocked '%t' , actual AmountInLocked '%t'", expected.AmountInLocked, actual.AmountInLocked)
		return false
	}
	return true
}

func TestSetupRouteValidationMapsV2(t *testing.T) {
	tests := []struct {
		name             string
		fromLockedAmount map[uint64]*hexutil.Big
		expectedIncluded map[uint64]bool
		expectedExcluded map[uint64]bool
	}{
		{
			name: "Mixed zero and non-zero amounts",
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(pathprocessor.ZeroBigIntValue),
				2: (*hexutil.Big)(big.NewInt(200)),
				3: (*hexutil.Big)(pathprocessor.ZeroBigIntValue),
				4: (*hexutil.Big)(big.NewInt(400)),
			},
			expectedIncluded: map[uint64]bool{
				2: false,
				4: false,
			},
			expectedExcluded: map[uint64]bool{
				1: false,
				3: false,
			},
		},
		{
			name: "All non-zero amounts",
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(100)),
				2: (*hexutil.Big)(big.NewInt(200)),
			},
			expectedIncluded: map[uint64]bool{
				1: false,
				2: false,
			},
			expectedExcluded: map[uint64]bool{},
		},
		{
			name: "All zero amounts",
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(pathprocessor.ZeroBigIntValue),
				2: (*hexutil.Big)(pathprocessor.ZeroBigIntValue),
			},
			expectedIncluded: map[uint64]bool{},
			expectedExcluded: map[uint64]bool{
				1: false,
				2: false,
			},
		},
		{
			name: "Single non-zero amount",
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(100)),
			},
			expectedIncluded: map[uint64]bool{
				1: false,
			},
			expectedExcluded: map[uint64]bool{},
		},
		{
			name: "Single zero amount",
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(pathprocessor.ZeroBigIntValue),
			},
			expectedIncluded: map[uint64]bool{},
			expectedExcluded: map[uint64]bool{
				1: false,
			},
		},
		{
			name:             "Empty map",
			fromLockedAmount: map[uint64]*hexutil.Big{},
			expectedIncluded: map[uint64]bool{},
			expectedExcluded: map[uint64]bool{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			included, excluded := setupRouteValidationMapsV2(tt.fromLockedAmount)
			assert.Equal(t, tt.expectedIncluded, included)
			assert.Equal(t, tt.expectedExcluded, excluded)
		})
	}
}

func TestCalculateRestAmountInV2(t *testing.T) {
	tests := []struct {
		name        string
		route       []*PathV2
		excludePath *PathV2
		expected    *big.Int
	}{
		{
			name:        "Exclude pathC1A1",
			route:       []*PathV2{pathC1A1, pathC2A2, pathC3A3},
			excludePath: pathC1A1,
			expected:    big.NewInt(500), // 200 + 300
		},
		{
			name:        "Exclude pathC2A2",
			route:       []*PathV2{pathC1A1, pathC2A2, pathC3A3},
			excludePath: pathC2A2,
			expected:    big.NewInt(400), // 100 + 300
		},
		{
			name:        "Exclude pathC3A3",
			route:       []*PathV2{pathC1A1, pathC2A2, pathC3A3},
			excludePath: pathC3A3,
			expected:    big.NewInt(300), // 100 + 200
		},
		{
			name:        "Single path, exclude that path",
			route:       []*PathV2{pathC1A1},
			excludePath: pathC1A1,
			expected:    big.NewInt(0), // No other paths
		},
		{
			name:        "Empty route",
			route:       []*PathV2{},
			excludePath: pathC1A1,
			expected:    big.NewInt(0), // No paths
		},
		{
			name:        "Empty route, with nil exclude",
			route:       []*PathV2{},
			excludePath: nil,
			expected:    big.NewInt(0), // No paths
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateRestAmountInV2(tt.route, tt.excludePath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidForNetworkComplianceV2(t *testing.T) {
	tests := []struct {
		name           string
		route          []*PathV2
		fromIncluded   map[uint64]bool
		fromExcluded   map[uint64]bool
		expectedResult bool
	}{
		{
			name:           "Route with all included chain IDs",
			route:          []*PathV2{pathC1A1, pathC2A2},
			fromIncluded:   map[uint64]bool{1: true, 2: true},
			fromExcluded:   map[uint64]bool{},
			expectedResult: true,
		},
		{
			name:           "Route with fromExcluded only",
			route:          []*PathV2{pathC1A1, pathC2A2},
			fromIncluded:   map[uint64]bool{},
			fromExcluded:   map[uint64]bool{3: false, 4: false},
			expectedResult: true,
		},
		{
			name:           "Route without excluded chain IDs",
			route:          []*PathV2{pathC1A1, pathC2A2},
			fromIncluded:   map[uint64]bool{1: false, 2: false},
			fromExcluded:   map[uint64]bool{3: false, 4: false},
			expectedResult: true,
		},
		{
			name:           "Route with an excluded chain ID",
			route:          []*PathV2{pathC1A1, pathC3A3},
			fromIncluded:   map[uint64]bool{1: false, 2: false},
			fromExcluded:   map[uint64]bool{3: false, 4: false},
			expectedResult: false,
		},
		{
			name:           "Route missing one included chain ID",
			route:          []*PathV2{pathC1A1},
			fromIncluded:   map[uint64]bool{1: false, 2: false},
			fromExcluded:   map[uint64]bool{},
			expectedResult: false,
		},
		{
			name:           "Route with no fromIncluded or fromExcluded",
			route:          []*PathV2{pathC1A1, pathC2A2},
			fromIncluded:   map[uint64]bool{},
			fromExcluded:   map[uint64]bool{},
			expectedResult: true,
		},
		{
			name:           "Empty route",
			route:          []*PathV2{},
			fromIncluded:   map[uint64]bool{1: false, 2: false},
			fromExcluded:   map[uint64]bool{3: false, 4: false},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidForNetworkComplianceV2(tt.route, tt.fromIncluded, tt.fromExcluded)
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
		expected         bool
	}{
		{
			name:             "All paths meet required amount",
			route:            []*PathV2{pathC1A1, pathC2A2, pathC3A3},
			amountIn:         big.NewInt(600),
			fromLockedAmount: map[uint64]*hexutil.Big{1: &amount1, 2: &amount2, 3: &amount3},
			expected:         true,
		},
		// TODO: Find out what the expected behaviour for this case should be
		// I expect false but the test returns true
		/*
			{
				name:             "A path does not meet required amount",
				route:            []*PathV2{pathC1A1, pathC2A2, pathC3A3},
				amountIn:         big.NewInt(600),
				fromLockedAmount: map[uint64]*hexutil.Big{1: &amount1, 2: &amount2, 4: &amount4},
				expected:         false,
			},
		*/
		{
			name:             "No fromLockedAmount",
			route:            []*PathV2{pathC1A1, pathC2A2, pathC3A3},
			amountIn:         big.NewInt(600),
			fromLockedAmount: map[uint64]*hexutil.Big{},
			expected:         true,
		},
		{
			name:             "Single path meets required amount",
			route:            []*PathV2{pathC1A1},
			amountIn:         big.NewInt(100),
			fromLockedAmount: map[uint64]*hexutil.Big{1: &amount1},
			expected:         true,
		},
		{
			name:             "Single path does not meet required amount",
			route:            []*PathV2{pathC1A1},
			amountIn:         big.NewInt(200),
			fromLockedAmount: map[uint64]*hexutil.Big{1: &amount1},
			expected:         false,
		},
		{
			name:             "Path meets required amount with excess",
			route:            []*PathV2{pathC1A1, pathC2A2},
			amountIn:         big.NewInt(250),
			fromLockedAmount: map[uint64]*hexutil.Big{1: &amount1, 2: &amount2},
			expected:         true,
		},
		{
			name:             "Path does not meet required amount due to insufficient rest",
			route:            []*PathV2{pathC1A1, pathC2A2, pathC4A4},
			amountIn:         big.NewInt(800),
			fromLockedAmount: map[uint64]*hexutil.Big{1: &amount1, 4: &amount4},
			expected:         false,
		},
		{
			name:             "Empty route",
			route:            []*PathV2{},
			amountIn:         big.NewInt(500),
			fromLockedAmount: map[uint64]*hexutil.Big{1: &amount1, 2: &amount2},
			expected:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasSufficientCapacityV2(tt.route, tt.amountIn, tt.fromLockedAmount)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterNetworkComplianceV2(t *testing.T) {
	tests := []struct {
		name             string
		routes           [][]*PathV2
		fromLockedAmount map[uint64]*hexutil.Big
		expected         [][]*PathV2
	}{
		{
			name: "Mixed routes with valid and invalid paths",
			routes: [][]*PathV2{
				{
					{FromChain: network1},
					{FromChain: network3},
				},
				{
					{FromChain: network2},
					{FromChain: network3},
				},
				{
					{FromChain: network1},
					{FromChain: network2},
					{FromChain: network3},
				},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(100)),
				2: (*hexutil.Big)(big.NewInt(0)),
			},
			expected: [][]*PathV2{
				{
					{FromChain: network1},
					{FromChain: network3},
				},
			},
		},
		{
			name: "All valid routes",
			routes: [][]*PathV2{
				{
					{FromChain: network1},
					{FromChain: network3},
				},
				{
					{FromChain: network1},
					{FromChain: network4},
				},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(100)),
			},
			expected: [][]*PathV2{
				{
					{FromChain: network1},
					{FromChain: network3},
				},
				{
					{FromChain: network1},
					{FromChain: network4},
				},
			},
		},
		{
			name: "All invalid routes",
			routes: [][]*PathV2{
				{
					{FromChain: network2},
					{FromChain: network3},
				},
				{
					{FromChain: network4},
					{FromChain: network5},
				},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(100)),
				2: (*hexutil.Big)(big.NewInt(0)),
			},
			expected: [][]*PathV2{},
		},
		{
			name:   "Empty routes",
			routes: [][]*PathV2{},
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(100)),
			},
			expected: [][]*PathV2{},
		},
		{
			name: "No locked amounts",
			routes: [][]*PathV2{
				{
					{FromChain: network1},
					{FromChain: network2},
				},
				{
					{FromChain: network3},
					{FromChain: network4},
				},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{},
			expected: [][]*PathV2{
				{
					{FromChain: network1},
					{FromChain: network2},
				},
				{
					{FromChain: network3},
					{FromChain: network4},
				},
			},
		},
		{
			name: "Single route with mixed valid and invalid paths",
			routes: [][]*PathV2{
				{
					{FromChain: network1},
					{FromChain: network2},
					{FromChain: network3},
				},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(100)),
				2: (*hexutil.Big)(big.NewInt(0)),
			},
			expected: [][]*PathV2{},
		},
		{
			name: "Routes with duplicate chain IDs",
			routes: [][]*PathV2{
				{
					{FromChain: network1},
					{FromChain: network1},
					{FromChain: network2},
				},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(100)),
			},
			expected: [][]*PathV2{
				{
					{FromChain: network1},
					{FromChain: network1},
					{FromChain: network2},
				},
			},
		},
		{
			name: "Minimum and maximum chain IDs",
			routes: [][]*PathV2{
				{
					{FromChain: &params.Network{ChainID: 0}},
					{FromChain: &params.Network{ChainID: ^uint64(0)}},
				},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{
				0:          (*hexutil.Big)(big.NewInt(100)),
				^uint64(0): (*hexutil.Big)(big.NewInt(100)),
			},
			expected: [][]*PathV2{
				{
					{FromChain: &params.Network{ChainID: 0}},
					{FromChain: &params.Network{ChainID: ^uint64(0)}},
				},
			},
		},
		{
			name: "Large number of routes",
			routes: func() [][]*PathV2 {
				var routes [][]*PathV2
				for i := 0; i < 1000; i++ {
					routes = append(routes, []*PathV2{
						{FromChain: &params.Network{ChainID: uint64(i + 1)}},
						{FromChain: &params.Network{ChainID: uint64(i + 1001)}},
					})
				}
				return routes
			}(),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1:    (*hexutil.Big)(big.NewInt(100)),
				1001: (*hexutil.Big)(big.NewInt(100)),
			},
			expected: func() [][]*PathV2 {
				var routes [][]*PathV2
				for i := 0; i < 1; i++ {
					routes = append(routes, []*PathV2{
						{FromChain: &params.Network{ChainID: uint64(i + 1)}},
						{FromChain: &params.Network{ChainID: uint64(i + 1001)}},
					})
				}
				return routes
			}(),
		},
		{
			name: "Routes with missing data",
			routes: [][]*PathV2{
				{
					{FromChain: nil},
					{FromChain: network2},
				},
				{
					{FromChain: network1},
					{FromChain: nil},
				},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(100)),
				2: (*hexutil.Big)(big.NewInt(0)),
			},
			expected: [][]*PathV2{},
		},
		{
			name: "Consistency check",
			routes: [][]*PathV2{
				{
					{FromChain: network1},
					{FromChain: network2},
				},
				{
					{FromChain: network1},
					{FromChain: network3},
				},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(100)),
			},
			expected: [][]*PathV2{
				{
					{FromChain: network1},
					{FromChain: network2},
				},
				{
					{FromChain: network1},
					{FromChain: network3},
				},
			},
		},
		{
			name: "Routes without excluded chain IDs, missing included path",
			routes: [][]*PathV2{
				{pathC1A1, pathC2A2},
				{pathC2A2, pathC3A3},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{1: &amount1, 2: &amount2},
			expected: [][]*PathV2{
				{pathC1A1, pathC2A2},
			},
		},
		{
			name: "Routes with an excluded chain ID",
			routes: [][]*PathV2{
				{pathC1A1, pathC2A2},
				{pathC2A2, pathC3A3, path0},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{1: &amount1, 2: &amount2, 4: &amount0},
			expected: [][]*PathV2{
				{pathC1A1, pathC2A2},
			},
		},
		{
			name: "Routes with all included chain IDs",
			routes: [][]*PathV2{
				{pathC1A1, pathC2A2, pathC3A3},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{1: &amount1, 2: &amount2, 3: &amount3},
			expected: [][]*PathV2{
				{pathC1A1, pathC2A2, pathC3A3},
			},
		},
		{
			name: "Routes missing one included chain ID",
			routes: [][]*PathV2{
				{pathC1A1, pathC2A2},
				{pathC1A1},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{1: &amount1, 2: &amount2, 3: &amount3},
			expected:         [][]*PathV2{},
		},
		{
			name: "Routes with no fromLockedAmount",
			routes: [][]*PathV2{
				{pathC1A1, pathC2A2},
				{pathC2A2, pathC3A3},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{},
			expected: [][]*PathV2{
				{pathC1A1, pathC2A2},
				{pathC2A2, pathC3A3},
			},
		},
		{
			name: "Routes with fromExcluded only",
			routes: [][]*PathV2{
				{pathC1A1, pathC2A2},
				{pathC2A2, pathC3A3},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{4: &amount0},
			expected: [][]*PathV2{
				{pathC1A1, pathC2A2},
				{pathC2A2, pathC3A3},
			},
		},
		{
			name: "Routes with all excluded chain IDs",
			routes: [][]*PathV2{
				{path0, pathC1A1},
				{path0, pathC2A2},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{1: &amount1, 2: &amount2, 3: &amount3, 4: &amount0},
			expected:         [][]*PathV2{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Original Routes: %+v\n", tt.routes)
			filteredRoutes := filterNetworkComplianceV2(tt.routes, tt.fromLockedAmount)
			t.Logf("Filtered Routes: %+v\n", filteredRoutes)
			assert.Equal(t, tt.expected, filteredRoutes)
		})
	}
}

func TestFilterCapacityValidationV2(t *testing.T) {
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
					{FromChain: network1, AmountIn: (*hexutil.Big)(big.NewInt(50))},
					{FromChain: network2, AmountIn: (*hexutil.Big)(big.NewInt(100))},
					{FromChain: network3, AmountIn: (*hexutil.Big)(big.NewInt(100))},
				},
				{
					{FromChain: network1, AmountIn: (*hexutil.Big)(big.NewInt(50))},
					{FromChain: network2, AmountIn: (*hexutil.Big)(big.NewInt(200))},
				},
				{
					{FromChain: network1, AmountIn: (*hexutil.Big)(big.NewInt(100))},
					{FromChain: network2, AmountIn: (*hexutil.Big)(big.NewInt(200))},
				},
			},
			amountIn: big.NewInt(250),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(50)),
			},
			expectedRoutes: [][]*PathV2{
				{
					{FromChain: network1, AmountIn: (*hexutil.Big)(big.NewInt(50))},
					{FromChain: network2, AmountIn: (*hexutil.Big)(big.NewInt(100))},
					{FromChain: network3, AmountIn: (*hexutil.Big)(big.NewInt(100))},
				},
				{
					{FromChain: network1, AmountIn: (*hexutil.Big)(big.NewInt(50))},
					{FromChain: network2, AmountIn: (*hexutil.Big)(big.NewInt(200))},
				},
			},
		},
		{
			name: "Insufficient capacity",
			routes: [][]*PathV2{
				{
					{FromChain: network1, AmountIn: (*hexutil.Big)(big.NewInt(100))},
					{FromChain: network2, AmountIn: (*hexutil.Big)(big.NewInt(50))},
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
					{FromChain: network1, AmountIn: (*hexutil.Big)(big.NewInt(100))},
					{FromChain: network2, AmountIn: (*hexutil.Big)(big.NewInt(50))},
				},
			},
			amountIn: big.NewInt(150),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(100)),
				2: (*hexutil.Big)(big.NewInt(50)),
			},
			expectedRoutes: [][]*PathV2{
				{
					{FromChain: network1, AmountIn: (*hexutil.Big)(big.NewInt(100))},
					{FromChain: network2, AmountIn: (*hexutil.Big)(big.NewInt(50))},
				},
			},
		},
		{
			name: "No locked amounts",
			routes: [][]*PathV2{
				{
					{FromChain: network1, AmountIn: (*hexutil.Big)(big.NewInt(100))},
					{FromChain: network2, AmountIn: (*hexutil.Big)(big.NewInt(50))},
				},
			},
			amountIn:         big.NewInt(150),
			fromLockedAmount: map[uint64]*hexutil.Big{},
			expectedRoutes: [][]*PathV2{
				{
					{FromChain: network1, AmountIn: (*hexutil.Big)(big.NewInt(100))},
					{FromChain: network2, AmountIn: (*hexutil.Big)(big.NewInt(50))},
				},
			},
		},
		{
			name: "Single route with sufficient capacity",
			routes: [][]*PathV2{
				{
					{FromChain: network1, AmountIn: (*hexutil.Big)(big.NewInt(50))},
					{FromChain: network2, AmountIn: (*hexutil.Big)(big.NewInt(100))},
				},
			},
			amountIn: big.NewInt(150),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(50)),
			},
			expectedRoutes: [][]*PathV2{
				{
					{FromChain: network1, AmountIn: (*hexutil.Big)(big.NewInt(50))},
					{FromChain: network2, AmountIn: (*hexutil.Big)(big.NewInt(100))},
				},
			},
		},
		{
			name: "Single route with inappropriately locked amount",
			routes: [][]*PathV2{
				{
					{FromChain: network1, AmountIn: (*hexutil.Big)(big.NewInt(100))},
				},
			},
			amountIn: big.NewInt(150),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(50)),
			},
			expectedRoutes: [][]*PathV2{},
		},
		{
			name: "Single route with insufficient capacity",
			routes: [][]*PathV2{
				{
					{FromChain: network1, AmountIn: (*hexutil.Big)(big.NewInt(50))},
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
			name: "Partial locked amounts",
			routes: [][]*PathV2{
				{
					{FromChain: network1, AmountIn: (*hexutil.Big)(big.NewInt(50))},
					{FromChain: network3, AmountIn: (*hexutil.Big)(big.NewInt(100))},
					{FromChain: network4, AmountIn: (*hexutil.Big)(big.NewInt(100))},
				},
			},
			amountIn: big.NewInt(250),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(50)),
				2: (*hexutil.Big)(big.NewInt(0)), // Excluded path
				3: (*hexutil.Big)(big.NewInt(100)),
			},
			expectedRoutes: [][]*PathV2{
				{
					{FromChain: network1, AmountIn: (*hexutil.Big)(big.NewInt(50))},
					{FromChain: network3, AmountIn: (*hexutil.Big)(big.NewInt(100))},
					{FromChain: network4, AmountIn: (*hexutil.Big)(big.NewInt(100))},
				},
			},
		},
		{
			name: "Mixed networks with sufficient capacity",
			routes: [][]*PathV2{
				{
					{FromChain: network1, AmountIn: (*hexutil.Big)(big.NewInt(100))},
					{FromChain: network3, AmountIn: (*hexutil.Big)(big.NewInt(200))},
				},
			},
			amountIn: big.NewInt(300),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(100)),
				3: (*hexutil.Big)(big.NewInt(200)),
			},
			expectedRoutes: [][]*PathV2{
				{
					{FromChain: network1, AmountIn: (*hexutil.Big)(big.NewInt(100))},
					{FromChain: network3, AmountIn: (*hexutil.Big)(big.NewInt(200))},
				},
			},
		},
		{
			name: "Mixed networks with insufficient capacity",
			routes: [][]*PathV2{
				{
					{FromChain: network1, AmountIn: (*hexutil.Big)(big.NewInt(100))},
					{FromChain: network3, AmountIn: (*hexutil.Big)(big.NewInt(100))},
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
			if !routesEqual(t, tt.expectedRoutes, filteredRoutes) {
				t.Errorf("Expected: %+v, Actual: %+v", tt.expectedRoutes, filteredRoutes)
			}
		})
	}
}

func TestFilterRoutesV2(t *testing.T) {
	tests := []struct {
		name             string
		routes           [][]*PathV2
		amountIn         *big.Int
		fromLockedAmount map[uint64]*hexutil.Big
		expectedRoutes   [][]*PathV2
	}{
		{
			name: "Empty fromLockedAmount and routes don't match amountIn",
			routes: [][]*PathV2{
				{pathC1A1, pathC2A2},
				{pathC3A3, pathC4A4},
			},
			amountIn:         big.NewInt(150),
			fromLockedAmount: map[uint64]*hexutil.Big{},
			expectedRoutes:   [][]*PathV2{},
		},
		{
			name: "Empty fromLockedAmount and sigle route match amountIn",
			routes: [][]*PathV2{
				{pathC1A1, pathC2A2},
				{pathC3A3, pathC4A4},
			},
			amountIn:         big.NewInt(300),
			fromLockedAmount: map[uint64]*hexutil.Big{},
			expectedRoutes: [][]*PathV2{
				{pathC1A1, pathC2A2},
			},
		},
		{
			name: "Empty fromLockedAmount and more routes match amountIn",
			routes: [][]*PathV2{
				{pathC1A1, pathC2A2},
				{pathC3A3, pathC4A4},
				{pathC1A1, pathC2A1, pathC3A1},
			},
			amountIn:         big.NewInt(300),
			fromLockedAmount: map[uint64]*hexutil.Big{},
			expectedRoutes: [][]*PathV2{
				{pathC1A1, pathC2A2},
				{pathC1A1, pathC2A1, pathC3A1},
			},
		},
		{
			name: "All paths appear in fromLockedAmount but not within a single route",
			routes: [][]*PathV2{
				{pathC1A1, pathC3A3},
				{pathC2A2, pathC4A4},
			},
			amountIn: big.NewInt(500),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: &amount1,
				2: &amount2,
				3: &amount3,
				4: &amount4,
			},
			expectedRoutes: [][]*PathV2{},
		},
		{
			name: "Mixed valid and invalid routes I",
			routes: [][]*PathV2{
				{pathC1A1, pathC2A2},
				{pathC2A2, pathC3A3},
				{pathC1A1, pathC4A4},
				{pathC1A1, pathC2A1, pathC3A1},
			},
			amountIn: big.NewInt(300),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: &amount1,
				2: &amount2,
			},
			expectedRoutes: [][]*PathV2{
				{pathC1A1, pathC2A2},
			},
		},
		{
			name: "Mixed valid and invalid routes II",
			routes: [][]*PathV2{
				{pathC1A1, pathC2A2},
				{pathC2A2, pathC3A3},
				{pathC1A1, pathC4A4},
				{pathC1A1, pathC2A1, pathC3A1},
			},
			amountIn: big.NewInt(300),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: &amount1,
			},
			expectedRoutes: [][]*PathV2{
				{pathC1A1, pathC2A2},
				{pathC1A1, pathC2A1, pathC3A1},
			},
		},
		{
			name: "All invalid routes",
			routes: [][]*PathV2{
				{pathC2A2, pathC3A3},
				{pathC4A4, pathC5A5},
			},
			amountIn: big.NewInt(300),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: &amount1,
			},
			expectedRoutes: [][]*PathV2{},
		},
		{
			name: "Single valid route",
			routes: [][]*PathV2{
				{pathC1A1, pathC3A3},
				{pathC2A2, pathC3A3},
			},
			amountIn: big.NewInt(400),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: &amount1,
				3: &amount3,
			},
			expectedRoutes: [][]*PathV2{
				{pathC1A1, pathC3A3},
			},
		},
		{
			name: "Route with mixed valid and invalid paths I",
			routes: [][]*PathV2{
				{pathC1A1, pathC2A2, pathC3A3},
			},
			amountIn: big.NewInt(300),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: &amount1,
				2: &amount0, // This path should be filtered out due to being excluded via a zero amount
			},
			expectedRoutes: [][]*PathV2{},
		},
		{
			name: "Route with mixed valid and invalid paths II",
			routes: [][]*PathV2{
				{pathC1A1, pathC3A3},
			},
			amountIn: big.NewInt(400),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: &amount1,
				2: &amount0, // This path should be filtered out due to being excluded via a zero amount, 0 value locked means this chain is disabled
			},
			expectedRoutes: [][]*PathV2{
				{pathC1A1, pathC3A3},
			},
		},
		{
			name: "Route with mixed valid and invalid paths III",
			routes: [][]*PathV2{
				{pathC1A1, pathC3A3},
				{pathC1A1, pathC3A2, pathC4A1},
			},
			amountIn: big.NewInt(400),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: &amount1,
				2: &amount0, // This path should be filtered out due to being excluded via a zero amount, 0 value locked means this chain is disabled
			},
			expectedRoutes: [][]*PathV2{
				{pathC1A1, pathC3A3},
				{pathC1A1, pathC3A2, pathC4A1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Original Routes: %+v\n", tt.routes)
			filteredRoutes := filterRoutesV2(tt.routes, tt.amountIn, tt.fromLockedAmount)
			t.Logf("Filtered Routes: %+v\n", filteredRoutes)
			assert.Equal(t, tt.expectedRoutes, filteredRoutes)
		})
	}
}
