package tklmanager

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/nim-token-lists/go/tkl"
	"github.com/stretchr/testify/require"

	types "github.com/status-im/status-go/pkg/services/wallet/token/tokenlist"
)

func TestCustomWritesBeforeStartAndImmediateVisibility(t *testing.T) {
	m, err := New(tkl.Config{Chains: []uint64{1}}, noStored)
	require.NoError(t, err)
	defer func() { _ = m.Stop() }()
	row := &types.Token{ChainID: 1, Address: common.HexToAddress("0x1234"), Symbol: "CUSTOM", Decimals: 18}
	writes := 0
	persist := func(ctx context.Context, token *types.Token) error {
		writes++
		require.Equal(t, row.Address, token.Address)
		return nil
	}
	require.NoError(t, m.UpsertCustom(context.Background(), row, persist))
	require.Equal(t, 1, writes)
	require.NoError(t, m.Start(context.Background(), false, nil))
	found, ok := m.GetTokenByChainAddress(1, row.Address)
	require.True(t, ok)
	require.Equal(t, "CUSTOM", found.Symbol)
	row.Symbol = "UPDATED"
	require.NoError(t, m.UpsertCustom(context.Background(), row, persist))
	found, ok = m.GetTokenByChainAddress(1, row.Address)
	require.True(t, ok)
	require.Equal(t, "UPDATED", found.Symbol)
	require.NoError(t, m.DeleteCustom(context.Background(), row.Key(), func(context.Context) error { return nil }))
	_, ok = m.GetTokenByChainAddress(1, row.Address)
	require.False(t, ok)
	// SQL delete remains idempotent even when the core has no such custom.
	require.NoError(t, m.DeleteCustom(context.Background(), row.Key(), func(context.Context) error { return nil }))
}

func TestCustomPersistenceFailureAndValidation(t *testing.T) {
	m, err := New(tkl.Config{Chains: []uint64{1}}, noStored)
	require.NoError(t, err)
	defer func() { _ = m.Stop() }()
	notify := make(chan struct{}, 10)
	require.NoError(t, m.Start(context.Background(), false, notify))
	row := &types.Token{ChainID: 1, Address: common.HexToAddress("0x1234"), Symbol: "CUSTOM", Decimals: 18}
	failure := errors.New("SQL failed")
	require.ErrorIs(t, m.UpsertCustom(context.Background(), row, func(context.Context, *types.Token) error { return failure }), failure)
	_, ok := m.GetTokenByChainAddress(1, row.Address)
	require.False(t, ok)
	require.Empty(t, notify)
	calls := 0
	persist := func(context.Context, *types.Token) error { calls++; return nil }
	row.Decimals = 256
	require.Error(t, m.UpsertCustom(context.Background(), row, persist))
	require.Zero(t, calls)
	row.Decimals = 18
	require.NoError(t, m.UpsertCustom(context.Background(), row, persist))
	require.Empty(t, notify)
	require.ErrorIs(t, m.DeleteCustom(context.Background(), row.Key(), func(context.Context) error { return failure }), failure)
	_, ok = m.GetTokenByChainAddress(1, row.Address)
	require.True(t, ok)
	require.Empty(t, notify)
	require.NoError(t, m.Stop())
	require.ErrorIs(t, m.UpsertCustom(context.Background(), row, persist), tkl.Closed)
	require.Equal(t, 1, calls)
}

func TestCustomCuratedPrecedenceAndCommitAfterCancellation(t *testing.T) {
	m, err := New(tkl.Config{Chains: []uint64{1}}, noStored)
	require.NoError(t, err)
	defer func() { _ = m.Stop() }()
	require.NoError(t, m.Start(context.Background(), false, nil))
	ctx, cancel := context.WithCancel(context.Background())
	row := &types.Token{ChainID: 1, Address: common.HexToAddress("0x1234"), Symbol: "CUSTOM", Decimals: 18}
	require.NoError(t, m.UpsertCustom(ctx, row, func(context.Context, *types.Token) error { cancel(); return nil }))
	_, ok := m.GetTokenByChainAddress(1, row.Address)
	require.True(t, ok, "durable SQL must be followed by publication despite cancellation")
	row.Address = common.Address{}
	row.Symbol = "OVERRIDE"
	require.NoError(t, m.UpsertCustom(context.Background(), row, func(context.Context, *types.Token) error { return nil }))
	native, ok := m.GetTokenByChainAddress(1, common.Address{})
	require.True(t, ok)
	require.Equal(t, "ETH", native.Symbol)
}

func TestCustomWritesDoNotNotifyRefreshObservers(t *testing.T) {
	signals := 0
	m, err := New(tkl.Config{Chains: []uint64{1}}, noStored, RefreshOptions{
		Persist:  func(context.Context, []Write) error { return nil },
		OnChange: func(tkl.Change) { signals++ },
	})
	require.NoError(t, err)
	defer func() { _ = m.Stop() }()
	require.NoError(t, m.Start(context.Background(), false, nil))
	for i := 1; i <= 20; i++ {
		row := &types.Token{ChainID: 1, Address: common.HexToAddress(fmt.Sprintf("0x%x", i)), Symbol: "CUSTOM", Decimals: 18}
		require.NoError(t, m.UpsertCustom(context.Background(), row, func(context.Context, *types.Token) error { return nil }))
		require.NoError(t, m.DeleteCustom(context.Background(), row.Key(), func(context.Context) error { return nil }))
	}
	require.Zero(t, signals)
	require.NoError(t, m.SetChains([]uint64{10}))
	require.Equal(t, 1, signals, "chain changes must still notify")
}

func TestDurableCustomWriteRecoversOnRead(t *testing.T) {
	readers := map[string]func(*testing.T, *Manager, *types.Token){
		"address": func(t *testing.T, m *Manager, row *types.Token) {
			found, ok := m.GetTokenByChainAddress(1, row.Address)
			require.True(t, ok)
			require.Equal(t, row.Symbol, found.Symbol)
		},
		"keys": func(t *testing.T, m *Manager, row *types.Token) {
			found, err := m.GetTokensByKeys([]string{row.Key()})
			require.NoError(t, err)
			require.Len(t, found, 1)
		},
		"batch": func(t *testing.T, m *Manager, row *types.Token) {
			found := m.GetTokensByChainAddresses([]ChainAddress{{ChainID: 1, Address: row.Address}})
			require.NotNil(t, found[0])
		},
		"all":   func(t *testing.T, m *Manager, _ *types.Token) { require.Len(t, m.UniqueTokens(), 2) },
		"chain": func(t *testing.T, m *Manager, _ *types.Token) { require.Len(t, m.GetTokensByChain(1), 2) },
		"lists": func(t *testing.T, m *Manager, _ *types.Token) {
			lists := m.TokenLists()
			require.Len(t, lists, 2)
			require.Len(t, lists[1].Tokens, 1)
		},
		"list": func(t *testing.T, m *Manager, _ *types.Token) {
			list, ok := m.TokenList("custom")
			require.True(t, ok)
			require.Len(t, list.Tokens, 1)
		},
	}
	for name, read := range readers {
		t.Run(name, func(t *testing.T) {
			m, err := New(tkl.Config{Chains: []uint64{1}}, noStored)
			require.NoError(t, err)
			defer func() { _ = m.Stop() }()
			require.NoError(t, m.Start(context.Background(), false, nil))
			row := &types.Token{ChainID: 1, Address: common.HexToAddress("0x1234"), Symbol: "DURABLE", Decimals: 18}
			writes := 0
			token := tkl.Token{ChainID: row.ChainID, Address: row.Address.Hex(), Symbol: row.Symbol, Decimals: 18}
			err = m.mutateCustom(context.Background(), func() (tkl.Mutation, error) {
				return m.handle.CustomValidateUpsert(token)
			}, func(_ context.Context, mutation tkl.Mutation) error {
				writes++
				// Inject a native commit failure after the durable write, without
				// adding failure hooks to the production manager.
				_, abortErr := m.handle.CustomAbort(mutation.ID)
				require.NoError(t, abortErr)
				return nil
			})
			require.NoError(t, err, "a durable SQL write must not be reported as failed")
			read(t, m, row)
			require.Equal(t, 1, writes, "recovery must not repeat the SQL write")
		})
	}
}

func TestCustomMirrorConcurrentReads(t *testing.T) {
	m, err := New(tkl.Config{Chains: []uint64{1}}, noStored)
	require.NoError(t, err)
	defer func() { _ = m.Stop() }()
	require.NoError(t, m.Start(context.Background(), false, nil))
	var readers sync.WaitGroup
	for i := 0; i < 4; i++ {
		readers.Add(1)
		go func() {
			defer readers.Done()
			for j := 0; j < 100; j++ {
				m.UniqueTokens()
				m.GetTokensByChain(1)
				m.GetTokenByChainAddress(1, common.HexToAddress("0x1234"))
			}
		}()
	}
	defer readers.Wait()
	for i := 0; i < 10; i++ {
		row := &types.Token{ChainID: 1, Address: common.HexToAddress("0x1234"), Symbol: fmt.Sprint(i), Decimals: 18}
		require.NoError(t, m.UpsertCustom(context.Background(), row, func(context.Context, *types.Token) error { return nil }))
		require.NoError(t, m.DeleteCustom(context.Background(), row.Key(), func(context.Context) error { return nil }))
	}
}

func TestCustomMirrorMatchesCorePolicy(t *testing.T) {
	const body = `{"name":"Curated","version":{"major":1,"minor":0,"patch":0},"tokens":[{"chainId":1,"address":"0x0000000000000000000000000000000000001234","symbol":"CURATED","decimals":18}]}`
	for _, policy := range []tkl.Policy{
		{},
		{Priority: "CustomFirstPriority"},
		{SkippedKeys: []string{"1-0x0000000000000000000000000000000000001234"}},
		{NativeAliases: []tkl.Identity{{ChainID: 1, Address: "0x0000000000000000000000000000000000001234"}}},
	} {
		t.Run(fmt.Sprint(policy), func(t *testing.T) {
			m, err := New(tkl.Config{Chains: []uint64{1}, Policy: policy, InitialLists: []tkl.ListContent{{ID: "curated", Format: tkl.StandardFormat}}}, withBodies(tkl.ListBody{ID: "curated", Origin: tkl.Bundled, Data: []byte(body)}))
			require.NoError(t, err)
			defer func() { _ = m.Stop() }()
			require.NoError(t, m.Start(context.Background(), false, nil))
			for _, addr := range []string{"0x1234", "0x5678"} {
				row := &types.Token{ChainID: 1, Address: common.HexToAddress(addr), Symbol: "CUSTOM", Decimals: 18}
				for _, remove := range []bool{false, true} {
					if remove {
						require.NoError(t, m.DeleteCustom(context.Background(), row.Key(), func(context.Context) error { return nil }))
					} else {
						require.NoError(t, m.UpsertCustom(context.Background(), row, func(context.Context, *types.Token) error { return nil }))
					}
					page, err := m.handle.GetAll(0, 0)
					require.NoError(t, err)
					want := make([]*types.Token, 0, len(page.Items))
					for _, token := range page.Items {
						want = append(want, convertToken(token))
					}
					require.Equal(t, want, m.UniqueTokens())
					require.Equal(t, want, m.GetTokensByChain(1))
				}
			}
		})
	}
}

func TestDurableCustomRecoveryBeforeChainChange(t *testing.T) {
	for _, start := range []bool{false, true} {
		t.Run(fmt.Sprint(start), func(t *testing.T) {
			m, err := New(tkl.Config{Chains: []uint64{1}}, noStored)
			require.NoError(t, err)
			defer func() { _ = m.Stop() }()
			if start {
				require.NoError(t, m.Start(context.Background(), false, nil))
			}
			row := tkl.Token{ChainID: 1, Address: common.HexToAddress("0x1234").Hex(), Symbol: "DURABLE", Decimals: 18}
			err = m.mutateCustom(context.Background(), func() (tkl.Mutation, error) { return m.handle.CustomValidateUpsert(row) }, func(_ context.Context, mutation tkl.Mutation) error {
				_, err := m.handle.CustomAbort(mutation.ID)
				return err
			})
			require.NoError(t, err)
			require.NoError(t, m.SetChains([]uint64{10}))
			require.NoError(t, m.Start(context.Background(), false, nil))
			require.NoError(t, m.SetChains([]uint64{1}))
			_, ok := m.GetTokenByChainAddress(1, common.HexToAddress(row.Address))
			require.True(t, ok)
			// Failed publication of a durable delete must recover as well.
			err = m.mutateCustom(context.Background(), func() (tkl.Mutation, error) {
				return m.handle.CustomValidateDelete(types.TokenKey(1, common.HexToAddress(row.Address)))
			}, func(_ context.Context, mutation tkl.Mutation) error {
				_, err := m.handle.CustomAbort(mutation.ID)
				return err
			})
			require.NoError(t, err)
			_, ok = m.GetTokenByChainAddress(1, common.HexToAddress(row.Address))
			require.False(t, ok)
		})
	}
}
