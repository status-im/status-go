package tokenlists

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/services/wallet/token/token-lists/fetcher"
)

func TestFetchedListMappers(t *testing.T) {
	var tokenList TokensList
	// check uniswap token list
	err := mapFetchedOtherListToTokenList(fetcher.FetchedTokensList[0], &tokenList)
	require.NoError(t, err)
	require.Len(t, tokenList.Tokens, 0) // cause it doesn't have id parameter

	// check aave token list
	err = mapFetchedOtherListToTokenList(fetcher.FetchedTokensList[1], &tokenList)
	require.NoError(t, err)
	require.Len(t, tokenList.Tokens, 0) // cause it doesn't have id parameter

	// check coingecko token list
	err = mapFetchedCoingeckoListToTokenList(fetcher.FetchedTokensList[2], &tokenList)
	require.NoError(t, err)
	require.Len(t, tokenList.Tokens, 7)
}
