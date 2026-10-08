//go:build tkl

package tklmanager

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/nim-token-lists/go/tkl"
	"github.com/stretchr/testify/require"

	types "github.com/status-im/status-go/pkg/services/wallet/token/tokenlist"
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

func TestShadowWaitsForDurableCustomRecovery(t *testing.T) {
	for _, operation := range []string{"upsert", "delete"} {
		t.Run(operation, func(t *testing.T) {
			seen := make(chan ShadowSnapshot, 10)
			m, err := New(tkl.Config{Chains: []uint64{1}}, func(context.Context) (tkl.Bootstrap, error) { return tkl.Bootstrap{}, nil }, RefreshOptions{
				Persist:  func(context.Context, []tkl.ListContent) error { return nil },
				OnShadow: func(_ context.Context, s ShadowSnapshot) { seen <- s },
			})
			require.NoError(t, err)
			defer func() { _ = m.Stop() }()
			nextSnapshot := func() ShadowSnapshot {
				select {
				case snapshot := <-seen:
					return snapshot
				case <-time.After(5 * time.Second):
					t.Fatal("shadow snapshot was not published")
					return ShadowSnapshot{}
				}
			}
			require.NoError(t, m.Start(context.Background(), false, nil))
			before := nextSnapshot()
			row := tkl.Token{ChainID: 1, Address: common.HexToAddress("0x1234").Hex(), Symbol: "DURABLE", Decimals: 18, Custom: true}
			key := convertToken(row).Key()
			input := &row
			prepare := func() (tkl.Mutation, error) { return m.handle.CustomValidateUpsert(row) }
			if operation == "delete" {
				require.NoError(t, m.UpsertCustom(context.Background(), convertToken(row), func(context.Context, *types.Token) error { return nil }))
				before = nextSnapshot()
				input = nil
				prepare = func() (tkl.Mutation, error) { return m.handle.CustomValidateDelete(key) }
			}
			writes := 0
			err = m.mutateCustom(context.Background(), key, input, prepare, func(_ context.Context, mutation tkl.Mutation) error {
				writes++
				// SQL succeeds, but the native proposal can no longer be committed.
				_, err := m.handle.CustomAbort(mutation.ID)
				return err
			})
			require.NoError(t, err)
			require.NotNil(t, m.customResync.Load())
			require.Equal(t, before.Sequence, m.shadow.submissions, "do not compare durable SQL inputs against a stale mirror")
			tokens, err := m.GetTokensByKeys([]string{key})
			require.NoError(t, err)
			after := nextSnapshot()
			require.Equal(t, before.Sequence+1, after.Sequence)
			require.Equal(t, m.mirror.Load().revision, after.Revision)
			require.Greater(t, after.Revision, before.Revision)
			if operation == "upsert" {
				require.Len(t, tokens, 1)
				require.Equal(t, []tkl.Token{row}, after.Customs)
				require.Len(t, after.Tokens, 2)
			} else {
				require.Empty(t, tokens)
				require.Empty(t, after.Customs)
				require.Len(t, after.Tokens, 1)
			}
			require.Equal(t, 1, writes, "recovery must not repeat SQL")
		})
	}
}

func TestShadowSkipsUnchangedRefreshes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-None-Match") == `"registry"` {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", `"registry"`)
		fmt.Fprint(w, `{"timestamp":"2026-01-01T00:00:00Z","version":{"major":1,"minor":0,"patch":0},"tokenLists":[]}`)
	}))
	defer server.Close()
	seen := make(chan ShadowSnapshot, shadowMaxSamples+2)
	var writes []tkl.ListContent
	m, err := New(tkl.Config{Chains: []uint64{1}, RegistryID: "registry", RegistryURL: server.URL}, func(context.Context) (tkl.Bootstrap, error) { return tkl.Bootstrap{}, nil }, RefreshOptions{
		Persist:  func(_ context.Context, rows []tkl.ListContent) error { writes = rows; return nil },
		OnShadow: func(_ context.Context, s ShadowSnapshot) { seen <- s },
	})
	require.NoError(t, err)
	defer func() { _ = m.Stop() }()
	require.NoError(t, m.Start(context.Background(), false, nil))
	<-seen
	require.NoError(t, m.TriggerRefresh(context.Background()))
	require.Len(t, writes, 1)
	first := <-seen
	for i := 0; i < shadowMaxSamples; i++ {
		require.NoError(t, m.TriggerRefresh(context.Background()))
		require.Empty(t, writes, "304 checks must not replace stored inputs")
		require.Equal(t, first.Revision, m.mirror.Load().revision)
		require.Equal(t, first.Sequence, m.shadow.submissions, "unchanged checks must not consume comparison samples")
	}
	require.False(t, m.shadow.disabled)
	require.Empty(t, seen)
}

func TestShadowCapturesInputChanges(t *testing.T) {
	seen := make(chan ShadowSnapshot, 10)
	m, err := New(tkl.Config{Chains: []uint64{1}}, func(context.Context) (tkl.Bootstrap, error) { return tkl.Bootstrap{}, nil }, RefreshOptions{
		Persist:  func(context.Context, []tkl.ListContent) error { return nil },
		OnShadow: func(_ context.Context, s ShadowSnapshot) { seen <- s },
	})
	require.NoError(t, err)
	defer func() { _ = m.Stop() }()
	nextSnapshot := func() ShadowSnapshot {
		select {
		case snapshot := <-seen:
			return snapshot
		case <-time.After(5 * time.Second):
			t.Fatal("changed inputs were not captured")
			return ShadowSnapshot{}
		}
	}
	require.NoError(t, m.Start(context.Background(), false, nil))
	first := nextSnapshot()
	// Source changes may not affect any token in the active chains.
	content := tkl.ListContent{ID: "inactive", Body: `{"tokens":[]}`}
	m.mu.Lock()
	m.shadowWrites([]tkl.ListContent{content})
	m.captureShadow()
	m.mu.Unlock()
	updated := nextSnapshot()
	require.Equal(t, first.Revision, updated.Revision)
	require.Equal(t, []tkl.ListContent{content}, updated.Contents)
	// Invalid customs are durable inputs even when the core excludes them.
	row := &types.Token{ChainID: 1, Address: common.HexToAddress("0x1234"), Symbol: "INVALID", Decimals: 24}
	require.NoError(t, m.UpsertCustom(context.Background(), row, func(context.Context, *types.Token) error { return nil }))
	custom := nextSnapshot()
	require.Equal(t, first.Revision, custom.Revision)
	require.Len(t, custom.Customs, 1)
	require.NoError(t, m.UpsertCustom(context.Background(), row, func(context.Context, *types.Token) error { return nil }))
	require.Equal(t, custom.Sequence, m.shadow.submissions, "identical customs need no new comparison")
	require.NoError(t, m.DeleteCustom(context.Background(), row.Key(), func(context.Context) error { return nil }))
	deleted := nextSnapshot()
	require.Equal(t, first.Revision, deleted.Revision)
	require.Empty(t, deleted.Customs)
	require.NoError(t, m.DeleteCustom(context.Background(), row.Key(), func(context.Context) error { return nil }))
	require.Equal(t, deleted.Sequence, m.shadow.submissions, "an absent custom needs no new comparison")
	require.NoError(t, m.SetChains([]uint64{1}))
	require.Equal(t, deleted.Sequence, m.shadow.submissions, "identical chain inputs need no new comparison")
	// Chain-policy changes still need comparison even with the same token count.
	require.NoError(t, m.SetChains([]uint64{10}))
	chains := nextSnapshot()
	require.Greater(t, chains.Revision, first.Revision)
	require.Len(t, chains.Tokens, len(first.Tokens))
	require.Equal(t, []uint64{10}, chains.Config.Chains)
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
		require.NoError(t, m.SetChains([]uint64{uint64(i + 2)}))
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
