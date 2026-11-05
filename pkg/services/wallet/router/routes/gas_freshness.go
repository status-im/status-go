package routes

import (
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

const (
	// GasReestimateInterval is how often the active route's gas is re-estimated.
	GasReestimateInterval = time.Minute
	// GasEstimateMaxAge is the oldest gas estimate that may be signed.
	GasEstimateMaxAge = GasReestimateInterval + 30*time.Second
)

func (p *Path) gasEstimatedAt(approval bool) time.Time {
	if approval {
		return p.ApprovalGasEstimatedAt
	}
	return p.TxGasEstimatedAt
}

// GasEstimateDue tells whether the main or approval tx gas should be re-estimated.
func (p *Path) GasEstimateDue(approval bool, now time.Time) bool {
	estimatedAt := p.gasEstimatedAt(approval)
	return estimatedAt.IsZero() || now.Sub(estimatedAt) >= GasReestimateInterval
}

// GasEstimateFresh tells whether the main or approval tx gas is recent enough to sign, and was
// estimated for the value a plain native transfer signs.
func (p *Path) GasEstimateFresh(approval bool, now time.Time) bool {
	estimatedAt := p.gasEstimatedAt(approval)
	if estimatedAt.IsZero() || now.Sub(estimatedAt) > GasEstimateMaxAge {
		return false
	}
	return approval || len(p.TxPackedData) > 0 || sameAmount(p.TxGasEstimatedValue, p.AmountIn)
}

func sameAmount(a, b *hexutil.Big) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.ToInt().Cmp(b.ToInt()) == 0
}
