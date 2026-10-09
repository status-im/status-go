package token

import (
	"context"
	"fmt"
	"math/big"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	types "github.com/status-im/status-go/pkg/services/wallet/token/tokenlist"
	tokentypes "github.com/status-im/status-go/pkg/services/wallet/token/types"

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

// referenceIndex answers queries the way the removed Go mirror did, from the
// catalogue's full token page.
type referenceIndex struct {
	tokens    []*types.Token
	byKey     map[string]*types.Token
	byAddress map[[2]any]*types.Token
	byChain   map[uint64][]*types.Token
	aliases   map[[2]any][2]any
}

func newReferenceIndex(tokens []*types.Token) *referenceIndex {
	ref := &referenceIndex{tokens: tokens, byKey: map[string]*types.Token{}, byAddress: map[[2]any]*types.Token{}, byChain: map[uint64][]*types.Token{}, aliases: map[[2]any][2]any{}}
	for _, token := range tokens {
		ref.byKey[token.Key()] = token
		ref.byAddress[[2]any{token.ChainID, token.Address}] = token
		ref.byChain[token.ChainID] = append(ref.byChain[token.ChainID], token)
	}
	skipped := map[string]bool{}
	for _, key := range walletcommon.SkippedTokenKeys() {
		skipped[strings.ToLower(key)] = true
	}
	for chain, addresses := range walletcommon.AdditionalNativeTokenAddresses() {
		for _, address := range addresses {
			if !skipped[types.TokenKey(chain, address)] {
				ref.aliases[[2]any{chain, address}] = [2]any{chain, common.Address{}}
			}
		}
	}
	return ref
}

func (r *referenceIndex) byChainAddress(chain uint64, address common.Address) *types.Token {
	key := [2]any{chain, address}
	if canonical, ok := r.aliases[key]; ok {
		key = canonical
	}
	return r.byAddress[key]
}

func (r *referenceIndex) byKeys(keys []string) []*types.Token {
	result := make([]*types.Token, 0, len(keys))
	for _, key := range keys {
		key = strings.ToLower(key)
		chain, address, ok := types.ChainAndAddressFromTokenKey(key)
		if !ok {
			continue
		}
		if token := r.byChainAddress(chain, address); token != nil {
			result = append(result, token)
		}
	}
	return result
}

func TestTKLQueriesMatchReferenceIndex(t *testing.T) {
	manager, cleanup := setupTestTokenDB(t)
	defer cleanup()
	chains := append(walletcommon.AllChainIDsAsUint64(), 777)
	facade := manager.tokensManager
	require.NoError(t, facade.Start(context.Background(), false, nil))
	all := facade.UniqueTokens()
	require.Greater(t, len(all), 1000)
	ref := newReferenceIndex(all)

	for _, chain := range append(chains, 123456) {
		got := facade.GetTokensByChain(chain)
		require.NotNil(t, got)
		require.Equal(t, len(ref.byChain[chain]), len(got), chain)
		for i, token := range ref.byChain[chain] {
			require.Equal(t, token, got[i])
		}
	}
	subset := []uint64{walletcommon.EthereumMainnet, walletcommon.BSCMainnet, walletcommon.BaseMainnet}
	var expected []*types.Token
	for _, token := range all {
		if slices.Contains(subset, token.ChainID) {
			expected = append(expected, token)
		}
	}
	require.Equal(t, expected, facade.GetTokensByChains(subset))

	var keys []string
	var ids []types.ChainAddress
	for i, token := range all {
		require.Equal(t, token, ref.byChainAddress(token.ChainID, token.Address))
		got, ok := facade.GetTokenByChainAddress(token.ChainID, token.Address)
		require.True(t, ok)
		require.Equal(t, token, got)
		if i%97 == 0 {
			keys = append(keys, token.Key(), strings.ToUpper(token.Key()), fmt.Sprintf("0%d-%s", token.ChainID, token.Address.Hex()))
			ids = append(ids, types.ChainAddress{ChainID: token.ChainID, Address: token.Address})
		}
	}
	for chain, addresses := range walletcommon.AdditionalNativeTokenAddresses() {
		for _, address := range addresses {
			want := ref.byChainAddress(chain, address)
			got, ok := facade.GetTokenByChainAddress(chain, address)
			require.Equal(t, want != nil, ok)
			require.Equal(t, want, got)
			keys = append(keys, types.TokenKey(chain, address))
			ids = append(ids, types.ChainAddress{ChainID: chain, Address: address})
		}
	}
	for _, key := range walletcommon.SkippedTokenKeys() {
		keys = append(keys, key)
		chain, address, _ := types.ChainAndAddressFromTokenKey(key)
		ids = append(ids, types.ChainAddress{ChainID: chain, Address: address})
	}
	keys = append(keys, "missing", "1-nothex", "1-0x1-extra", "", "1-0x0000000000000000000000000000000000000009", keys[0])
	ids = append(ids, types.ChainAddress{ChainID: 1, Address: common.HexToAddress("0x9")}, ids[0])
	got, err := facade.GetTokensByKeys(keys)
	require.NoError(t, err)
	require.Equal(t, ref.byKeys(keys), got)
	batch := facade.GetTokensByChainAddresses(ids)
	require.Len(t, batch, len(ids))
	for i, id := range ids {
		require.Equal(t, ref.byChainAddress(id.ChainID, id.Address), batch[i], id)
	}
	empty, err := facade.GetTokensByKeys(nil)
	require.NoError(t, err)
	require.NotNil(t, empty)
	require.Empty(t, empty)

	lists := facade.TokenLists()
	require.Greater(t, len(lists), len(initialListIDsFromEmbedded())-1)
	for _, list := range lists {
		single, ok := facade.TokenList(list.ID)
		require.True(t, ok)
		require.Equal(t, list, single)
	}
	_, ok := facade.TokenList("missing")
	require.False(t, ok)
}

func TestTKLRefreshPersistsFetchedBytesForNextLogin(t *testing.T) {
	env := newBenchEnv(t, false)
	defer env.close()
	facade := env.start(t)
	require.NoError(t, facade.TriggerRefresh(context.Background()))
	stored, err := NewContentStore(env.manager.walletDB).GetAll()
	require.NoError(t, err)
	require.Len(t, stored, len(env.lists.bodies))
	for id, body := range env.lists.bodies {
		require.Equal(t, body, stored[id].Data, id)
		require.Equal(t, fmt.Sprintf(`"%s-0"`, id), stored[id].Etag, id)
		require.False(t, stored[id].Fetched.IsZero(), id)
	}
	before := facade.UniqueTokens()
	list, ok := facade.TokenList(walletcommon.StatusTokenListID)
	require.True(t, ok)
	require.NotEqual(t, types.LocalSourceURL, list.Source)
	env.stop(t)

	// The next login loads the persisted bytes, not the bundled lists.
	facade = env.start(t)
	require.Equal(t, before, facade.UniqueTokens())
	reloaded, ok := facade.TokenList(walletcommon.StatusTokenListID)
	require.True(t, ok)
	require.Equal(t, list, reloaded)
}

func TestTKLBatchLookupMatchesSingleLookups(t *testing.T) {
	m, cleanup := setupTestTokenDB(t)
	defer cleanup()
	require.NoError(t, m.tokensManager.Start(context.Background(), false, nil))
	community := common.HexToAddress("0xc0ffee")
	_, err := m.walletDB.Exec("INSERT INTO tokens (network_id,address,name,symbol,decimals,community_id) VALUES (?,?,?,?,?,?)", 1, community, "Community", "COMM", 18, "community")
	require.NoError(t, err)
	usdc := common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")
	ids := []types.ChainAddress{{ChainID: 1, Address: usdc}, {ChainID: 1, Address: community}, {ChainID: 1, Address: common.HexToAddress("0x9")}, {ChainID: walletcommon.ZkSyncMainnet, Address: walletcommon.ZkSyncETHTokenAddress()}, {ChainID: 1}, {ChainID: 1, Address: usdc}}
	tokens, err := m.GetTokensByChainAddresses(ids)
	require.NoError(t, err)
	require.Len(t, tokens, len(ids))
	for i, id := range ids {
		single, err := m.GetTokenByChainAddress(id.ChainID, id.Address)
		if err != nil {
			require.Nil(t, tokens[i], i)
			continue
		}
		require.Equal(t, single, tokens[i], i)
	}
	require.Equal(t, "COMM", tokens[1].Symbol)
	require.Nil(t, tokens[2])
	require.NotSame(t, tokens[0].Token, tokens[5].Token)
}

func TestTKLChainTokensMatchTokensByChains(t *testing.T) {
	m, cleanup := setupTestTokenDB(t)
	defer cleanup()
	require.NoError(t, m.tokensManager.Start(context.Background(), false, nil))
	community := common.HexToAddress("0xc0ffee")
	_, err := m.walletDB.Exec("INSERT INTO tokens (network_id,address,name,symbol,decimals,community_id) VALUES (?,?,?,?,?,?)", walletcommon.BaseMainnet, community, "Community", "COMM", 6, "community")
	require.NoError(t, err)
	chains := walletcommon.AllChainIDsAsUint64()
	sets := [][]uint64{nil, chains, {walletcommon.BaseMainnet}, {walletcommon.EthereumMainnet, walletcommon.BaseMainnet, 777}}
	for _, chain := range chains {
		sets = append(sets, []uint64{chain})
	}
	var dst []tokentypes.ChainToken
	for _, set := range sets {
		tokens, err := m.GetTokensByChains(set)
		require.NoError(t, err)
		dst, err = m.GetChainTokens(set, dst)
		require.NoError(t, err)
		if len(tokens) == 0 {
			require.Empty(t, dst, set)
			continue
		}
		require.Equal(t, tokentypes.ChainTokens(tokens), dst, set)
	}
	base, err := m.GetChainTokens([]uint64{walletcommon.BaseMainnet}, nil)
	require.NoError(t, err)
	require.Contains(t, base, tokentypes.ChainToken{ChainID: walletcommon.BaseMainnet, Address: community, Decimals: 6})
}
