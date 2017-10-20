package geth

import (
	"github.com/status-im/status-go/geth/params"
)

type NodeConstructor interface {
	Make() (Node, error)
	Config() *params.NodeConfig
	SetConfig(config *params.NodeConfig)
}
