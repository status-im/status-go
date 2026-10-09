package tklmanager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/nim-token-lists/go/tkl"
	"github.com/stretchr/testify/require"

	types "github.com/status-im/status-go/pkg/services/wallet/token/tokenlist"
)

func noStored(context.Context) (tkl.Bootstrap, []tkl.ListBody, error) {
	return tkl.Bootstrap{}, nil, nil
}

func withBodies(bodies ...tkl.ListBody) Loader {
	return func(context.Context) (tkl.Bootstrap, []tkl.ListBody, error) { return tkl.Bootstrap{}, bodies, nil }
}

func TestBootstrapAndOwnedReads(t *testing.T) {
	m, err := New(tkl.Config{Chains: []uint64{1}}, func(context.Context) (tkl.Bootstrap, []tkl.ListBody, error) {
		return tkl.Bootstrap{Customs: []tkl.Token{{ChainID: 1, Address: "0x0000000000000000000000000000000000000001", Symbol: "ONE", Decimals: 18}}}, nil, nil
	})
	require.NoError(t, err)
	defer func() { _ = m.Stop() }()
	require.NoError(t, m.Start(context.Background(), false, nil))
	token, ok := m.GetTokenByChainAddress(1, common.HexToAddress("0x1"))
	require.True(t, ok)
	require.Equal(t, "ONE", token.Symbol)
	token.Symbol = "changed"
	again, ok := m.GetTokenByChainAddress(1, common.HexToAddress("0x1"))
	require.True(t, ok)
	require.Equal(t, "ONE", again.Symbol)
	byKey, err := m.GetTokensByKeys([]string{"01-0x1"})
	require.NoError(t, err)
	require.Len(t, byKey, 1)
	require.Equal(t, "ONE", byKey[0].Symbol)
	require.NoError(t, m.SetChains([]uint64{10}))
	_, ok = m.GetTokenByChainAddress(1, common.HexToAddress("0x1"))
	require.False(t, ok)
	require.NoError(t, m.SetChains([]uint64{1}))
	_, ok = m.GetTokenByChainAddress(1, common.HexToAddress("0x1"))
	require.True(t, ok)
}

func TestStartRetryAndPrestartChains(t *testing.T) {
	fail := true
	m, err := New(tkl.Config{Chains: []uint64{1}}, func(context.Context) (tkl.Bootstrap, []tkl.ListBody, error) {
		if fail {
			return tkl.Bootstrap{}, nil, errors.New("storage failed")
		}
		return tkl.Bootstrap{}, nil, nil
	})
	require.NoError(t, err)
	defer func() { _ = m.Stop() }()
	require.NoError(t, m.SetChains([]uint64{10}))
	require.Error(t, m.Start(context.Background(), false, nil))
	require.Empty(t, m.UniqueTokens())
	fail = false
	require.NoError(t, m.Start(context.Background(), false, nil))
	_, ok := m.GetTokenByChainAddress(10, common.Address{})
	require.True(t, ok)
	require.NoError(t, m.Stop())
	require.NoError(t, m.Stop())
	require.ErrorIs(t, m.Start(context.Background(), false, nil), tkl.Closed)
}

func TestListsFollowChains(t *testing.T) {
	m, err := New(tkl.Config{Chains: []uint64{1}}, noStored)
	require.NoError(t, err)
	defer func() { _ = m.Stop() }()
	require.Nil(t, m.TokenLists())
	require.NoError(t, m.Start(context.Background(), false, nil))
	require.NotEmpty(t, m.TokenLists())
	require.NoError(t, m.SetChains([]uint64{10}))
	lists := m.TokenLists()
	require.NotEmpty(t, lists)
	for _, list := range lists {
		for _, token := range list.Tokens {
			require.Equal(t, uint64(10), token.ChainID)
		}
	}
	require.NoError(t, m.Stop())
	require.Nil(t, m.TokenLists())
	require.Nil(t, m.UniqueTokens())
}

func TestAliasPolicyRebuiltWithRevision(t *testing.T) {
	alias := tkl.Identity{ChainID: 1, Address: "0x0000000000000000000000000000000000000002"}
	config := tkl.Config{Chains: []uint64{1}, Policy: tkl.Policy{NativeAliases: []tkl.Identity{alias}}}
	m, err := New(config, noStored)
	require.NoError(t, err)
	defer func() { _ = m.Stop() }()
	config.Policy.NativeAliases[0].Address = "0x0000000000000000000000000000000000000003"
	require.NoError(t, m.Start(context.Background(), false, nil))
	_, ok := m.GetTokenByChainAddress(1, common.HexToAddress(alias.Address))
	require.True(t, ok)
	// Exercise the documented policy-update protocol without adding a public
	// policy API before application wiring needs one.
	m.mu.Lock()
	err = m.setPolicy(tkl.Policy{NativeAliases: []tkl.Identity{alias}, SkippedKeys: []string{types.TokenKey(1, common.HexToAddress(alias.Address))}})
	m.mu.Unlock()
	require.NoError(t, err)
	_, ok = m.GetTokenByChainAddress(1, common.HexToAddress(alias.Address))
	require.False(t, ok)
	found, err := m.GetTokensByKeys([]string{types.TokenKey(1, common.HexToAddress(alias.Address))})
	require.NoError(t, err)
	require.Empty(t, found)
}

func TestAliasesListsAndConcurrentReads(t *testing.T) {
	const document = `{"name":"Test","timestamp":"2026-01-01T00:00:00Z","version":{"major":1,"minor":0,"patch":0},"tags":{"nested":{"name":"original"}},"tokens":[{"chainId":1,"address":"0x0000000000000000000000000000000000000001","symbol":"ONE","name":"One","decimals":18}]}`
	m, err := New(tkl.Config{Chains: []uint64{1}, InitialLists: []tkl.ListContent{{ID: "test", Format: tkl.StandardFormat}}, Policy: tkl.Policy{NativeAliases: []tkl.Identity{{ChainID: 1, Address: "0x0000000000000000000000000000000000000002"}}}}, withBodies(tkl.ListBody{ID: "test", Origin: tkl.Bundled, Data: []byte(document)}))
	require.NoError(t, err)
	defer func() { _ = m.Stop() }()
	require.NoError(t, m.Start(context.Background(), false, nil))
	tokens, err := m.GetTokensByKeys([]string{"1-0x0000000000000000000000000000000000000002", "missing"})
	require.NoError(t, err)
	require.Len(t, tokens, 1)
	require.True(t, tokens[0].IsNative())
	list, ok := m.TokenList("test")
	require.True(t, ok)
	list.Tags["nested"].(map[string]interface{})["name"] = "modified"
	list.Tokens[0].Symbol = "modified"
	list, ok = m.TokenList("test")
	require.True(t, ok)
	require.Equal(t, "original", list.Tags["nested"].(map[string]interface{})["name"])
	require.Equal(t, "ONE", list.Tokens[0].Symbol)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				m.UniqueTokens()
				m.GetTokensByChain(1)
				m.TokenLists()
				m.GetTokenByChainAddress(1, common.Address{})
			}
		}()
	}
	for i := 0; i < 10; i++ {
		require.NoError(t, m.SetChains([]uint64{1, 10}))
		require.NoError(t, m.SetChains([]uint64{1}))
	}
	require.NoError(t, m.Stop())
	wg.Wait()
}

var benchmarkToken *types.Token

func BenchmarkLookup(b *testing.B) {
	m, err := New(tkl.Config{Chains: []uint64{1}}, noStored)
	require.NoError(b, err)
	defer func() { _ = m.Stop() }()
	require.NoError(b, m.Start(context.Background(), false, nil))
	b.ReportAllocs()
	for b.Loop() {
		benchmarkToken, _ = m.GetTokenByChainAddress(1, common.Address{})
	}
}

func BenchmarkUniqueTokens(b *testing.B) {
	tokens := make([]tkl.Token, 10000)
	for i := range tokens {
		tokens[i] = tkl.Token{ChainID: 1, Address: fmt.Sprintf("0x%040x", i+1), Symbol: "TOKEN", Name: "Token", Decimals: 18}
	}
	body, err := json.Marshal(map[string]any{"name": "Benchmark", "version": map[string]int{"major": 1, "minor": 0, "patch": 0}, "tokens": tokens})
	require.NoError(b, err)
	m, err := New(tkl.Config{Chains: []uint64{1}, InitialLists: []tkl.ListContent{{ID: "benchmark", Format: tkl.StandardFormat}}}, withBodies(tkl.ListBody{ID: "benchmark", Origin: tkl.Bundled, Data: body}))
	require.NoError(b, err)
	defer func() { _ = m.Stop() }()
	require.NoError(b, m.Start(context.Background(), false, nil))
	require.Len(b, m.UniqueTokens(), 10001)
	b.ReportAllocs()
	for b.Loop() {
		if len(m.UniqueTokens()) != 10001 {
			b.Fatal("unexpected token count")
		}
	}
}

func TestLoadPrefersValidStoredBodyAndDoesNotRetainBodies(t *testing.T) {
	const document = `{"name":"Test","timestamp":"2026-01-01T00:00:00Z","version":{"major":1,"minor":0,"patch":0},"tokens":[{"chainId":1,"address":"0x000000000000000000000000000000000000000%d","symbol":"%s","name":"Token","decimals":18}]}`
	for _, tc := range []struct {
		name   string
		stored string
		want   string
	}{
		{"valid", fmt.Sprintf(document, 2, "STORED"), "0x2"},
		{"corrupt", "broken", "0x1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bundled := []byte(fmt.Sprintf(document, 1, "BUNDLED"))
			stored := []byte(tc.stored)
			m, err := New(tkl.Config{Chains: []uint64{1}, InitialLists: []tkl.ListContent{{ID: "test", Format: tkl.StandardFormat}}}, func(context.Context) (tkl.Bootstrap, []tkl.ListBody, error) {
				return tkl.Bootstrap{Stored: []tkl.ListContent{{ID: "test", Format: tkl.StandardFormat, Source: "https://example.org/list", ETag: "v1", FetchedAt: 1}}},
					[]tkl.ListBody{{ID: "test", Origin: tkl.Bundled, Data: bundled}, {ID: "test", Origin: tkl.Stored, Data: stored}}, nil
			})
			require.NoError(t, err)
			defer func() { _ = m.Stop() }()
			require.NoError(t, m.Start(context.Background(), false, nil))
			for _, buf := range [][]byte{bundled, stored} {
				for i := range buf {
					buf[i] = 'x'
				}
			}
			for _, address := range []string{"0x1", "0x2"} {
				_, ok := m.GetTokenByChainAddress(1, common.HexToAddress(address))
				require.Equal(t, address == tc.want, ok, address)
			}
			token, ok := m.GetTokenByChainAddress(1, common.HexToAddress(tc.want))
			require.True(t, ok)
			require.Contains(t, []string{"STORED", "BUNDLED"}, token.Symbol)
		})
	}
}

func TestBatchLookupAlignsWithRequests(t *testing.T) {
	const document = `{"name":"Test","timestamp":"2026-01-01T00:00:00Z","version":{"major":1,"minor":0,"patch":0},"tokens":[{"chainId":1,"address":"0x0000000000000000000000000000000000000001","symbol":"ONE","name":"One","decimals":18},{"chainId":10,"address":"0x0000000000000000000000000000000000000001","symbol":"OPONE","name":"One","decimals":6}]}`
	alias := common.HexToAddress("0x2")
	m, err := New(tkl.Config{Chains: []uint64{1, 10}, InitialLists: []tkl.ListContent{{ID: "test", Format: tkl.StandardFormat}}, Policy: tkl.Policy{NativeAliases: []tkl.Identity{{ChainID: 1, Address: alias.Hex()}}}}, withBodies(tkl.ListBody{ID: "test", Origin: tkl.Bundled, Data: []byte(document)}))
	require.NoError(t, err)
	defer func() { _ = m.Stop() }()
	require.Equal(t, make([]*types.Token, 1), m.GetTokensByChainAddresses([]types.ChainAddress{{ChainID: 1}}))
	require.NoError(t, m.Start(context.Background(), false, nil))
	one := common.HexToAddress("0x1")
	ids := []types.ChainAddress{{ChainID: 1, Address: common.HexToAddress("0x9")}, {ChainID: 1}, {ChainID: 10, Address: one}, {ChainID: 1, Address: alias}, {ChainID: 1, Address: one}, {ChainID: 10, Address: one}, {ChainID: 56, Address: one}}
	tokens := m.GetTokensByChainAddresses(ids)
	require.Len(t, tokens, len(ids))
	for i, id := range ids {
		single, ok := m.GetTokenByChainAddress(id.ChainID, id.Address)
		require.Equal(t, ok, tokens[i] != nil, i)
		require.Equal(t, single, tokens[i], i)
	}
	require.Nil(t, tokens[0])
	require.Equal(t, "OPONE", tokens[2].Symbol)
	require.True(t, tokens[3].IsNative())
	require.NotSame(t, tokens[2], tokens[5])
	require.Empty(t, m.GetTokensByChainAddresses(nil))
}

func TestBatchLookupReadsAliasesWithItsQuery(t *testing.T) {
	const document = `{"name":"Test","timestamp":"2026-01-01T00:00:00Z","version":{"major":1,"minor":0,"patch":0},"tokens":[{"chainId":1,"address":"0x0000000000000000000000000000000000000001","symbol":"ONE","name":"One","decimals":18}]}`
	mainnet := tkl.Identity{ChainID: 1, Address: "0x0000000000000000000000000000000000000002"}
	optimism := tkl.Identity{ChainID: 10, Address: "0x0000000000000000000000000000000000000002"}
	skip := func(id tkl.Identity) string { return types.TokenKey(id.ChainID, common.HexToAddress(id.Address)) }
	aliases := []tkl.Identity{mainnet, optimism}
	m, err := New(tkl.Config{Chains: []uint64{1, 10}, InitialLists: []tkl.ListContent{{ID: "test", Format: tkl.StandardFormat}}, Policy: tkl.Policy{NativeAliases: aliases, SkippedKeys: []string{skip(optimism)}}}, withBodies(tkl.ListBody{ID: "test", Origin: tkl.Bundled, Data: []byte(document)}))
	require.NoError(t, err)
	defer func() { _ = m.Stop() }()
	require.NoError(t, m.Start(context.Background(), false, nil))
	ids := []types.ChainAddress{{ChainID: 1, Address: common.HexToAddress(mainnet.Address)}, {ChainID: 10, Address: common.HexToAddress(optimism.Address)}}
	before := m.GetTokensByChainAddresses(ids)
	require.True(t, before[0].IsNative())
	require.Nil(t, before[1])

	// Swap which alias is active between reading the aliases and querying.
	switched := false
	testHookBeforeBatchQuery = func() {
		if switched {
			return
		}
		switched = true
		m.mu.Lock()
		defer m.mu.Unlock()
		require.NoError(t, m.setPolicy(tkl.Policy{NativeAliases: aliases, SkippedKeys: []string{skip(mainnet)}}))
	}
	defer func() { testHookBeforeBatchQuery = nil }()
	after := m.GetTokensByChainAddresses(ids)
	require.True(t, switched)
	require.Nil(t, after[0])
	require.NotNil(t, after[1])
	require.True(t, after[1].IsNative())
}

func narrowTokens(tokens []*types.Token) []types.ChainToken {
	result := make([]types.ChainToken, len(tokens))
	for i, token := range tokens {
		result[i] = token.ChainToken()
	}
	return result
}

func TestChainTokensMatchTokensByChains(t *testing.T) {
	const document = `{"name":"Test","timestamp":"2026-01-01T00:00:00Z","version":{"major":1,"minor":0,"patch":0},"tokens":[{"chainId":1,"address":"0x0000000000000000000000000000000000000001","symbol":"ONE","name":"One","decimals":18},{"chainId":10,"address":"0x00000000000000000000000000000000000000AB","symbol":"OPONE","name":"One","decimals":6},{"chainId":1,"address":"0x0000000000000000000000000000000000000003","symbol":"THREE","name":"Three","decimals":0}]}`
	m, err := New(tkl.Config{
		Chains:       []uint64{1, 10, 56},
		InitialLists: []tkl.ListContent{{ID: "test", Format: tkl.StandardFormat}},
		Policy:       tkl.Policy{NativeTokens: []tkl.Token{{ChainID: 1, Address: "0x0000000000000000000000000000000000000000", Symbol: "ETH", Name: "Ether", Decimals: 18}}},
	}, func(context.Context) (tkl.Bootstrap, []tkl.ListBody, error) {
		return tkl.Bootstrap{Customs: []tkl.Token{{ChainID: 56, Address: "0x0000000000000000000000000000000000000005", Symbol: "CUS", Decimals: 9}}},
			[]tkl.ListBody{{ID: "test", Origin: tkl.Bundled, Data: []byte(document)}}, nil
	})
	require.NoError(t, err)
	defer func() { _ = m.Stop() }()
	dst := make([]types.ChainToken, 1, 16)
	require.Empty(t, m.GetChainTokens([]uint64{1}, dst))
	require.NoError(t, m.Start(context.Background(), false, nil))

	check := func() {
		for _, chains := range [][]uint64{nil, {1}, {10}, {56}, {1, 10, 56}, {56, 1}, {10, 10}, {999}} {
			got := m.GetChainTokens(chains, dst)
			require.Equal(t, narrowTokens(m.GetTokensByChains(chains)), got, chains)
			if len(got) > 0 && len(got) <= cap(dst) {
				require.Same(t, &dst[:1][0], &got[0], "fills dst when it has room")
			}
		}
	}
	check()
	native := m.GetChainTokens([]uint64{1}, nil)
	require.True(t, native[0].IsNative())
	require.Equal(t, uint(18), native[0].Decimals)
	require.Len(t, m.GetChainTokens([]uint64{1, 10, 56}, nil), 7)

	m.mu.Lock()
	err = m.setPolicy(tkl.Policy{SkippedKeys: []string{types.TokenKey(1, common.HexToAddress("0x1"))}})
	m.mu.Unlock()
	require.NoError(t, err)
	check()
	require.NoError(t, m.SetChains([]uint64{10}))
	check()

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var buf []types.ChainToken
			for range 50 {
				buf = m.GetChainTokens([]uint64{10}, buf)
				if len(buf) != 2 || buf[1].Decimals != 6 {
					t.Error(buf)
					return
				}
			}
		}()
	}
	wg.Wait()
}
