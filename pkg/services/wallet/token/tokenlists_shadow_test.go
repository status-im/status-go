//go:build tkl

package token

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"
	"github.com/status-im/nim-token-lists/go/tkl"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/pkg/services/wallet/token/tklmanager"
	tokentypes "github.com/status-im/status-go/pkg/services/wallet/token/types"
)

func TestShadowEmbeddedParityAndMetadataMismatch(t *testing.T) {
	m, cleanup := setupTestTokenDB(t)
	defer cleanup()
	seen := make(chan tklmanager.ShadowSnapshot, 2)
	facade, err := newTKLReadManager(m, []uint64{1, 10, 56}, time.Time{}, tklmanager.RefreshOptions{Persist: func(context.Context, []tkl.ListContent) error { return nil }, OnShadow: func(_ context.Context, s tklmanager.ShadowSnapshot) { seen <- s }})
	require.NoError(t, err)
	defer func() { _ = facade.Stop() }()
	m.tokensManager = facade
	require.NoError(t, facade.Start(context.Background(), false, nil))
	first := <-seen
	report := compareShadow(context.Background(), first)
	require.Equal(t, "match", report.Status)
	require.Zero(t, report.Differences)
	require.Greater(t, report.SDKTokens, 100)
	first.Tokens[0].Symbol = "BAD"
	report = compareShadow(context.Background(), first)
	require.Equal(t, "mismatch", report.Status)
	require.Equal(t, 1, report.Differences)
	require.NoError(t, m.UpsertCustom(tokentypes.Token{Token: &types.Token{ChainID: 1, Address: common.HexToAddress("0x1234"), Symbol: "CUSTOM", Decimals: 18}}))
	report = compareShadow(context.Background(), <-seen)
	require.Equal(t, "match", report.Status)
	require.Equal(t, 1, report.ExpectedCustomMarkers)
}

func TestShadowNeverCallsCorruptInputAMatch(t *testing.T) {
	s := tklmanager.ShadowSnapshot{Config: tkl.Config{Chains: []uint64{1}, MainListID: "main", InitialLists: []tkl.ListContent{{ID: "main", Body: "broken"}}}}
	report := compareShadow(context.Background(), s)
	require.Equal(t, "source_error", report.Status)
	require.Equal(t, 1, report.ParseErrors)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.Equal(t, "cancelled", compareShadow(ctx, s).Status)
}
