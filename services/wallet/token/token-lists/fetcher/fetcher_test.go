package fetcher

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFetcher(t *testing.T) {
	walletDb, closeFn := SetupTestWalletDB(t)
	server, closeServer := GetTestServer()
	t.Cleanup(func() {
		closeFn()
		closeServer()
	})

	tokenListsFetcher := NewTokenListsFetcher(walletDb)

	ctx := context.Background()

	tokenListsFetcher.SetURLOfRemoteListOfTokenLists(server.URL)

	storedListsCount, err := tokenListsFetcher.FetchAndStore(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, storedListsCount)
}
