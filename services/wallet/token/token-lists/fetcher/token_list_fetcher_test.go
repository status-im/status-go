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

	ctx := context.TODO()

	// Copy the token list to avoid modifying the original
	tokenList0 := listOfTokenLists[0]
	tokenList := TokenList{
		ID:        tokenList0.ID,
		SourceURL: strings.ReplaceAll(tokenList0.SourceURL, serverURLPlaceholder, server.URL),
	}

	tokenChannel := make(chan FetchedTokenList, 1)

	tokenListsFetcher := NewTokenListsFetcher(walletDb)

	// Fetch the token list for the first time, for an empty Etag, when the server returns the Etag
	err := tokenListsFetcher.fetchTokenList(ctx, tokenList, "", tokenChannel)
	require.NoError(t, err)

	fetchedTokenList := <-tokenChannel

	require.Equal(t, tokenList.ID, fetchedTokenList.ID)
	require.Equal(t, UniswapEtag, fetchedTokenList.Etag)
	require.Equal(t, uniswapTokenListJsonResponse, fetchedTokenList.JsonData)

	// Fetch the token list again using the previously returned Etag when the server returns the same Etag with status 304 (http.StatusNotModified)
	tokenList.SourceURL = server.URL + "/uniswap-same-etag.json"

	err = tokenListsFetcher.fetchTokenList(ctx, tokenList, fetchedTokenList.Etag, tokenChannel)
	require.NoError(t, err)

	// Fetch the token list again using the previously returned Etag when the server returns a new Etag
	tokenList.SourceURL = server.URL + "/uniswap-new-etag.json"

	err = tokenListsFetcher.fetchTokenList(ctx, tokenList, fetchedTokenList.Etag, tokenChannel)
	require.NoError(t, err)

	fetchedTokenList = <-tokenChannel

	require.Equal(t, tokenList.ID, fetchedTokenList.ID)
	require.Equal(t, UniswapNewEtag, fetchedTokenList.Etag)
	require.Equal(t, uniswapTokenListJsonResponse, fetchedTokenList.JsonData)
}
