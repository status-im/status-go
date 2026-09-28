//go:build tkl

package tklmanager

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"
	"github.com/status-im/nim-token-lists/go/tkl"
	"github.com/stretchr/testify/require"
)

func BenchmarkShadowCapture(b *testing.B) {
	m := &Manager{started: true, shadow: &shadowState{started: time.Now(), queue: make(chan ShadowSnapshot, 1)}}
	current := &snapshot{revision: 1, tokens: make([]*types.Token, 10000)}
	for i := range current.tokens {
		current.tokens[i] = &types.Token{ChainID: 1, Symbol: "TOKEN", Name: "Token", Decimals: 18}
	}
	m.mirror.Store(current)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.mu.Lock()
		m.shadow.submissions = 0
		m.captureShadow()
		m.mu.Unlock()
	}
}

func TestShadowCapturesCommittedRevisionAndOwnsValues(t *testing.T) {
	seen := make(chan ShadowSnapshot, 10)
	m, err := New(tkl.Config{Chains: []uint64{1}}, func(context.Context) (tkl.Bootstrap, error) { return tkl.Bootstrap{}, nil }, RefreshOptions{Persist: func(context.Context, []tkl.ListContent) error { return nil }, OnShadow: func(_ context.Context, s ShadowSnapshot) { seen <- s }})
	require.NoError(t, err)
	defer func() { _ = m.Stop() }()
	require.NoError(t, m.Start(context.Background(), false, nil))
	first := <-seen
	require.Equal(t, uint64(1), first.Revision)
	require.Len(t, first.Tokens, 1)
	first.Tokens[0].Symbol = "mutated"
	first.Config.Chains[0] = 99
	native, _ := m.GetTokenByChainAddress(1, common.Address{})
	require.Equal(t, "ETH", native.Symbol)
	row := &types.Token{ChainID: 1, Address: common.HexToAddress("0x1234"), Symbol: "CUSTOM", Decimals: 18}
	require.Error(t, m.UpsertCustom(context.Background(), row, func(context.Context, *types.Token) error { return errors.New("SQL failed") }))
	require.Empty(t, seen)
	require.NoError(t, m.UpsertCustom(context.Background(), row, func(context.Context, *types.Token) error { return nil }))
	next := <-seen
	require.Len(t, next.Customs, 1)
	require.Equal(t, []uint64{1}, next.Config.Chains)
	require.Len(t, next.Tokens, 2)
	require.Greater(t, next.Revision, first.Revision)
	require.NoError(t, m.SetChains([]uint64{10}))
	chains := <-seen
	require.Equal(t, []uint64{10}, chains.Config.Chains)
	require.Len(t, chains.Customs, 1)
}

func TestShadowCoalescesAndCapsSubmissions(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	seen := make(chan ShadowSnapshot, 2)
	first := true
	m, err := New(tkl.Config{Chains: []uint64{1}}, func(context.Context) (tkl.Bootstrap, error) { return tkl.Bootstrap{}, nil }, RefreshOptions{Persist: func(context.Context, []tkl.ListContent) error { return nil }, OnShadow: func(ctx context.Context, s ShadowSnapshot) {
		if first {
			first = false
			close(entered)
			select {
			case <-release:
			case <-ctx.Done():
				return
			}
		}
		seen <- s
	}})
	require.NoError(t, err)
	defer func() { _ = m.Stop() }()
	require.NoError(t, m.Start(context.Background(), false, nil))
	<-entered
	for i := 0; i < shadowMaxSamples; i++ {
		require.NoError(t, m.SetChains([]uint64{uint64(i + 1)}))
	}
	close(release)
	<-seen
	last := <-seen
	require.Equal(t, "sample_limit", last.Skipped)
	require.Greater(t, last.Coalesced, 0)
	require.True(t, m.shadow.disabled)
	require.Nil(t, m.shadow.contents)
}

func TestShadowByteLimitDoesNotBreakCatalogue(t *testing.T) {
	seen := make(chan ShadowSnapshot, 2)
	m, err := New(tkl.Config{Chains: []uint64{1}}, func(context.Context) (tkl.Bootstrap, error) { return tkl.Bootstrap{}, nil }, RefreshOptions{Persist: func(context.Context, []tkl.ListContent) error { return nil }, OnShadow: func(_ context.Context, s ShadowSnapshot) { seen <- s }})
	require.NoError(t, err)
	defer func() { _ = m.Stop() }()
	require.NoError(t, m.Start(context.Background(), false, nil))
	<-seen
	m.mu.Lock()
	m.shadow.bytes = shadowMaxBytes + 1
	m.mu.Unlock()
	require.NoError(t, m.SetChains([]uint64{10}))
	require.Equal(t, "input_byte_limit", (<-seen).Skipped)
	_, ok := m.GetTokenByChainAddress(10, common.Address{})
	require.True(t, ok)
}

func TestShadowStopCancelsAndDrainsWorker(t *testing.T) {
	entered := make(chan struct{})
	exited := make(chan struct{})
	m, err := New(tkl.Config{Chains: []uint64{1}}, func(context.Context) (tkl.Bootstrap, error) { return tkl.Bootstrap{}, nil }, RefreshOptions{Persist: func(context.Context, []tkl.ListContent) error { return nil }, OnShadow: func(ctx context.Context, _ ShadowSnapshot) { close(entered); <-ctx.Done(); close(exited) }})
	require.NoError(t, err)
	require.NoError(t, m.Start(context.Background(), false, nil))
	<-entered
	require.NoError(t, m.Stop())
	select {
	case <-exited:
	default:
		t.Fatal("Stop returned before observer drain")
	}
}

func TestShadowBudgetDisablesWithoutAffectingReads(t *testing.T) {
	seen := make(chan ShadowSnapshot, 2)
	m, err := New(tkl.Config{Chains: []uint64{1}}, func(context.Context) (tkl.Bootstrap, error) { return tkl.Bootstrap{}, nil }, RefreshOptions{Persist: func(context.Context, []tkl.ListContent) error { return nil }, OnShadow: func(_ context.Context, s ShadowSnapshot) { seen <- s }})
	require.NoError(t, err)
	defer func() { _ = m.Stop() }()
	require.NoError(t, m.Start(context.Background(), false, nil))
	<-seen
	m.mu.Lock()
	m.shadow.started = time.Now().Add(-25 * time.Hour)
	m.mu.Unlock()
	require.NoError(t, m.SetChains([]uint64{10}))
	require.Equal(t, "window_expired", (<-seen).Skipped)
	native, ok := m.GetTokenByChainAddress(10, common.Address{})
	require.True(t, ok)
	require.Equal(t, "ETH", native.Symbol)
}
