//go:build tkl

package tklmanager

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"
	"github.com/status-im/nim-token-lists/go/tkl"
	"github.com/stretchr/testify/require"
)

func TestCustomWritesBeforeStartAndImmediateVisibility(t *testing.T) {
	m, err := New(tkl.Config{Chains: []uint64{1}}, func(context.Context) (tkl.Bootstrap, error) { return tkl.Bootstrap{}, nil })
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
	m, err := New(tkl.Config{Chains: []uint64{1}}, func(context.Context) (tkl.Bootstrap, error) { return tkl.Bootstrap{}, nil })
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
	require.Len(t, notify, 1)
	require.ErrorIs(t, m.DeleteCustom(context.Background(), row.Key(), func(context.Context) error { return failure }), failure)
	_, ok = m.GetTokenByChainAddress(1, row.Address)
	require.True(t, ok)
	require.Len(t, notify, 1)
	require.NoError(t, m.Stop())
	require.ErrorIs(t, m.UpsertCustom(context.Background(), row, persist), tkl.Closed)
	require.Equal(t, 1, calls)
}

func TestCustomCuratedPrecedenceAndCommitAfterCancellation(t *testing.T) {
	m, err := New(tkl.Config{Chains: []uint64{1}}, func(context.Context) (tkl.Bootstrap, error) { return tkl.Bootstrap{}, nil })
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
