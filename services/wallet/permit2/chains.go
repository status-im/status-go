package permit2

// enabledChains lists the chains where a swap may be routed through LI.FI's Permit2Proxy,
// collapsing the ERC-20 approval and the swap into a single transaction.
// Per-chain rather than one switch: each chain has its own Permit2Proxy deployment and has
// to be validated separately. Abstract and zkSync are left out on purpose, their gas
// accounting doesn't match the buffered estimate this path relies on.
var enabledChains = map[uint64]struct{}{
	1:     {}, // Ethereum
	10:    {}, // Optimism
	8453:  {}, // Base
	42161: {}, // Arbitrum
}

// EnabledForChain reports whether the permit swap path is enabled for the chain. Callers
// fall back to the regular approve-then-swap flow when it returns false.
func EnabledForChain(chainID uint64) bool {
	_, ok := enabledChains[chainID]
	return ok
}
