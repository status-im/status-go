package gas

import "github.com/ethereum/go-ethereum"

// calculateNetworkCongestionFromHistory calculates network congestion from fee history
func calculateNetworkCongestionFromHistory(feeHistory *ethereum.FeeHistory, nBlocks int) float64 {
	startIdx := max(len(feeHistory.GasUsedRatio)-nBlocks, 0)
	gasUsedRatio := feeHistory.GasUsedRatio[startIdx:]

	return calculateNetworkCongestion(gasUsedRatio)
}

// calculateNetworkCongestion calculates network congestion based on the gas used ratio
// (gasUsed / gasLimit) for the last blocks
// This algorithm doesn't work well for rollups, where congestion should consider the state of the L1
func calculateNetworkCongestion(gasUsedRatio []float64) float64 {
	congestion := 0.0
	for _, ratio := range gasUsedRatio[:] {
		blockCongestion := (ratio - 0.5)
		if blockCongestion > 0 {
			congestion += blockCongestion
		}
	}
	congestion = congestion / float64(len(gasUsedRatio))

	// Congestion measures how much the network exceeds the 50% EIP-1559 utilization target,
	// normalized to 0-1 scale.
	congestion = congestion * 2
	if congestion > 1.0 {
		congestion = 1.0
	}

	return congestion
}
