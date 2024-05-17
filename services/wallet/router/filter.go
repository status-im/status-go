package router

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

func filterRoutesV2(routes [][]*PathV2, amountIn *big.Int, fromLockedAmount map[uint64]*hexutil.Big) [][]*PathV2 {
	if len(fromLockedAmount) == 0 {
		return routes
	}

	routesAfterNetworkCompliance := filterNetworkComplianceV2(routes, fromLockedAmount)
	return filterCapacityValidationV2(routesAfterNetworkCompliance, amountIn, fromLockedAmount)
}

// filterNetworkComplianceV2 performs the first level of filtering based on network inclusion/exclusion criteria.
func filterNetworkComplianceV2(routes [][]*PathV2, fromLockedAmount map[uint64]*hexutil.Big) [][]*PathV2 {
	var filteredRoutes [][]*PathV2
	fromIncluded, fromExcluded := setupRouteValidationMapsV2(fromLockedAmount)

	for _, route := range routes {
		if isValidForNetworkComplianceV2(route, fromIncluded, fromExcluded) {
			filteredRoutes = append(filteredRoutes, route)
		}
	}
	return filteredRoutes
}

// isValidForNetworkComplianceV2 checks if a route complies with network inclusion/exclusion criteria.
func isValidForNetworkComplianceV2(route []*PathV2, fromIncluded, fromExcluded map[uint64]bool) bool {
	for _, path := range route {
		if fromExcluded[path.From.ChainID] {
			return false
		}
		if _, ok := fromIncluded[path.From.ChainID]; ok {
			fromIncluded[path.From.ChainID] = true
		}
	}

	for _, included := range fromIncluded {
		if !included {
			return false
		}
	}

	return true
}

// setupRouteValidationMapsV2 initializes maps for network inclusion and exclusion based on locked amounts.
func setupRouteValidationMapsV2(fromLockedAmount map[uint64]*hexutil.Big) (map[uint64]bool, map[uint64]bool) {
	fromIncluded := make(map[uint64]bool)
	fromExcluded := make(map[uint64]bool)

	for chainID, amount := range fromLockedAmount {
		if amount.ToInt().Cmp(zero) == 0 {
			fromExcluded[chainID] = true
		} else {
			fromIncluded[chainID] = false
		}
	}
	return fromIncluded, fromExcluded
}

// filterCapacityValidationV2 performs the second level of filtering based on amount and capacity validation.
func filterCapacityValidationV2(routes [][]*PathV2, amountIn *big.Int, fromLockedAmount map[uint64]*hexutil.Big) [][]*PathV2 {
	var filteredRoutes [][]*PathV2

	for _, route := range routes {
		if hasSufficientCapacityV2(route, amountIn, fromLockedAmount) {
			filteredRoutes = append(filteredRoutes, route)
		}
	}
	return filteredRoutes
}

// hasSufficientCapacityV2 checks if a route has sufficient capacity to handle the required amount.
func hasSufficientCapacityV2(route []*PathV2, amountIn *big.Int, fromLockedAmount map[uint64]*hexutil.Big) bool {
	totalRestAmount := calculateTotalRestAmountV2(route)

	for _, path := range route {
		if amount, ok := fromLockedAmount[path.From.ChainID]; ok {
			requiredAmountIn := new(big.Int).Sub(amountIn, amount.ToInt())
			if totalRestAmount.Cmp(requiredAmountIn) < 0 {
				return false
			}
			path.AmountIn = amount
			path.AmountInLocked = true
			totalRestAmount.Sub(totalRestAmount, amount.ToInt())
		}
	}
	return true
}

// calculateTotalRestAmountV2 calculates the total maximum amount that can be used from all paths in the route.
func calculateTotalRestAmountV2(route []*PathV2) *big.Int {
	total := big.NewInt(0)
	for _, path := range route {
		total.Add(total, path.AmountIn.ToInt())
	}
	return total
}
