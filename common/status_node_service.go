package common

import (
	"github.com/ethereum/go-ethereum/rpc"
)

type StatusService interface {
	Start() error
	Stop() error
	APIs() []rpc.API
}
