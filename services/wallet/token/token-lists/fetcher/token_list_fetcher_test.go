package fetcher

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFetchingTokensList(t *testing.T) {
	walletDb, closeFn := SetupTestWalletDB(t)
	server, closeServer := GetTestServer()
	t.Cleanup(func() {
		closeFn()
		closeServer()
	})

	// Copy the token list to avoid modifying the original
	tokenList0 := listOfTokenLists[0]
	tokenList := TokenList{
		ID:        tokenList0.ID,
		SourceURL: strings.ReplaceAll(tokenList0.SourceURL, serverURLPlaceholder, server.URL),
	}

	tokenChannel := make(chan FetchedTokenList, 1)

	tokenListsFetcher := NewTokenListsFetcher(walletDb)

	err := tokenListsFetcher.fetchTokenList(context.TODO(), tokenList, tokenChannel)
	require.NoError(t, err)

	fetchedTokenList := <-tokenChannel

	require.Equal(t, tokenList.ID, fetchedTokenList.ID)
	require.Equal(t, uniswapTokenListJsonResponse, fetchedTokenList.JsonData)
}
