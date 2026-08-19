package node

import (
	"github.com/ethereum/go-ethereum/rpc"
)

// StatusService is implemented by every service registered on the StatusNode.
type StatusService interface {
	Start() error
	Stop() error
	APIs() []rpc.API
}
