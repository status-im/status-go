package swap

type SwapParameters struct {
	mode                int
	paymentThreshold    int
	disconnectThreshold int
}

type SwapOption func(*SwapParameters)

func WithMode(mode int) SwapOption {
	return func(params *SwapParameters) {
		params.mode = mode
	}
}

func WithThreshold(payment, disconnect int) SwapOption {
	return func(params *SwapParameters) {
		params.disconnectThreshold = disconnect
		params.paymentThreshold = payment
	}
}

func DefaultOptions() []SwapOption {
	return []SwapOption{
		WithMode(SoftMode),
		WithThreshold(100, -100),
	}
}
