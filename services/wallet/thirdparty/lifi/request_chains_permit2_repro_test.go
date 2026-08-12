package lifi

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
)

// ---------------------------------------------------------------------------
// Finding 3: untrusted Permit2 addresses from LI.FI
//
// FetchChains takes "permit2" and "permit2Proxy" straight from the /chains response and
// stores them in ChainInfo without any check beyond "is it hex". processor_lifi.go gates
// the permit path only on ChainInfo.SupportsPermit2() (~line 171) and then hands those
// addresses to permit2.Resolver.Resolve, which makes the attacker-supplied Permit2 the
// EIP-712 verifyingContract the user signs a token-spending permit against (~line 382
// packs the calldata against the equally unchecked Permit2Proxy). A compromised or spoofed
// API response therefore chooses the contract that gets to pull the user's tokens.
//
// The canonical Uniswap Permit2 singleton is 0x000000000022D473030F116dDEE9F6B43aC78BA3
// on every chain the permit path is actually enabled for (permit2.EnabledForChain:
// Ethereum 1, Optimism 10, Base 8453, Arbitrum 42161). The two chains whose Permit2 is
// non-canonical - zkSync and Abstract, called out in the ChainInfo doc comment - are
// deliberately excluded from that list, so pinning the singleton costs nothing today.
//
// Seam: FetchChains talks to the hardcoded baseURL through a non-injectable HTTPClient, so
// there is no cheap httptest seam. These tests target the lowest layer where the decision
// should live: the JSON -> ChainInfo mapping and the SupportsPermit2 gate the processor
// consults. Both are package-private, hence an in-package test.
// ---------------------------------------------------------------------------

// canonicalPermit2 is Uniswap's Permit2 singleton, deployed at the same address on all
// chains where the permit path is enabled.
const canonicalPermit2 = "0x000000000022D473030F116dDEE9F6B43aC78BA3"

// lifiPermit2Proxy is LI.FI's Permit2Proxy on Ethereum mainnet.
const lifiPermit2Proxy = "0x1231DEB6f5749EF6cE6943a275A1D3E7486F4EaE"

// attackerPermit2 stands in for any address a compromised or spoofed /chains response
// could put in the permit2 field.
const attackerPermit2 = "0xdEAD000000000000000000000000000000000bad"

func TestChainInfo_SupportsPermit2_RejectsNonCanonicalPermit2(t *testing.T) {
	tests := []struct {
		name    string
		chainID uint64
		permit2 string
		proxy   string
		want    bool
	}{
		{
			// Control: what a healthy LI.FI response looks like. Passes today.
			name:    "canonical permit2 on Ethereum",
			chainID: 1,
			permit2: canonicalPermit2,
			proxy:   lifiPermit2Proxy,
			want:    true,
		},
		{
			name:    "attacker-supplied permit2 on Ethereum",
			chainID: 1,
			permit2: attackerPermit2,
			proxy:   lifiPermit2Proxy,
			want:    false,
		},
		{
			name:    "attacker-supplied permit2 on Optimism",
			chainID: 10,
			permit2: attackerPermit2,
			proxy:   lifiPermit2Proxy,
			want:    false,
		},
		{
			// Control: an absent permit2 already disables the path. Passes today.
			name:    "missing permit2",
			chainID: 1,
			permit2: "",
			proxy:   lifiPermit2Proxy,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := ChainInfo{
				ID:           tt.chainID,
				Permit2:      parseAddress(tt.permit2),
				Permit2Proxy: parseAddress(tt.proxy),
			}

			assert.Equal(t, tt.want, info.SupportsPermit2(),
				"the permit path must only be enabled for the canonical Permit2 singleton")
		})
	}
}

// TestChainsResponse_HostilePermit2IsNotTrusted walks the same decision from the raw API
// payload, which is where the untrusted value actually enters. Mirrors the mapping
// FetchChains performs after json.Unmarshal.
func TestChainsResponse_HostilePermit2IsNotTrusted(t *testing.T) {
	const payload = `{
		"chains": [
			{
				"id": 1,
				"permit2": "0xdEAD000000000000000000000000000000000bad",
				"permit2Proxy": "0x1231DEB6f5749EF6cE6943a275A1D3E7486F4EaE",
				"diamondAddress": "0x1231DEB6f5749EF6cE6943a275A1D3E7486F4EaE"
			}
		]
	}`

	var parsed chainsResponse
	require.NoError(t, json.Unmarshal([]byte(payload), &parsed))
	require.Len(t, parsed.Chains, 1)

	ch := parsed.Chains[0]
	info := ChainInfo{
		ID:             ch.ID,
		Permit2:        parseAddress(ch.Permit2),
		Permit2Proxy:   parseAddress(ch.Permit2Proxy),
		DiamondAddress: parseAddress(ch.DiamondAddress),
	}

	// Either the address is dropped during mapping, or the gate the processor consults
	// refuses it. Anything else means the user signs a permit against whatever contract
	// the API named.
	assert.False(t, info.SupportsPermit2(),
		"a chains response naming a non-canonical Permit2 must not enable the permit path")
	assert.NotEqual(t, common.HexToAddress(attackerPermit2), info.Permit2,
		"a non-canonical Permit2 from the API must not be carried into ChainInfo")
}
