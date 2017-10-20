package node

import (
	"sync"

	"github.com/ethereum/go-ethereum/node"
)

type safeNode struct {
	*node.Node
	*sync.RWMutex
}

func newNode(en ...*node.Node) *safeNode {
	n := &safeNode{
		RWMutex: &sync.RWMutex{},
	}

	if len(en) != 0 {
		n.SetNode(en[0])
	}

	return n
}

func (n *safeNode) GetNode() *node.Node {
	n.RLock()
	defer n.RUnlock()
	return n.Node
}

func (n *safeNode) SetNode(node *node.Node) {
	n.Lock()
	n.Node = node
	n.Unlock()
}
