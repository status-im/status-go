package pathprocessor

import "github.com/ethereum/go-ethereum/common/hexutil"

// gasForQuotedTx returns the larger of our buffered estimate and the provider's quoted gas; the
// quote may miss gas our node accounts for, and an out-of-gas tx still burns its fee.
func gasForQuotedTx(routerGas *hexutil.Uint64, quotedGas uint64) uint64 {
	if routerGas != nil && uint64(*routerGas) > quotedGas {
		return uint64(*routerGas)
	}
	return quotedGas
}
