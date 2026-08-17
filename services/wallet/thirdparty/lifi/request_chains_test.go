package lifi

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
)

func TestChainsResponseMapping(t *testing.T) {
	const payload = `{
		"chains": [
			{
				"id": 1,
				"permit2": "0x000000000022D473030F116dDEE9F6B43aC78BA3",
				"permit2Proxy": "0x89c6340B1a1f4b25D36cd8B063D49045caF3f818"
			},
			{
				"id": 999,
				"permit2": "",
				"permit2Proxy": "not-an-address"
			}
		]
	}`

	var parsed chainsResponse
	require.NoError(t, json.Unmarshal([]byte(payload), &parsed))
	require.Len(t, parsed.Chains, 2)

	require.Equal(t, common.HexToAddress("0x000000000022D473030F116dDEE9F6B43aC78BA3"),
		parseAddress(parsed.Chains[0].Permit2))
	require.Equal(t, common.HexToAddress("0x89c6340B1a1f4b25D36cd8B063D49045caF3f818"),
		parseAddress(parsed.Chains[0].Permit2Proxy))

	// A chain without usable addresses maps to zero rather than failing the whole fetch.
	require.Equal(t, common.Address{}, parseAddress(parsed.Chains[1].Permit2))
	require.Equal(t, common.Address{}, parseAddress(parsed.Chains[1].Permit2Proxy))
}

func TestParseAddress(t *testing.T) {
	require.Equal(t, common.HexToAddress("0x1231DEB6f5749EF6cE6943a275A1D3E7486F4EaE"),
		parseAddress("0x1231DEB6f5749EF6cE6943a275A1D3E7486F4EaE"))
	require.Equal(t, common.Address{}, parseAddress(""))
	require.Equal(t, common.Address{}, parseAddress("not-an-address"))
	require.Equal(t, common.Address{}, parseAddress("0x1234"))
}
