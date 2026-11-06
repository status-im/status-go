package pathprocessor

import (
	"github.com/ethereum/go-ethereum/common/hexutil"

	pathprocessorcommon "github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor/common"
)

// gasForQuotedTx returns the larger of our buffered estimate and the provider's quoted gas; the
// quote may miss gas our node accounts for, and an out-of-gas tx still burns its fee.
func gasForQuotedTx(routerGas *hexutil.Uint64, quotedGas uint64) uint64 {
	if routerGas != nil && uint64(*routerGas) > quotedGas {
		return uint64(*routerGas)
	}
	return quotedGas
}

// GasBufferFactorProvider is implemented by processors that need a non-default buffer, e.g. bridges.
type GasBufferFactorProvider interface {
	GasBufferFactor(params ProcessorInputParams) float64
}

// GasBufferFactor is the buffer the router applies to the processor's raw gas estimate.
func GasBufferFactor(processor PathProcessor, params ProcessorInputParams) float64 {
	if p, ok := processor.(GasBufferFactorProvider); ok {
		return p.GasBufferFactor(params)
	}
	return pathprocessorcommon.IncreaseEstimatedGasFactor
}

// BufferGasEstimate applies factor to a raw estimate. Truncated, so raw == ceil(buffered / factor).
func BufferGasEstimate(estimation uint64, factor float64) uint64 {
	return uint64(float64(estimation) * factor)
}

// BufferedGasEstimate returns the gas limit to sign for processor's raw estimate.
func BufferedGasEstimate(processor PathProcessor, params ProcessorInputParams, estimation uint64) uint64 {
	return BufferGasEstimate(estimation, GasBufferFactor(processor, params))
}
