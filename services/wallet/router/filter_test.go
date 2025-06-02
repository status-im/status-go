package router

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/wallet/router/routes"

	"github.com/stretchr/testify/assert"
)

var (
	network1 = &params.Network{ChainID: 1}
	network2 = &params.Network{ChainID: 2}
	network3 = &params.Network{ChainID: 3}
	network4 = &params.Network{ChainID: 4}
	network5 = &params.Network{ChainID: 5}

	amount1 = hexutil.Big(*big.NewInt(100))
	amount2 = hexutil.Big(*big.NewInt(200))
	amount3 = hexutil.Big(*big.NewInt(300))
	amount4 = hexutil.Big(*big.NewInt(400))
	amount5 = hexutil.Big(*big.NewInt(500))

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

func TestFilterRoutes(t *testing.T) {
	tests := []struct {
		name           string
		routes         []routes.Route
		amountIn       *big.Int
		expectedRoutes []routes.Route
	}{
		{
			name: "Routes don't match amountIn",
			routes: []routes.Route{
				{pathC1A1, pathC2A2},
				{pathC3A3, pathC4A4},
			},
			amountIn:       big.NewInt(150),
			expectedRoutes: []routes.Route{},
		},
		{
			name: "Sigle route match amountIn",
			routes: []routes.Route{
				{pathC1A1, pathC2A2},
				{pathC3A3, pathC4A4},
			},
			amountIn: big.NewInt(300),
			expectedRoutes: []routes.Route{
				{pathC1A1, pathC2A2},
			},
		},
		{
			name: "More routes match amountIn",
			routes: []routes.Route{
				{pathC1A1, pathC2A2},
				{pathC3A3, pathC4A4},
				{pathC1A1, pathC2A1, pathC3A1},
			},
			amountIn: big.NewInt(300),
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
			amountIn:       big.NewInt(300),
			expectedRoutes: []routes.Route{},
		},
		{
			name: "Route with mixed valid and invalid paths III",
			routes: []routes.Route{
				{pathC1A1, pathC3A3},
				{pathC1A1, pathC3A2, pathC4A1},
			},
			amountIn: big.NewInt(400),
			expectedRoutes: []routes.Route{
				{pathC1A1, pathC3A3},
				{pathC1A1, pathC3A2, pathC4A1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Original Routes: %+v\n", tt.routes)
			filteredRoutes := filterRoutes(tt.routes, tt.amountIn)
			t.Logf("Filtered Routes: %+v\n", filteredRoutes)
			assert.Equal(t, tt.expectedRoutes, filteredRoutes)
		})
	}
}
