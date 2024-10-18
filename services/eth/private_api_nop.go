//go:build !enable_private_api

package eth

import (
	geth_rpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/rpc"
)

func privateAPIs(*rpc.Client) (apis []geth_rpc.API) {
	return nil
}
