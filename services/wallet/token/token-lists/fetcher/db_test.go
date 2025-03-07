package fetcher

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDbActions(t *testing.T) {
	walletDb, closeFn := SetupTestWalletDB(t)
	t.Cleanup(closeFn)

	tokenListsFetcher := NewTokenListsFetcher(walletDb)

	tokenListsFetched := []FetchedTokenList{
		{
			TokenList: TokenList{
				ID:        defaultTokensList[0].ID,
				SourceURL: defaultTokensList[0].SourceURL,
				Schema:    defaultTokensList[0].Schema,
			},
			Fetched:  time.Now().Add(-48 * time.Hour),
			JsonData: uniswapTokenListJsonResponse,
		},
		{
			TokenList: TokenList{
				ID:        defaultTokensList[1].ID,
				SourceURL: defaultTokensList[1].SourceURL,
				Schema:    defaultTokensList[1].Schema,
			},
			Fetched:  time.Now().Add(-48 * time.Hour),
			JsonData: aaveTokenListJsonResponse,
		},
	}

	for _, tokenList := range tokenListsFetched {
		err := tokenListsFetcher.StoreTokenList(tokenList.ID, tokenList.JsonData)
		require.NoError(t, err)
	}

	dbTokenLists, err := tokenListsFetcher.GetAllTokenLists()
	require.NoError(t, err)
	require.Len(t, dbTokenLists, len(tokenListsFetched))
	uniswapIndex := 0
	if dbTokenLists[0].ID == "aave" {
		uniswapIndex = 1
	}

	require.Equal(t, tokenListsFetched[0].ID, dbTokenLists[uniswapIndex].ID)
	require.Equal(t, tokenListsFetched[0].JsonData, dbTokenLists[uniswapIndex].JsonData)
	require.True(t, dbTokenLists[uniswapIndex].Fetched.Compare(tokenListsFetched[0].Fetched) == 1)

	require.Equal(t, tokenListsFetched[1].ID, dbTokenLists[1-uniswapIndex].ID)
	require.Equal(t, tokenListsFetched[1].JsonData, dbTokenLists[1-uniswapIndex].JsonData)
	require.True(t, dbTokenLists[1-uniswapIndex].Fetched.Compare(tokenListsFetched[1].Fetched) == 1)
}
