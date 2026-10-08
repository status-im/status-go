//go:build tkl

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

func TestBootstrapAndOwnedReads(t *testing.T) {
	m, err := New(tkl.Config{Chains: []uint64{1}}, func(context.Context) (tkl.Bootstrap, error) {
		return tkl.Bootstrap{Customs: []tkl.Token{{ChainID: 1, Address: "0x0000000000000000000000000000000000000001", Symbol: "ONE", Decimals: 18}}}, nil
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
	m, err := New(tkl.Config{Chains: []uint64{1}}, func(context.Context) (tkl.Bootstrap, error) {
		if fail {
			return tkl.Bootstrap{}, errors.New("storage failed")
		}
		return tkl.Bootstrap{}, nil
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

func TestListsLoadOnlyOnDemandAndInvalidate(t *testing.T) {
	m, err := New(tkl.Config{Chains: []uint64{1}}, func(context.Context) (tkl.Bootstrap, error) { return tkl.Bootstrap{}, nil })
	require.NoError(t, err)
	defer func() { _ = m.Stop() }()
	require.NoError(t, m.Start(context.Background(), false, nil))
	first := m.mirror.Load()
	require.Nil(t, first.lists)
	require.NotEmpty(t, m.TokenLists())
	require.NotNil(t, first.lists)
	require.NoError(t, m.SetChains([]uint64{10}))
	second := m.mirror.Load()
	require.NotSame(t, first, second)
	require.Nil(t, second.lists)
	lists := m.TokenLists()
	require.NotEmpty(t, lists)
	for _, list := range lists {
		for _, token := range list.Tokens {
			require.Equal(t, uint64(10), token.ChainID)
		}
	}
}

func TestAliasPolicyRebuiltWithRevision(t *testing.T) {
	alias := tkl.Identity{ChainID: 1, Address: "0x0000000000000000000000000000000000000002"}
	config := tkl.Config{Chains: []uint64{1}, Policy: tkl.Policy{NativeAliases: []tkl.Identity{alias}}}
	m, err := New(config, func(context.Context) (tkl.Bootstrap, error) { return tkl.Bootstrap{}, nil })
	require.NoError(t, err)
	defer func() { _ = m.Stop() }()
	config.Policy.NativeAliases[0].Address = "0x0000000000000000000000000000000000000003"
	require.NoError(t, m.Start(context.Background(), false, nil))
	_, ok := m.GetTokenByChainAddress(1, common.HexToAddress(alias.Address))
	require.True(t, ok)
	// Exercise the documented policy-update protocol without adding a public
	// policy API before application wiring needs one.
	m.mu.Lock()
	policy := tkl.Policy{NativeAliases: []tkl.Identity{alias}, SkippedKeys: []string{types.TokenKey(1, common.HexToAddress(alias.Address))}}
	_, err = m.handle.SetPolicy(policy)
	if err == nil {
		m.policy = policy
		err = m.rebuild()
	}
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
	m, err := New(tkl.Config{Chains: []uint64{1}, InitialLists: []tkl.ListContent{{ID: "test", Body: document, Format: tkl.StandardFormat}}, Policy: tkl.Policy{NativeAliases: []tkl.Identity{{ChainID: 1, Address: "0x0000000000000000000000000000000000000002"}}}}, func(context.Context) (tkl.Bootstrap, error) { return tkl.Bootstrap{}, nil })
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

func BenchmarkMirrorLookup(b *testing.B) {
	m, err := New(tkl.Config{Chains: []uint64{1}}, func(context.Context) (tkl.Bootstrap, error) { return tkl.Bootstrap{}, nil })
	require.NoError(b, err)
	defer func() { _ = m.Stop() }()
	require.NoError(b, m.Start(context.Background(), false, nil))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkToken, _ = m.GetTokenByChainAddress(1, common.Address{})
	}
}

func BenchmarkMirrorRebuild(b *testing.B) {
	tokens := make([]tkl.Token, 10000)
	for i := range tokens {
		tokens[i] = tkl.Token{ChainID: 1, Address: fmt.Sprintf("0x%040x", i+1), Symbol: "TOKEN", Name: "Token", Decimals: 18}
	}
	body, err := json.Marshal(map[string]any{"name": "Benchmark", "version": map[string]int{"major": 1, "minor": 0, "patch": 0}, "tokens": tokens})
	require.NoError(b, err)
	m, err := New(tkl.Config{Chains: []uint64{1}, InitialLists: []tkl.ListContent{{ID: "benchmark", Format: tkl.StandardFormat, Body: string(body)}}}, func(context.Context) (tkl.Bootstrap, error) { return tkl.Bootstrap{}, nil })
	require.NoError(b, err)
	defer func() { _ = m.Stop() }()
	require.NoError(b, m.Start(context.Background(), false, nil))
	require.Len(b, m.UniqueTokens(), 10001)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.mu.Lock()
		m.mirror.Store(nil)
		err = m.rebuild()
		m.mu.Unlock()
		if err != nil {
			b.Fatal(err)
		}
	}
}
