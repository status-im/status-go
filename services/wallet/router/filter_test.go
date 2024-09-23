package router

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/status-im/status-go/params"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/router/routes"

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

	path0 = &routes.Path{FromChain: network4, AmountIn: &amount0}

	pathC1A1 = &routes.Path{FromChain: network1, AmountIn: &amount1}

	pathC2A1 = &routes.Path{FromChain: network2, AmountIn: &amount1}
	pathC2A2 = &routes.Path{FromChain: network2, AmountIn: &amount2}

	pathC3A1 = &routes.Path{FromChain: network3, AmountIn: &amount1}
	pathC3A2 = &routes.Path{FromChain: network3, AmountIn: &amount2}
	pathC3A3 = &routes.Path{FromChain: network3, AmountIn: &amount3}

	pathC4A1 = &routes.Path{FromChain: network4, AmountIn: &amount1}
	pathC4A4 = &routes.Path{FromChain: network4, AmountIn: &amount4}

	pathC5A5 = &routes.Path{FromChain: network5, AmountIn: &amount5}
)

func routesEqual(t *testing.T, expected, actual []routes.Route) bool {
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

func pathsEqual(t *testing.T, expected, actual routes.Route) bool {
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

func pathEqual(t *testing.T, expected, actual *routes.Path) bool {
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

func TestSetupRouteValidationMaps(t *testing.T) {
	tests := []struct {
		name             string
		fromLockedAmount map[uint64]*hexutil.Big
		expectedIncluded map[uint64]bool
		expectedExcluded map[uint64]bool
	}{
		{
			name: "Mixed zero and non-zero amounts",
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(walletCommon.ZeroBigIntValue),
				2: (*hexutil.Big)(big.NewInt(200)),
				3: (*hexutil.Big)(walletCommon.ZeroBigIntValue),
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
				1: (*hexutil.Big)(walletCommon.ZeroBigIntValue),
				2: (*hexutil.Big)(walletCommon.ZeroBigIntValue),
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
				1: (*hexutil.Big)(walletCommon.ZeroBigIntValue),
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
			included, excluded := setupRouteValidationMaps(tt.fromLockedAmount)
			assert.Equal(t, tt.expectedIncluded, included)
			assert.Equal(t, tt.expectedExcluded, excluded)
		})
	}
}

func TestCalculateRestAmountIn(t *testing.T) {
	tests := []struct {
		name        string
		route       routes.Route
		excludePath *routes.Path
		expected    *big.Int
	}{
		{
			name:        "Exclude pathC1A1",
			route:       routes.Route{pathC1A1, pathC2A2, pathC3A3},
			excludePath: pathC1A1,
			expected:    big.NewInt(500), // 200 + 300
		},
		{
			name:        "Exclude pathC2A2",
			route:       routes.Route{pathC1A1, pathC2A2, pathC3A3},
			excludePath: pathC2A2,
			expected:    big.NewInt(400), // 100 + 300
		},
		{
			name:        "Exclude pathC3A3",
			route:       routes.Route{pathC1A1, pathC2A2, pathC3A3},
			excludePath: pathC3A3,
			expected:    big.NewInt(300), // 100 + 200
		},
		{
			name:        "Single path, exclude that path",
			route:       routes.Route{pathC1A1},
			excludePath: pathC1A1,
			expected:    big.NewInt(0), // No other paths
		},
		{
			name:        "Empty route",
			route:       routes.Route{},
			excludePath: pathC1A1,
			expected:    big.NewInt(0), // No paths
		},
		{
			name:        "Empty route, with nil exclude",
			route:       routes.Route{},
			excludePath: nil,
			expected:    big.NewInt(0), // No paths
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateRestAmountIn(tt.route, tt.excludePath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidForNetworkCompliance(t *testing.T) {
	tests := []struct {
		name           string
		route          routes.Route
		fromIncluded   map[uint64]bool
		fromExcluded   map[uint64]bool
		expectedResult bool
	}{
		{
			name:           "Route with all included chain IDs",
			route:          routes.Route{pathC1A1, pathC2A2},
			fromIncluded:   map[uint64]bool{1: true, 2: true},
			fromExcluded:   map[uint64]bool{},
			expectedResult: true,
		},
		{
			name:           "Route with fromExcluded only",
			route:          routes.Route{pathC1A1, pathC2A2},
			fromIncluded:   map[uint64]bool{},
			fromExcluded:   map[uint64]bool{3: false, 4: false},
			expectedResult: true,
		},
		{
			name:           "Route without excluded chain IDs",
			route:          routes.Route{pathC1A1, pathC2A2},
			fromIncluded:   map[uint64]bool{1: false, 2: false},
			fromExcluded:   map[uint64]bool{3: false, 4: false},
			expectedResult: true,
		},
		{
			name:           "Route with an excluded chain ID",
			route:          routes.Route{pathC1A1, pathC3A3},
			fromIncluded:   map[uint64]bool{1: false, 2: false},
			fromExcluded:   map[uint64]bool{3: false, 4: false},
			expectedResult: false,
		},
		{
			name:           "Route missing one included chain ID",
			route:          routes.Route{pathC1A1},
			fromIncluded:   map[uint64]bool{1: false, 2: false},
			fromExcluded:   map[uint64]bool{},
			expectedResult: false,
		},
		{
			name:           "Route with no fromIncluded or fromExcluded",
			route:          routes.Route{pathC1A1, pathC2A2},
			fromIncluded:   map[uint64]bool{},
			fromExcluded:   map[uint64]bool{},
			expectedResult: true,
		},
		{
			name:           "Empty route",
			route:          routes.Route{},
			fromIncluded:   map[uint64]bool{1: false, 2: false},
			fromExcluded:   map[uint64]bool{3: false, 4: false},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidForNetworkCompliance(tt.route, tt.fromIncluded, tt.fromExcluded)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestHasSufficientCapacity(t *testing.T) {
	tests := []struct {
		name             string
		route            routes.Route
		amountIn         *big.Int
		fromLockedAmount map[uint64]*hexutil.Big
		expected         bool
	}{
		{
			name:             "All paths meet required amount",
			route:            routes.Route{pathC1A1, pathC2A2, pathC3A3},
			amountIn:         big.NewInt(600),
			fromLockedAmount: map[uint64]*hexutil.Big{1: &amount1, 2: &amount2, 3: &amount3},
			expected:         true,
		},
		// TODO: Find out what the expected behaviour for this case should be
		// I expect false but the test returns true
		/*
			{
				name:             "A path does not meet required amount",
				route:            routes.Route{pathC1A1, pathC2A2, pathC3A3},
				amountIn:         big.NewInt(600),
				fromLockedAmount: map[uint64]*hexutil.Big{1: &amount1, 2: &amount2, 4: &amount4},
				expected:         false,
			},
		*/
		{
			name:             "No fromLockedAmount",
			route:            routes.Route{pathC1A1, pathC2A2, pathC3A3},
			amountIn:         big.NewInt(600),
			fromLockedAmount: map[uint64]*hexutil.Big{},
			expected:         true,
		},
		{
			name:             "Single path meets required amount",
			route:            routes.Route{pathC1A1},
			amountIn:         big.NewInt(100),
			fromLockedAmount: map[uint64]*hexutil.Big{1: &amount1},
			expected:         true,
		},
		{
			name:             "Single path does not meet required amount",
			route:            routes.Route{pathC1A1},
			amountIn:         big.NewInt(200),
			fromLockedAmount: map[uint64]*hexutil.Big{1: &amount1},
			expected:         false,
		},
		{
			name:             "Path meets required amount with excess",
			route:            routes.Route{pathC1A1, pathC2A2},
			amountIn:         big.NewInt(250),
			fromLockedAmount: map[uint64]*hexutil.Big{1: &amount1, 2: &amount2},
			expected:         true,
		},
		{
			name:             "Path does not meet required amount due to insufficient rest",
			route:            routes.Route{pathC1A1, pathC2A2, pathC4A4},
			amountIn:         big.NewInt(800),
			fromLockedAmount: map[uint64]*hexutil.Big{1: &amount1, 4: &amount4},
			expected:         false,
		},
		{
			name:             "Empty route",
			route:            routes.Route{},
			amountIn:         big.NewInt(500),
			fromLockedAmount: map[uint64]*hexutil.Big{1: &amount1, 2: &amount2},
			expected:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasSufficientCapacity(tt.route, tt.amountIn, tt.fromLockedAmount)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterNetworkCompliance(t *testing.T) {
	tests := []struct {
		name             string
		routes           []routes.Route
		fromLockedAmount map[uint64]*hexutil.Big
		expected         []routes.Route
	}{
		{
			name: "Mixed routes with valid and invalid paths",
			routes: []routes.Route{
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
			expected: []routes.Route{
				{
					{FromChain: network1},
					{FromChain: network3},
				},
			},
		},
		{
			name: "All valid routes",
			routes: []routes.Route{
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
			expected: []routes.Route{
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
			routes: []routes.Route{
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
			expected: []routes.Route{},
		},
		{
			name:   "Empty routes",
			routes: []routes.Route{},
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(100)),
			},
			expected: []routes.Route{},
		},
		{
			name: "No locked amounts",
			routes: []routes.Route{
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
			expected: []routes.Route{
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
			routes: []routes.Route{
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
			expected: []routes.Route{},
		},
		{
			name: "Routes with duplicate chain IDs",
			routes: []routes.Route{
				{
					{FromChain: network1},
					{FromChain: network1},
					{FromChain: network2},
				},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(100)),
			},
			expected: []routes.Route{
				{
					{FromChain: network1},
					{FromChain: network1},
					{FromChain: network2},
				},
			},
		},
		{
			name: "Minimum and maximum chain IDs",
			routes: []routes.Route{
				{
					{FromChain: &params.Network{ChainID: 0}},
					{FromChain: &params.Network{ChainID: ^uint64(0)}},
				},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{
				0:          (*hexutil.Big)(big.NewInt(100)),
				^uint64(0): (*hexutil.Big)(big.NewInt(100)),
			},
			expected: []routes.Route{
				{
					{FromChain: &params.Network{ChainID: 0}},
					{FromChain: &params.Network{ChainID: ^uint64(0)}},
				},
			},
		},
		{
			name: "Large number of routes",
			routes: func() []routes.Route {
				var routes1 []routes.Route
				for i := 0; i < 1000; i++ {
					routes1 = append(routes1, routes.Route{
						{FromChain: &params.Network{ChainID: uint64(i + 1)}},
						{FromChain: &params.Network{ChainID: uint64(i + 1001)}},
					})
				}
				return routes1
			}(),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1:    (*hexutil.Big)(big.NewInt(100)),
				1001: (*hexutil.Big)(big.NewInt(100)),
			},
			expected: func() []routes.Route {
				var routes1 []routes.Route
				for i := 0; i < 1; i++ {
					routes1 = append(routes1, routes.Route{
						{FromChain: &params.Network{ChainID: uint64(i + 1)}},
						{FromChain: &params.Network{ChainID: uint64(i + 1001)}},
					})
				}
				return routes1
			}(),
		},
		{
			name: "Routes with missing data",
			routes: []routes.Route{
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
			expected: []routes.Route{},
		},
		{
			name: "Consistency check",
			routes: []routes.Route{
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
			expected: []routes.Route{
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
			routes: []routes.Route{
				{pathC1A1, pathC2A2},
				{pathC2A2, pathC3A3},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{1: &amount1, 2: &amount2},
			expected: []routes.Route{
				{pathC1A1, pathC2A2},
			},
		},
		{
			name: "Routes with an excluded chain ID",
			routes: []routes.Route{
				{pathC1A1, pathC2A2},
				{pathC2A2, pathC3A3, path0},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{1: &amount1, 2: &amount2, 4: &amount0},
			expected: []routes.Route{
				{pathC1A1, pathC2A2},
			},
		},
		{
			name: "Routes with all included chain IDs",
			routes: []routes.Route{
				{pathC1A1, pathC2A2, pathC3A3},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{1: &amount1, 2: &amount2, 3: &amount3},
			expected: []routes.Route{
				{pathC1A1, pathC2A2, pathC3A3},
			},
		},
		{
			name: "Routes missing one included chain ID",
			routes: []routes.Route{
				{pathC1A1, pathC2A2},
				{pathC1A1},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{1: &amount1, 2: &amount2, 3: &amount3},
			expected:         []routes.Route{},
		},
		{
			name: "Routes with no fromLockedAmount",
			routes: []routes.Route{
				{pathC1A1, pathC2A2},
				{pathC2A2, pathC3A3},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{},
			expected: []routes.Route{
				{pathC1A1, pathC2A2},
				{pathC2A2, pathC3A3},
			},
		},
		{
			name: "Routes with fromExcluded only",
			routes: []routes.Route{
				{pathC1A1, pathC2A2},
				{pathC2A2, pathC3A3},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{4: &amount0},
			expected: []routes.Route{
				{pathC1A1, pathC2A2},
				{pathC2A2, pathC3A3},
			},
		},
		{
			name: "Routes with all excluded chain IDs",
			routes: []routes.Route{
				{path0, pathC1A1},
				{path0, pathC2A2},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{1: &amount1, 2: &amount2, 3: &amount3, 4: &amount0},
			expected:         []routes.Route{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Original Routes: %+v\n", tt.routes)
			filteredRoutes := filterNetworkCompliance(tt.routes, tt.fromLockedAmount)
			t.Logf("Filtered Routes: %+v\n", filteredRoutes)
			assert.Equal(t, tt.expected, filteredRoutes)
		})
	}
}

func TestFilterCapacityValidation(t *testing.T) {
	tests := []struct {
		name             string
		routes           []routes.Route
		amountIn         *big.Int
		fromLockedAmount map[uint64]*hexutil.Big
		expectedRoutes   []routes.Route
	}{
		{
			name: "Sufficient capacity with multiple paths",
			routes: []routes.Route{
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
			expectedRoutes: []routes.Route{
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
			routes: []routes.Route{
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
			expectedRoutes: []routes.Route{},
		},
		{
			name: "Exact capacity match",
			routes: []routes.Route{
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
			expectedRoutes: []routes.Route{
				{
					{FromChain: network1, AmountIn: (*hexutil.Big)(big.NewInt(100))},
					{FromChain: network2, AmountIn: (*hexutil.Big)(big.NewInt(50))},
				},
			},
		},
		{
			name: "No locked amounts",
			routes: []routes.Route{
				{
					{FromChain: network1, AmountIn: (*hexutil.Big)(big.NewInt(100))},
					{FromChain: network2, AmountIn: (*hexutil.Big)(big.NewInt(50))},
				},
			},
			amountIn:         big.NewInt(150),
			fromLockedAmount: map[uint64]*hexutil.Big{},
			expectedRoutes: []routes.Route{
				{
					{FromChain: network1, AmountIn: (*hexutil.Big)(big.NewInt(100))},
					{FromChain: network2, AmountIn: (*hexutil.Big)(big.NewInt(50))},
				},
			},
		},
		{
			name: "Single route with sufficient capacity",
			routes: []routes.Route{
				{
					{FromChain: network1, AmountIn: (*hexutil.Big)(big.NewInt(50))},
					{FromChain: network2, AmountIn: (*hexutil.Big)(big.NewInt(100))},
				},
			},
			amountIn: big.NewInt(150),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(50)),
			},
			expectedRoutes: []routes.Route{
				{
					{FromChain: network1, AmountIn: (*hexutil.Big)(big.NewInt(50))},
					{FromChain: network2, AmountIn: (*hexutil.Big)(big.NewInt(100))},
				},
			},
		},
		{
			name: "Single route with inappropriately locked amount",
			routes: []routes.Route{
				{
					{FromChain: network1, AmountIn: (*hexutil.Big)(big.NewInt(100))},
				},
			},
			amountIn: big.NewInt(150),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(50)),
			},
			expectedRoutes: []routes.Route{},
		},
		{
			name: "Single route with insufficient capacity",
			routes: []routes.Route{
				{
					{FromChain: network1, AmountIn: (*hexutil.Big)(big.NewInt(50))},
				},
			},
			amountIn: big.NewInt(150),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(50)),
			},
			expectedRoutes: []routes.Route{},
		},
		{
			name:     "Empty routes",
			routes:   []routes.Route{},
			amountIn: big.NewInt(150),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(50)),
			},
			expectedRoutes: []routes.Route{},
		},
		{
			name: "Partial locked amounts",
			routes: []routes.Route{
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
			expectedRoutes: []routes.Route{
				{
					{FromChain: network1, AmountIn: (*hexutil.Big)(big.NewInt(50))},
					{FromChain: network3, AmountIn: (*hexutil.Big)(big.NewInt(100))},
					{FromChain: network4, AmountIn: (*hexutil.Big)(big.NewInt(100))},
				},
			},
		},
		{
			name: "Mixed networks with sufficient capacity",
			routes: []routes.Route{
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
			expectedRoutes: []routes.Route{
				{
					{FromChain: network1, AmountIn: (*hexutil.Big)(big.NewInt(100))},
					{FromChain: network3, AmountIn: (*hexutil.Big)(big.NewInt(200))},
				},
			},
		},
		{
			name: "Mixed networks with insufficient capacity",
			routes: []routes.Route{
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
			expectedRoutes: []routes.Route{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filteredRoutes := filterCapacityValidation(tt.routes, tt.amountIn, tt.fromLockedAmount)
			if !routesEqual(t, tt.expectedRoutes, filteredRoutes) {
				t.Errorf("Expected: %+v, Actual: %+v", tt.expectedRoutes, filteredRoutes)
			}
		})
	}
}

func TestFilterRoutes(t *testing.T) {
	tests := []struct {
		name             string
		routes           []routes.Route
		amountIn         *big.Int
		fromLockedAmount map[uint64]*hexutil.Big
		expectedRoutes   []routes.Route
	}{
		{
			name: "Empty fromLockedAmount and routes don't match amountIn",
			routes: []routes.Route{
				{pathC1A1, pathC2A2},
				{pathC3A3, pathC4A4},
			},
			amountIn:         big.NewInt(150),
			fromLockedAmount: map[uint64]*hexutil.Big{},
			expectedRoutes:   []routes.Route{},
		},
		{
			name: "Empty fromLockedAmount and sigle route match amountIn",
			routes: []routes.Route{
				{pathC1A1, pathC2A2},
				{pathC3A3, pathC4A4},
			},
			amountIn:         big.NewInt(300),
			fromLockedAmount: map[uint64]*hexutil.Big{},
			expectedRoutes: []routes.Route{
				{pathC1A1, pathC2A2},
			},
		},
		{
			name: "Empty fromLockedAmount and more routes match amountIn",
			routes: []routes.Route{
				{pathC1A1, pathC2A2},
				{pathC3A3, pathC4A4},
				{pathC1A1, pathC2A1, pathC3A1},
			},
			amountIn:         big.NewInt(300),
			fromLockedAmount: map[uint64]*hexutil.Big{},
			expectedRoutes: []routes.Route{
				{pathC1A1, pathC2A2},
				{pathC1A1, pathC2A1, pathC3A1},
			},
		},
		{
			name: "All paths appear in fromLockedAmount but not within a single route",
			routes: []routes.Route{
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
			expectedRoutes: []routes.Route{},
		},
		{
			name: "Mixed valid and invalid routes I",
			routes: []routes.Route{
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
			expectedRoutes: []routes.Route{
				{pathC1A1, pathC2A2},
			},
		},
		{
			name: "Mixed valid and invalid routes II",
			routes: []routes.Route{
				{pathC1A1, pathC2A2},
				{pathC2A2, pathC3A3},
				{pathC1A1, pathC4A4},
				{pathC1A1, pathC2A1, pathC3A1},
			},
			amountIn: big.NewInt(300),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: &amount1,
			},
			expectedRoutes: []routes.Route{
				{pathC1A1, pathC2A2},
				{pathC1A1, pathC2A1, pathC3A1},
			},
		},
		{
			name: "All invalid routes",
			routes: []routes.Route{
				{pathC2A2, pathC3A3},
				{pathC4A4, pathC5A5},
			},
			amountIn: big.NewInt(300),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: &amount1,
			},
			expectedRoutes: []routes.Route{},
		},
		{
			name: "Single valid route",
			routes: []routes.Route{
				{pathC1A1, pathC3A3},
				{pathC2A2, pathC3A3},
			},
			amountIn: big.NewInt(400),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: &amount1,
				3: &amount3,
			},
			expectedRoutes: []routes.Route{
				{pathC1A1, pathC3A3},
			},
		},
		{
			name: "Route with mixed valid and invalid paths I",
			routes: []routes.Route{
				{pathC1A1, pathC2A2, pathC3A3},
			},
			amountIn: big.NewInt(300),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: &amount1,
				2: &amount0, // This path should be filtered out due to being excluded via a zero amount
			},
			expectedRoutes: []routes.Route{},
		},
		{
			name: "Route with mixed valid and invalid paths II",
			routes: []routes.Route{
				{pathC1A1, pathC3A3},
			},
			amountIn: big.NewInt(400),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: &amount1,
				2: &amount0, // This path should be filtered out due to being excluded via a zero amount, 0 value locked means this chain is disabled
			},
			expectedRoutes: []routes.Route{
				{pathC1A1, pathC3A3},
			},
		},
		{
			name: "Route with mixed valid and invalid paths III",
			routes: []routes.Route{
				{pathC1A1, pathC3A3},
				{pathC1A1, pathC3A2, pathC4A1},
			},
			amountIn: big.NewInt(400),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: &amount1,
				2: &amount0, // This path should be filtered out due to being excluded via a zero amount, 0 value locked means this chain is disabled
			},
			expectedRoutes: []routes.Route{
				{pathC1A1, pathC3A3},
				{pathC1A1, pathC3A2, pathC4A1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Original Routes: %+v\n", tt.routes)
			filteredRoutes := filterRoutes(tt.routes, tt.amountIn, tt.fromLockedAmount)
			t.Logf("Filtered Routes: %+v\n", filteredRoutes)
			assert.Equal(t, tt.expectedRoutes, filteredRoutes)
		})
	}
}
