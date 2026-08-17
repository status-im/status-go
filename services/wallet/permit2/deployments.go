package permit2

import (
	"github.com/ethereum/go-ethereum/common"

	walletCommon "github.com/status-im/status-go/services/wallet/common"
)

// Deployment pins the contracts a permit swap on a chain may involve. Pinned rather than
// read from LI.FI's /chains, because that response would otherwise decide which contract
// receives the user's unlimited approval and which one the permit is signed against. A
// LI.FI redeploy makes the pins stale, which turns the permit path off for that chain
// until status-go ships an update: the swap falls back to approve-then-swap.
type Deployment struct {
	// Permit2 is Uniswap's singleton, the contract the user approves once per token and
	// the EIP-712 verifyingContract of the PermitTransferFrom.
	Permit2 common.Address
	// Proxy is LI.FI's Permit2Proxy: it pulls the tokens with the user's signature and
	// forwards the swap calldata to the diamond it was deployed against.
	Proxy common.Address
	// Diamond is the LI.FI diamond the proxy forwards to. It is immutable in the proxy, so
	// a quote targeting anything else cannot be routed through the permit path at all.
	Diamond common.Address
}

// canonicalPermit2 is Uniswap's Permit2 singleton. Deployed with CREATE2, so it carries
// the same address on every chain the permit path is enabled for.
var canonicalPermit2 = common.HexToAddress("0x000000000022D473030F116dDEE9F6B43aC78BA3")

// LI.FI's Permit2Proxy and diamond, as published on https://li.quest/v1/chains.
var (
	lifiPermit2Proxy = common.HexToAddress("0x89c6340B1a1f4b25D36cd8B063D49045caF3f818")
	lifiDiamond      = common.HexToAddress("0x1231DEB6f5749EF6cE6943a275A1D3E7486F4EaE")
)

// deployments lists the chains where a swap may be routed through LI.FI's Permit2Proxy,
// collapsing the ERC-20 approval and the swap into a single transaction. Membership is
// what enables the path: no pinned deployment, no permit.
//
// Abstract and zkSync are left out on purpose: their gas accounting doesn't match the
// buffered estimate this path relies on, and their Permit2 is not the canonical one.
var deployments = map[uint64]Deployment{
	walletCommon.EthereumMainnet: {Permit2: canonicalPermit2, Proxy: lifiPermit2Proxy, Diamond: lifiDiamond},
	walletCommon.OptimismMainnet: {Permit2: canonicalPermit2, Proxy: lifiPermit2Proxy, Diamond: lifiDiamond},
	walletCommon.BaseMainnet:     {Permit2: canonicalPermit2, Proxy: lifiPermit2Proxy, Diamond: lifiDiamond},
	walletCommon.ArbitrumMainnet: {Permit2: canonicalPermit2, Proxy: lifiPermit2Proxy, Diamond: lifiDiamond},
}

// DeploymentForChain returns the pinned contracts for the chain, and false when the permit
// path is not enabled there. Callers fall back to the regular approve-then-swap flow when
// it returns false.
func DeploymentForChain(chainID uint64) (Deployment, bool) {
	deployment, ok := deployments[chainID]
	return deployment, ok
}

// TrustedAddresses reports whether the given Permit2 and Permit2Proxy are the ones pinned
// for the chain.
func TrustedAddresses(chainID uint64, permit2, proxy common.Address) bool {
	deployment, ok := deployments[chainID]
	return ok && deployment.Permit2 == permit2 && deployment.Proxy == proxy
}

// ValidateSwapTarget checks that the permit about to be packed only involves pinned
// contracts, target being the address the wrapped calldata is aimed at.
//
// The route-time gate already covers this; it is repeated at build time because the quote
// is fetched again there and could name a different target.
func ValidateSwapTarget(chainID uint64, details *Details, target common.Address) error {
	if details == nil {
		return ErrMissingPermitDetails
	}

	deployment, ok := deployments[chainID]
	if !ok {
		return ErrChainNotEnabled
	}
	if details.ChainID != chainID || details.Spender != deployment.Proxy || target != deployment.Diamond {
		return ErrUntrustedPermitTarget
	}
	if details.Type == TypePermit2 && details.Permit2 != deployment.Permit2 {
		return ErrUntrustedPermitTarget
	}

	return nil
}
