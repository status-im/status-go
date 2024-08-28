package routes

import (
	"math/big"
)

type Graph []*Node

type Node struct {
	Path     *Path
	Children Graph
}

func newNode(path *Path) *Node {
	return &Node{Path: path, Children: make(Graph, 0)}
}

func BuildGraph(AmountIn *big.Int, route Route, level int, sourceChainIDs []uint64) Graph {
	graph := make(Graph, 0)
	for _, path := range route {
		found := false
		for _, chainID := range sourceChainIDs {
			if chainID == path.FromChain.ChainID {
				found = true
				break
			}
		}
		if found {
			continue
		}
		node := newNode(path)

		newRoute := make(Route, 0)
		for _, p := range route {
			if path.Equal(p) {
				continue
			}
			newRoute = append(newRoute, p)
		}

		newAmountIn := new(big.Int).Sub(AmountIn, path.AmountIn.ToInt())
		if newAmountIn.Sign() > 0 {
			newSourceChainIDs := make([]uint64, len(sourceChainIDs))
			copy(newSourceChainIDs, sourceChainIDs)
			newSourceChainIDs = append(newSourceChainIDs, path.FromChain.ChainID)
			node.Children = BuildGraph(newAmountIn, newRoute, level+1, newSourceChainIDs)

			if len(node.Children) == 0 {
				continue
			}
		}

		graph = append(graph, node)
	}

	return graph
}

func (n Node) BuildAllRoutes() []Route {
	res := make([]Route, 0)

	if len(n.Children) == 0 && n.Path != nil {
		res = append(res, Route{n.Path})
	}

	for _, node := range n.Children {
		for _, route := range node.BuildAllRoutes() {
			extendedRoute := route
			if n.Path != nil {
				extendedRoute = append(Route{n.Path}, route...)
			}
			res = append(res, extendedRoute)
		}
	}

	return res
}
