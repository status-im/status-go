package routes

import "time"

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

// GasEstimateFresh tells whether the main or approval tx gas is recent enough to sign.
func (p *Path) GasEstimateFresh(approval bool, now time.Time) bool {
	estimatedAt := p.gasEstimatedAt(approval)
	return !estimatedAt.IsZero() && now.Sub(estimatedAt) <= GasEstimateMaxAge
}
