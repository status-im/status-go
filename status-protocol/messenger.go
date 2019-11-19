package status_protocol

import ethnode "github.com/status-im/status-go/status-eth-node"

type Messenger struct {
	node *ethnode.Node
}

func NewMessenger(n *ethnode.Node) *Messenger {
	return &Messenger{
		node: n,
	}
}
