package token

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	types "github.com/status-im/status-go/pkg/services/wallet/token/tokenlist"

	walletcommon "github.com/status-im/status-go/pkg/services/wallet/common"
)

func TestTKLBootstrapExistingDatabase(t *testing.T) {
	manager, cleanup := setupTestTokenDB(t)
	defer cleanup()
	// A corrupt cache must fall back to the embedded Status list.
	require.NoError(t, NewContentStore(manager.walletDB).Set(walletcommon.StatusTokenListID, storedContent{SourceURL: "https://prod.market.status.im/static/token-list.json", Data: []byte("broken")}))
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
	// Bootstrap must not rewrite or delete cached source data.
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

func TestTKLEmbeddedCatalogue(t *testing.T) {
	manager, cleanup := setupTestTokenDB(t)
	defer cleanup()
	chains := walletcommon.AllChainIDsAsUint64()
	facade, err := newTKLReadManager(manager, chains, time.Time{})
	require.NoError(t, err)
	defer func() { require.NoError(t, facade.Stop()) }()
	require.NoError(t, facade.Start(context.Background(), false, nil))
	require.NotEmpty(t, facade.UniqueTokens())
	for _, chain := range chains {
		native, ok := facade.GetTokenByChainAddress(chain, common.Address{})
		require.True(t, ok)
		symbol, name := walletcommon.EthSymbol, walletcommon.EthName
		if chain == walletcommon.BSCMainnet || chain == walletcommon.BSCTestnet {
			symbol, name = walletcommon.BNBSymbol, walletcommon.BNBName
		}
		require.Equal(t, symbol, native.Symbol)
		require.Equal(t, name, native.Name)
		require.Equal(t, uint(18), native.Decimals)
		for _, token := range facade.GetTokensByChain(chain) {
			found, ok := facade.GetTokenByChainAddress(chain, token.Address)
			require.True(t, ok)
			require.Equal(t, token, found)
		}
	}
	for _, id := range initialListIDsFromEmbedded() {
		list, ok := facade.TokenList(id)
		require.True(t, ok, id)
		require.NotEmpty(t, list.Tokens, id)
		require.Equal(t, types.LocalSourceURL, list.Source)
		require.Equal(t, (time.Time{}).Format(time.RFC3339), list.FetchedTimestamp)
	}
}
