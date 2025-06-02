package router

import (
	"math/big"

	"github.com/status-im/status-go/services/wallet/router/routes"

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

func filterRoutes(routes []routes.Route, amountIn *big.Int) []routes.Route {
	for i := len(routes) - 1; i >= 0; i-- {
		routeAmount := big.NewInt(0)
		for _, p := range routes[i] {
			routeAmount.Add(routeAmount, p.AmountIn.ToInt())
		}

		if routeAmount.Cmp(amountIn) == 0 {
			continue
		}

		routes = append(routes[:i], routes[i+1:]...)
	}

	return routes
}

// isValidForNetworkCompliance checks if a route complies with network inclusion/exclusion criteria.
func isValidForNetworkCompliance(route routes.Route, fromIncluded, fromExcluded map[uint64]bool) bool {
	logger.Debug("Initial inclusion/exclusion maps",
		zap.Any("fromIncluded", fromIncluded),
		zap.Any("fromExcluded", fromExcluded),
	)

	if fromIncluded == nil || fromExcluded == nil {
		return false
	}

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

// calculateRestAmountIn calculates the remaining amount in for the route excluding the specified path
func calculateRestAmountIn(route routes.Route, excludePath *routes.Path) *big.Int {
	restAmountIn := big.NewInt(0)
	for _, path := range route {
		if path != excludePath {
			restAmountIn.Add(restAmountIn, path.AmountIn.ToInt())
		}
	}
	return restAmountIn
}
