package router

import (
	"github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/router/routes"
)

func removeBestRouteFromAllRouters(allRoutes []routes.Route, best routes.Route) []routes.Route {
	for i := len(allRoutes) - 1; i >= 0; i-- {
		route := allRoutes[i]
		routeFound := true
		for _, p := range route {
			found := false
			for _, b := range best {
				if p.ProcessorName == b.ProcessorName &&
					(p.FromChain == nil && b.FromChain == nil || p.FromChain.ChainID == b.FromChain.ChainID) &&
					(p.ToChain == nil && b.ToChain == nil || p.ToChain.ChainID == b.ToChain.ChainID) &&
					(p.FromToken == nil && b.FromToken == nil || p.FromToken.Symbol == b.FromToken.Symbol) {
					found = true
					break
				}
			}
			if !found {
				routeFound = false
				break
			}
		}
		if routeFound {
			return append(allRoutes[:i], allRoutes[i+1:]...)
		}
	}

	return nil
}

func getChainPriority(chainID uint64) int {
	switch chainID {
	case common.EthereumMainnet, common.EthereumSepolia:
		return 1
	case common.OptimismMainnet, common.OptimismSepolia:
		return 2
	case common.ArbitrumMainnet, common.ArbitrumSepolia:
		return 3
	default:
		return 0
	}
}

func getRoutePriority(route routes.Route) int {
	priority := 0
	for _, path := range route {
		priority += getChainPriority(path.FromChain.ChainID)
	}
	return priority
}
