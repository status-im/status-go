package router

import (
	"math"
	"math/big"

	"github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/router/pathprocessor"
)

type Graph []*Node

type Node struct {
	Path     *Path
	Children Graph
}

func newNode(path *Path) *Node {
	return &Node{Path: path, Children: make(Graph, 0)}
}

func buildGraph(AmountIn *big.Int, routes []*Path, level int, sourceChainIDs []uint64) Graph {
	graph := make(Graph, 0)
	for _, route := range routes {
		found := false
		for _, chainID := range sourceChainIDs {
			if chainID == route.FromChain.ChainID {
				found = true
				break
			}
		}
		if found {
			continue
		}
		node := newNode(route)

		newRoutes := make([]*Path, 0)
		for _, r := range routes {
			if route.Equal(r) {
				continue
			}
			newRoutes = append(newRoutes, r)
		}

		newAmountIn := new(big.Int).Sub(AmountIn, route.AmountIn.ToInt())
		if newAmountIn.Sign() > 0 {
			newSourceChainIDs := make([]uint64, len(sourceChainIDs))
			copy(newSourceChainIDs, sourceChainIDs)
			newSourceChainIDs = append(newSourceChainIDs, route.FromChain.ChainID)
			node.Children = buildGraph(newAmountIn, newRoutes, level+1, newSourceChainIDs)

			if len(node.Children) == 0 {
				continue
			}
		}

		graph = append(graph, node)
	}

	return graph
}

func (n Node) buildAllRoutes() [][]*Path {
	res := make([][]*Path, 0)

	if len(n.Children) == 0 && n.Path != nil {
		res = append(res, []*Path{n.Path})
	}

	for _, node := range n.Children {
		for _, route := range node.buildAllRoutes() {
			extendedRoute := route
			if n.Path != nil {
				extendedRoute = append([]*Path{n.Path}, route...)
			}
			res = append(res, extendedRoute)
		}
	}

	return res
}

func findBest(routes [][]*Path, tokenPrice float64, nativeTokenPrice float64) []*Path {
	var best []*Path
	bestCost := big.NewFloat(math.Inf(1))
	for _, route := range routes {
		currentCost := big.NewFloat(0)
		for _, path := range route {
			tokenDenominator := big.NewFloat(math.Pow(10, float64(path.FromToken.Decimals)))

			// calculate the cost of the path
			nativeTokenPrice := new(big.Float).SetFloat64(nativeTokenPrice)

			// tx fee
			txFeeInEth := common.GweiToEth(common.WeiToGwei(path.TxFee.ToInt()))
			pathCost := new(big.Float).Mul(txFeeInEth, nativeTokenPrice)

			if path.TxL1Fee.ToInt().Cmp(pathprocessor.ZeroBigIntValue) > 0 {
				txL1FeeInEth := common.GweiToEth(common.WeiToGwei(path.TxL1Fee.ToInt()))
				pathCost.Add(pathCost, new(big.Float).Mul(txL1FeeInEth, nativeTokenPrice))
			}

			if path.TxBonderFees != nil && path.TxBonderFees.ToInt().Cmp(pathprocessor.ZeroBigIntValue) > 0 {
				pathCost.Add(pathCost, new(big.Float).Mul(
					new(big.Float).Quo(new(big.Float).SetInt(path.TxBonderFees.ToInt()), tokenDenominator),
					new(big.Float).SetFloat64(tokenPrice)))

			}

			if path.TxTokenFees != nil && path.TxTokenFees.ToInt().Cmp(pathprocessor.ZeroBigIntValue) > 0 && path.FromToken != nil {
				pathCost.Add(pathCost, new(big.Float).Mul(
					new(big.Float).Quo(new(big.Float).SetInt(path.TxTokenFees.ToInt()), tokenDenominator),
					new(big.Float).SetFloat64(tokenPrice)))
			}

			if path.ApprovalRequired {
				// tx approval fee
				approvalFeeInEth := common.GweiToEth(common.WeiToGwei(path.ApprovalFee.ToInt()))
				pathCost.Add(pathCost, new(big.Float).Mul(approvalFeeInEth, nativeTokenPrice))

				if path.ApprovalL1Fee.ToInt().Cmp(pathprocessor.ZeroBigIntValue) > 0 {
					approvalL1FeeInEth := common.GweiToEth(common.WeiToGwei(path.ApprovalL1Fee.ToInt()))
					pathCost.Add(pathCost, new(big.Float).Mul(approvalL1FeeInEth, nativeTokenPrice))
				}
			}

			currentCost = new(big.Float).Add(currentCost, pathCost)
		}

		if currentCost.Cmp(bestCost) == -1 {
			best = route
			bestCost = currentCost
		}
	}

	return best
}
