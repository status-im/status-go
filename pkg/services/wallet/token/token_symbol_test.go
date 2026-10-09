package token

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	walletcommon "github.com/status-im/status-go/pkg/services/wallet/common"
	types "github.com/status-im/status-go/pkg/services/wallet/token/tokenlist"
	tokentypes "github.com/status-im/status-go/pkg/services/wallet/token/types"
)

// clientTokenBySymbolOnChain is the client's getTokenBySymbolOnChain: the
// first token of GetTokensByChain whose symbol or name matches, Nim cmpIgnoreCase.
func clientTokenBySymbolOnChain(tokens []*tokentypes.Token, symbol string) *tokentypes.Token {
	for _, token := range tokens {
		if nimCmpIgnoreCase(token.Symbol, symbol) || nimCmpIgnoreCase(token.Name, symbol) {
			return token
		}
	}
	return nil
}

func nimCmpIgnoreCase(a, b string) bool {
	lower := func(c byte) byte {
		if 'A' <= c && c <= 'Z' {
			return c + ('a' - 'A')
		}
		return c
	}
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if lower(a[i]) != lower(b[i]) {
			return false
		}
	}
	return true
}

func TestGetTokenBySymbolOnChainMatchesClientRule(t *testing.T) {
	tm := setupMarketTestManager(t, false)
	communityOnly := &tokentypes.Token{
		Token:         &types.Token{ChainID: walletcommon.EthereumMainnet, Address: common.HexToAddress("0xc0ffee"), Symbol: "COMMSYM", Name: "Community Ünique", Decimals: 18},
		CommunityData: &tokentypes.CommunityData{ID: "community"},
	}
	upsertCommunityToken(t, communityOnly, tm)
	shadowed := &tokentypes.Token{
		Token:         &types.Token{ChainID: walletcommon.EthereumMainnet, Address: common.HexToAddress("0xbeef"), Symbol: "USDC", Name: "Shadowed USDC", Decimals: 6},
		CommunityData: &tokentypes.CommunityData{ID: "community"},
	}
	upsertCommunityToken(t, shadowed, tm)
	otherChain := &tokentypes.Token{
		Token:         &types.Token{ChainID: walletcommon.OptimismMainnet, Address: common.HexToAddress("0xfeed"), Symbol: "OPCOMM", Name: "Optimism Community", Decimals: 6},
		CommunityData: &tokentypes.CommunityData{ID: "community"},
	}
	upsertCommunityToken(t, otherChain, tm)

	probes := 0
	for _, chainID := range []uint64{walletcommon.EthereumMainnet, walletcommon.OptimismMainnet, walletcommon.BSCMainnet, walletcommon.EthereumSepolia, 999999} {
		tokens, err := tm.GetTokensByChain(chainID)
		require.NoError(t, err)
		symbols := []string{"", "no-such-token", "usdc", "USDC", "UsDc", "eth", "Ether", "bnb", "snt", "stt",
			"commsym", "COMMUNITY ÜNIQUE", "community ünique", "opcomm", "OPTIMISM COMMUNITY", "shadowed usdc",
			"Kelvin", "uSdc"}
		for _, token := range tokens {
			symbols = append(symbols, token.Symbol, token.Name)
			if len(symbols)%7 == 0 {
				symbols = append(symbols, swapCase(token.Symbol), swapCase(token.Name))
			}
		}
		for _, symbol := range symbols {
			want := clientTokenBySymbolOnChain(tokens, symbol)
			got, err := tm.GetTokenBySymbolOnChain(chainID, symbol)
			require.NoError(t, err)
			require.Equal(t, want, got, "chain %d symbol %q", chainID, symbol)
			probes++
		}
	}
	require.Greater(t, probes, 1000)

	found, err := tm.GetTokenBySymbolOnChain(walletcommon.EthereumMainnet, "commsym")
	require.NoError(t, err)
	require.Equal(t, communityOnly.Key(), found.Key(), "community customs answer when the catalogue has no match")
	found, err = tm.GetTokenBySymbolOnChain(walletcommon.EthereumMainnet, "community ünique")
	require.NoError(t, err)
	require.Nil(t, found, "only ASCII letters fold")
	found, err = tm.GetTokenBySymbolOnChain(walletcommon.EthereumMainnet, "COMMUNITY Ünique")
	require.NoError(t, err)
	require.Equal(t, communityOnly.Key(), found.Key(), "names match too")
	found, err = tm.GetTokenBySymbolOnChain(walletcommon.EthereumMainnet, "usdc")
	require.NoError(t, err)
	require.NotEqual(t, shadowed.Key(), found.Key(), "catalogue tokens come first")
	require.Nil(t, found.CommunityData)
	found, err = tm.GetTokenBySymbolOnChain(walletcommon.EthereumMainnet, "opcomm")
	require.NoError(t, err)
	require.Nil(t, found, "customs of other chains do not match")
}

func swapCase(s string) string {
	b := []byte(s)
	for i, c := range b {
		switch {
		case 'a' <= c && c <= 'z':
			b[i] = c - ('a' - 'A')
		case 'A' <= c && c <= 'Z':
			b[i] = c + ('a' - 'A')
		}
	}
	return string(b)
}

func TestGetTokenBySymbolOnChainBeforeCatalogueLoads(t *testing.T) {
	tm, stop := setupTestTokenDB(t)
	defer stop()
	custom := &tokentypes.Token{
		Token:         &types.Token{ChainID: 777, Address: common.HexToAddress("0xc0ffee"), Symbol: "ZIL", Name: "Zilliqa", Decimals: 12},
		CommunityData: &tokentypes.CommunityData{ID: "community"},
	}
	upsertCommunityToken(t, custom, tm)
	found, err := tm.GetTokenBySymbolOnChain(777, "zilliqa")
	require.NoError(t, err)
	require.Equal(t, custom.Key(), found.Key())
	require.NoError(t, tm.tokensManager.Start(context.Background(), false, nil))
	found, err = tm.GetTokenBySymbolOnChain(777, "zil")
	require.NoError(t, err)
	require.Equal(t, custom.Key(), found.Key())
}
