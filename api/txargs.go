package api

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/sign"
)

// prepareTxArgs given gas and gasPrice will prepare a valid sign.TxArgs.
func prepareTxArgs(gas, gasPrice int64) (args sign.TxArgs) {
	if gas > 0 {
		g := hexutil.Uint64(gas)
		args.Gas = &g
	}
	if gasPrice > 0 {
		gp := (*hexutil.Big)(big.NewInt(gasPrice))
		args.GasPrice = gp
	}
	return
}
