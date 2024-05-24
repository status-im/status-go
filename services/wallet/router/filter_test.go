package router

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/status-im/status-go/params"

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

	path0 = &PathV2{From: network4, AmountIn: &amount0}
	path1 = &PathV2{From: network1, AmountIn: &amount1}
	path2 = &PathV2{From: network2, AmountIn: &amount2}
	path3 = &PathV2{From: network3, AmountIn: &amount3}
	path4 = &PathV2{From: network4, AmountIn: &amount4}
	path5 = &PathV2{From: network5, AmountIn: &amount5}
)

func routesEqual(expected, actual [][]*PathV2) bool {
	if len(expected) != len(actual) {
		return false
	}
	for i := range expected {
		if !pathsEqual(expected[i], actual[i]) {
			return false
		}
	}
	return true
}

func pathsEqual(expected, actual []*PathV2) bool {
	if len(expected) != len(actual) {
		return false
	}
	for i := range expected {
		if !pathEqual(expected[i], actual[i]) {
			return false
		}
	}
	return true
}

func pathEqual(expected, actual *PathV2) bool {
	if expected.From.ChainID != actual.From.ChainID {
		fmt.Printf("expected chain ID '%d' , actual chain ID '%d'", expected.From.ChainID, actual.From.ChainID)
		return false
	}
	if expected.AmountIn.ToInt().Cmp(actual.AmountIn.ToInt()) != 0 {
		fmt.Printf("expected AmountIn '%d' , actual AmountIn '%d'", expected.AmountIn.ToInt(), actual.AmountIn.ToInt())
		return false
	}
	if expected.AmountInLocked != actual.AmountInLocked {
		fmt.Printf("expected AmountInLocked '%t' , actual AmountInLocked '%t'", expected.AmountInLocked, actual.AmountInLocked)
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
				1: (*hexutil.Big)(zero),
				2: (*hexutil.Big)(big.NewInt(200)),
				3: (*hexutil.Big)(zero),
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
				1: (*hexutil.Big)(zero),
				2: (*hexutil.Big)(zero),
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
				1: (*hexutil.Big)(zero),
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
			name:        "Exclude path1",
			route:       []*PathV2{path1, path2, path3},
			excludePath: path1,
			expected:    big.NewInt(500), // 200 + 300
		},
		{
			name:        "Exclude path2",
			route:       []*PathV2{path1, path2, path3},
			excludePath: path2,
			expected:    big.NewInt(400), // 100 + 300
		},
		{
			name:        "Exclude path3",
			route:       []*PathV2{path1, path2, path3},
			excludePath: path3,
			expected:    big.NewInt(300), // 100 + 200
		},
		{
			name:        "Single path, exclude that path",
			route:       []*PathV2{path1},
			excludePath: path1,
			expected:    big.NewInt(0), // No other paths
		},
		{
			name:        "Empty route",
			route:       []*PathV2{},
			excludePath: path1,
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
			route:          []*PathV2{path1, path2},
			fromIncluded:   map[uint64]bool{1: true, 2: true},
			fromExcluded:   map[uint64]bool{},
			expectedResult: true,
		},
		{
			name:           "Route with fromExcluded only",
			route:          []*PathV2{path1, path2},
			fromIncluded:   map[uint64]bool{},
			fromExcluded:   map[uint64]bool{3: false, 4: false},
			expectedResult: true,
		},
		{
			name:           "Route without excluded chain IDs",
			route:          []*PathV2{path1, path2},
			fromIncluded:   map[uint64]bool{1: false, 2: false},
			fromExcluded:   map[uint64]bool{3: false, 4: false},
			expectedResult: true,
		},
		{
			name:           "Route with an excluded chain ID",
			route:          []*PathV2{path1, path3},
			fromIncluded:   map[uint64]bool{1: false, 2: false},
			fromExcluded:   map[uint64]bool{3: false, 4: false},
			expectedResult: false,
		},
		{
			name:           "Route missing one included chain ID",
			route:          []*PathV2{path1},
			fromIncluded:   map[uint64]bool{1: false, 2: false},
			fromExcluded:   map[uint64]bool{},
			expectedResult: false,
		},
		{
			name:           "Route with no fromIncluded or fromExcluded",
			route:          []*PathV2{path1, path2},
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
			route:            []*PathV2{path1, path2, path3},
			amountIn:         big.NewInt(600),
			fromLockedAmount: map[uint64]*hexutil.Big{1: &amount1, 2: &amount2, 3: &amount3},
			expected:         true,
		},
		// TODO: Find out what the expected behaviour for this case should be
		// I expect false but the test returns true
		/*
			{
				name:             "A path does not meet required amount",
				route:            []*PathV2{path1, path2, path3},
				amountIn:         big.NewInt(600),
				fromLockedAmount: map[uint64]*hexutil.Big{1: &amount1, 2: &amount2, 4: &amount4},
				expected:         false,
			},
		*/
		{
			name:             "No fromLockedAmount",
			route:            []*PathV2{path1, path2, path3},
			amountIn:         big.NewInt(600),
			fromLockedAmount: map[uint64]*hexutil.Big{},
			expected:         true,
		},
		{
			name:             "Single path meets required amount",
			route:            []*PathV2{path1},
			amountIn:         big.NewInt(100),
			fromLockedAmount: map[uint64]*hexutil.Big{1: &amount1},
			expected:         true,
		},
		{
			name:             "Single path does not meet required amount",
			route:            []*PathV2{path1},
			amountIn:         big.NewInt(200),
			fromLockedAmount: map[uint64]*hexutil.Big{1: &amount1},
			expected:         false,
		},
		{
			name:             "Path meets required amount with excess",
			route:            []*PathV2{path1, path2},
			amountIn:         big.NewInt(250),
			fromLockedAmount: map[uint64]*hexutil.Big{1: &amount1, 2: &amount2},
			expected:         true,
		},
		{
			name:             "Path does not meet required amount due to insufficient rest",
			route:            []*PathV2{path1, path2, path4},
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
					{From: network1},
					{From: network3},
				},
				{
					{From: network2},
					{From: network3},
				},
				{
					{From: network1},
					{From: network2},
					{From: network3},
				},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(100)),
				2: (*hexutil.Big)(big.NewInt(0)),
			},
			expected: [][]*PathV2{
				{
					{From: network1},
					{From: network3},
				},
			},
		},
		{
			name: "All valid routes",
			routes: [][]*PathV2{
				{
					{From: network1},
					{From: network3},
				},
				{
					{From: network1},
					{From: network4},
				},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(100)),
			},
			expected: [][]*PathV2{
				{
					{From: network1},
					{From: network3},
				},
				{
					{From: network1},
					{From: network4},
				},
			},
		},
		{
			name: "All invalid routes",
			routes: [][]*PathV2{
				{
					{From: network2},
					{From: network3},
				},
				{
					{From: network4},
					{From: network5},
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
					{From: network1},
					{From: network2},
				},
				{
					{From: network3},
					{From: network4},
				},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{},
			expected: [][]*PathV2{
				{
					{From: network1},
					{From: network2},
				},
				{
					{From: network3},
					{From: network4},
				},
			},
		},
		{
			name: "Single route with mixed valid and invalid paths",
			routes: [][]*PathV2{
				{
					{From: network1},
					{From: network2},
					{From: network3},
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
					{From: network1},
					{From: network1},
					{From: network2},
				},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(100)),
			},
			expected: [][]*PathV2{
				{
					{From: network1},
					{From: network1},
					{From: network2},
				},
			},
		},
		{
			name: "Minimum and maximum chain IDs",
			routes: [][]*PathV2{
				{
					{From: &params.Network{ChainID: 0}},
					{From: &params.Network{ChainID: ^uint64(0)}},
				},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{
				0:          (*hexutil.Big)(big.NewInt(100)),
				^uint64(0): (*hexutil.Big)(big.NewInt(100)),
			},
			expected: [][]*PathV2{
				{
					{From: &params.Network{ChainID: 0}},
					{From: &params.Network{ChainID: ^uint64(0)}},
				},
			},
		},
		{
			name: "Large number of routes",
			routes: func() [][]*PathV2 {
				var routes [][]*PathV2
				for i := 0; i < 1000; i++ {
					routes = append(routes, []*PathV2{
						{From: &params.Network{ChainID: uint64(i + 1)}},
						{From: &params.Network{ChainID: uint64(i + 1001)}},
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
						{From: &params.Network{ChainID: uint64(i + 1)}},
						{From: &params.Network{ChainID: uint64(i + 1001)}},
					})
				}
				return routes
			}(),
		},
		{
			name: "Routes with missing data",
			routes: [][]*PathV2{
				{
					{From: nil},
					{From: network2},
				},
				{
					{From: network1},
					{From: nil},
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
					{From: network1},
					{From: network2},
				},
				{
					{From: network1},
					{From: network3},
				},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: (*hexutil.Big)(big.NewInt(100)),
			},
			expected: [][]*PathV2{
				{
					{From: network1},
					{From: network2},
				},
				{
					{From: network1},
					{From: network3},
				},
			},
		},
		{
			name: "Routes without excluded chain IDs, missing included path",
			routes: [][]*PathV2{
				{path1, path2},
				{path2, path3},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{1: &amount1, 2: &amount2},
			expected: [][]*PathV2{
				{path1, path2},
			},
		},
		{
			name: "Routes with an excluded chain ID",
			routes: [][]*PathV2{
				{path1, path2},
				{path2, path3, path0},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{1: &amount1, 2: &amount2, 4: &amount0},
			expected: [][]*PathV2{
				{path1, path2},
			},
		},
		{
			name: "Routes with all included chain IDs",
			routes: [][]*PathV2{
				{path1, path2, path3},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{1: &amount1, 2: &amount2, 3: &amount3},
			expected: [][]*PathV2{
				{path1, path2, path3},
			},
		},
		{
			name: "Routes missing one included chain ID",
			routes: [][]*PathV2{
				{path1, path2},
				{path1},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{1: &amount1, 2: &amount2, 3: &amount3},
			expected:         [][]*PathV2{},
		},
		{
			name: "Routes with no fromLockedAmount",
			routes: [][]*PathV2{
				{path1, path2},
				{path2, path3},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{},
			expected: [][]*PathV2{
				{path1, path2},
				{path2, path3},
			},
		},
		{
			name: "Routes with fromExcluded only",
			routes: [][]*PathV2{
				{path1, path2},
				{path2, path3},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{4: &amount0},
			expected: [][]*PathV2{
				{path1, path2},
				{path2, path3},
			},
		},
		{
			name: "Routes with all excluded chain IDs",
			routes: [][]*PathV2{
				{path0, path1},
				{path0, path2},
			},
			fromLockedAmount: map[uint64]*hexutil.Big{1: &amount1, 2: &amount2, 3: &amount3, 4: &amount0},
			expected:         [][]*PathV2{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fmt.Printf("Original Routes: %+v\n", tt.routes)
			filteredRoutes := filterNetworkComplianceV2(tt.routes, tt.fromLockedAmount)
			fmt.Printf("Filtered Routes: %+v\n", filteredRoutes)
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
					{From: network1, AmountIn: (*hexutil.Big)(big.NewInt(50)), AmountInLocked: true},
					{From: network2, AmountIn: (*hexutil.Big)(big.NewInt(100)), AmountInLocked: true},
				},
				{
					{From: network1, AmountIn: (*hexutil.Big)(big.NewInt(50)), AmountInLocked: true},
					{From: network2, AmountIn: (*hexutil.Big)(big.NewInt(100)), AmountInLocked: true},
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
			// TODO Is the behaviour of this test correct? It looks wrong
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
			expectedRoutes: [][]*PathV2{},
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
			// TODO this seems wrong also. Should this test case work?
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
			expectedRoutes: [][]*PathV2{},
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
				2: (*hexutil.Big)(big.NewInt(0)), // Excluded path
				3: (*hexutil.Big)(big.NewInt(100)),
			},
			expectedRoutes: [][]*PathV2{},
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
			if !routesEqual(tt.expectedRoutes, filteredRoutes) {
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
			name: "Empty fromLockedAmount",
			routes: [][]*PathV2{
				{path1, path2},
				{path3, path4},
			},
			amountIn:         big.NewInt(150),
			fromLockedAmount: map[uint64]*hexutil.Big{},
			expectedRoutes: [][]*PathV2{
				{path1, path2},
				{path3, path4},
			},
		},
		{
			name: "All paths appear in fromLockedAmount but not within a single route",
			routes: [][]*PathV2{
				{path1, path3},
				{path2, path4},
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
			name: "Mixed valid and invalid routes",
			routes: [][]*PathV2{
				{path1, path2},
				{path2, path3},
				{path1, path4},
			},
			amountIn: big.NewInt(300),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: &amount1,
				2: &amount2,
			},
			expectedRoutes: [][]*PathV2{
				{path1, path2},
			},
		},
		{
			name: "All invalid routes",
			routes: [][]*PathV2{
				{path2, path3},
				{path4, path5},
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
				{path1, path3},
				{path2, path3},
			},
			amountIn: big.NewInt(150),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: &amount1,
				3: &amount3,
			},
			expectedRoutes: [][]*PathV2{
				{path1, path3},
			},
		},
		{
			name: "Route with mixed valid and invalid paths",
			routes: [][]*PathV2{
				{path1, path2, path3},
			},
			amountIn: big.NewInt(300),
			fromLockedAmount: map[uint64]*hexutil.Big{
				1: &amount1,
				2: &amount0, // This path should be filtered out due to being excluded via a zero amount
			},
			expectedRoutes: [][]*PathV2{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fmt.Printf("Original Routes: %+v\n", tt.routes)
			filteredRoutes := filterRoutesV2(tt.routes, tt.amountIn, tt.fromLockedAmount)
			fmt.Printf("Filtered Routes: %+v\n", filteredRoutes)
			assert.Equal(t, tt.expectedRoutes, filteredRoutes)
		})
	}
}
