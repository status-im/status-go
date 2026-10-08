//go:build tkl

package token

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/go-wallet-sdk/pkg/tokens/autofetcher"
	"github.com/stretchr/testify/require"

	types "github.com/status-im/status-go/pkg/services/wallet/token/tokenlist"

	walletcommon "github.com/status-im/status-go/pkg/services/wallet/common"
)

func TestTKLBootstrapExistingDatabase(t *testing.T) {
	manager, cleanup := setupTestTokenDB(t)
	defer cleanup()
	// A corrupt cache must fall back to the embedded Status list.
	require.NoError(t, NewContentStore(manager.walletDB).Set(walletcommon.StatusTokenListID, autofetcher.Content{SourceURL: "https://prod.market.status.im/static/token-list.json", Data: []byte("broken")}))
	facade, err := newTKLReadManager(manager, []uint64{1}, time.Time{})
	require.NoError(t, err)
	defer func() { _ = facade.Stop() }()
	require.NoError(t, facade.Start(context.Background(), false, nil))
	require.NotEmpty(t, facade.UniqueTokens())
	token, ok := facade.GetTokenByChainAddress(1, common.Address{})
	require.True(t, ok)
	require.Equal(t, "ETH", token.Symbol)
	list, ok := facade.TokenList(walletcommon.StatusTokenListID)
	require.True(t, ok)
	require.NotEmpty(t, list.Tokens)
	// Bootstrap must not rewrite or delete rows used by the rollback path.
	stored, err := NewContentStore(manager.walletDB).Get(walletcommon.StatusTokenListID)
	require.NoError(t, err)
	require.Equal(t, []byte("broken"), stored.Data)
}

func TestTKLBootstrapInvalidAndCommunityCustoms(t *testing.T) {
	m, cleanup := setupTestTokenDB(t)
	defer cleanup()
	for i, decimals := range []uint{18, 256, 19} {
		_, err := m.walletDB.Exec("INSERT INTO tokens (network_id,address,name,symbol,decimals) VALUES (?,?,?,?,?)", 1, common.BigToAddress(big.NewInt(int64(i+1))), "Custom", "CUSTOM", decimals)
		require.NoError(t, err)
	}
	_, err := m.walletDB.Exec("INSERT INTO tokens (network_id,address,name,symbol,decimals,community_id) VALUES (?,?,?,?,?,?)", 1, common.HexToAddress("0x4"), "Community", "COMM", 18, "community")
	require.NoError(t, err)
	facade, err := newTKLReadManager(m, []uint64{1}, time.Time{})
	require.NoError(t, err)
	defer func() { _ = facade.Stop() }()
	require.NoError(t, facade.Start(context.Background(), false, nil))
	custom, ok := facade.GetTokenByChainAddress(1, common.HexToAddress("0x1"))
	require.True(t, ok)
	require.True(t, custom.CustomToken) // The core identifies ordinary custom tokens.
	for _, address := range []string{"0x2", "0x3", "0x4"} {
		_, ok := facade.GetTokenByChainAddress(1, common.HexToAddress(address))
		require.False(t, ok, address)
	}
}

func TestTKLEmbeddedReadParity(t *testing.T) {
	manager, cleanup := setupTestTokenDB(t)
	defer cleanup()
	chains := walletcommon.AllChainIDsAsUint64()
	facade, err := newTKLReadManager(manager, chains, time.Time{})
	require.NoError(t, err)
	defer func() { _ = facade.Stop() }()
	old, err := setUpTokenListsManager(manager, manager.walletDB, chains, time.Time{}, time.Hour, time.Minute)
	require.NoError(t, err)
	require.NoError(t, old.Start(context.Background(), false, nil))
	defer func() { _ = old.Stop() }()
	require.NoError(t, facade.Start(context.Background(), false, nil))
	byKey := func(tokens []*types.Token) map[string]*types.Token {
		result := make(map[string]*types.Token)
		for _, token := range tokens {
			result[token.Key()] = token
		}
		return result
	}
	require.Equal(t, byKey(old.UniqueTokens()), byKey(facade.UniqueTokens()))
	for _, chain := range chains {
		require.Equal(t, byKey(old.GetTokensByChain(chain)), byKey(facade.GetTokensByChain(chain)))
	}
	for _, list := range old.TokenLists() {
		actual, ok := facade.TokenList(list.ID)
		require.True(t, ok, list.ID)
		require.Equal(t, list.Source, actual.Source, list.ID)
		require.Equal(t, list.FetchedTimestamp, actual.FetchedTimestamp, list.ID)
		require.Equal(t, byKey(list.Tokens), byKey(actual.Tokens), list.ID)
	}
}
