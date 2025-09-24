//go:build !rpc_panic

package rpc

import (
	"fmt"
	"runtime"
	
	"github.com/ethereum/go-ethereum/log"
)

func handlePanic(method string, errRes *error) {
	err := recover()
	if err == nil {
		return
	}

	const size = 64 << 10
	buf := make([]byte, size)
	buf = buf[:runtime.Stack(buf, false)]
	log.Error("RPC method " + method + " crashed: " + fmt.Sprintf("%v\n%s", err, buf))
	*errRes = &internalServerError{errcodePanic, "method handler crashed"}
}
