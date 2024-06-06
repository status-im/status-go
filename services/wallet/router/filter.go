package router

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/services/wallet/router/bridge"

	"go.uber.org/zap"
)

var logger *zap.Logger

func init() {
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		panic(err)
	}
}

func filterRoutesV2(routes [][]*PathV2, amountIn *big.Int, fromLockedAmount map[uint64]*hexutil.Big) [][]*PathV2 {
	if len(fromLockedAmount) == 0 {
		return routes
	}

	routesAfterNetworkCompliance := filterNetworkComplianceV2(routes, fromLockedAmount)
	return filterCapacityValidationV2(routesAfterNetworkCompliance, amountIn, fromLockedAmount)
}

// filterNetworkComplianceV2 performs the first level of filtering based on network inclusion/exclusion criteria.
func filterNetworkComplianceV2(routes [][]*PathV2, fromLockedAmount map[uint64]*hexutil.Big) [][]*PathV2 {
	filteredRoutes := make([][]*PathV2, 0)
	if routes == nil || fromLockedAmount == nil {
		return filteredRoutes
	}

	fromIncluded, fromExcluded := setupRouteValidationMapsV2(fromLockedAmount)

	for _, route := range routes {
		if route == nil {
			continue
		}

		// Create fresh copies of the maps for each route check, because they are manipulated
		if isValidForNetworkComplianceV2(route, copyMap(fromIncluded), copyMap(fromExcluded)) {
			filteredRoutes = append(filteredRoutes, route)
		}
	}
	return filteredRoutes
}

// isValidForNetworkComplianceV2 checks if a route complies with network inclusion/exclusion criteria.
func isValidForNetworkComplianceV2(route []*PathV2, fromIncluded, fromExcluded map[uint64]bool) bool {
	logger.Debug("Initial inclusion/exclusion maps",
		zap.Any("fromIncluded", fromIncluded),
		zap.Any("fromExcluded", fromExcluded),
	)

	for _, path := range route {
		if path == nil || path.FromChain == nil {
			logger.Debug("Invalid path", zap.Any("path", path))
			return false
		}
		if _, ok := fromExcluded[path.FromChain.ChainID]; ok {
			logger.Debug("Excluded chain ID", zap.Uint64("chainID", path.FromChain.ChainID))
			return false
		}
		if _, ok := fromIncluded[path.FromChain.ChainID]; ok {
			fromIncluded[path.FromChain.ChainID] = true
		}
	}

	logger.Debug("fromIncluded after loop", zap.Any("fromIncluded", fromIncluded))

	for chainID, included := range fromIncluded {
		if !included {
			logger.Debug("Missing included chain ID", zap.Uint64("chainID", chainID))
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
		if amount.ToInt().Cmp(bridge.ZeroBigIntValue) <= 0 {
			fromExcluded[chainID] = false
		} else {
			fromIncluded[chainID] = false
		}
	}
	return fromIncluded, fromExcluded
}

// filterCapacityValidationV2 performs the second level of filtering based on amount and capacity validation.
func filterCapacityValidationV2(routes [][]*PathV2, amountIn *big.Int, fromLockedAmount map[uint64]*hexutil.Big) [][]*PathV2 {
	filteredRoutes := make([][]*PathV2, 0)

	for _, route := range routes {
		if hasSufficientCapacityV2(route, amountIn, fromLockedAmount) {
			filteredRoutes = append(filteredRoutes, route)
		}
	}
	return filteredRoutes
}

// hasSufficientCapacityV2 checks if a route has sufficient capacity to handle the required amount.
func hasSufficientCapacityV2(route []*PathV2, amountIn *big.Int, fromLockedAmount map[uint64]*hexutil.Big) bool {
	for _, path := range route {
		if amount, ok := fromLockedAmount[path.FromChain.ChainID]; ok {
			requiredAmountIn := new(big.Int).Sub(amountIn, amount.ToInt())
			restAmountIn := calculateRestAmountInV2(route, path)

			logger.Debug("Checking path", zap.Any("path", path))
			logger.Debug("Required amount in", zap.String("requiredAmountIn", requiredAmountIn.String()))
			logger.Debug("Rest amount in", zap.String("restAmountIn", restAmountIn.String()))

			if restAmountIn.Cmp(requiredAmountIn) >= 0 {
				path.AmountIn = amount
				path.AmountInLocked = true
				logger.Debug("Path has sufficient capacity", zap.Any("path", path))
			} else {
				logger.Debug("Path does not have sufficient capacity", zap.Any("path", path))
				return false
			}
		}
	}
	return true
}

// calculateRestAmountIn calculates the remaining amount in for the route excluding the specified path
func calculateRestAmountInV2(route []*PathV2, excludePath *PathV2) *big.Int {
	restAmountIn := big.NewInt(0)
	for _, path := range route {
		if path != excludePath {
			restAmountIn.Add(restAmountIn, path.AmountIn.ToInt())
		}
	}
	return restAmountIn
}

// copyMap creates a copy of the given map[uint64]bool
func copyMap(original map[uint64]bool) map[uint64]bool {
	c := make(map[uint64]bool)
	for k, v := range original {
		c[k] = v
	}
	return c
}
