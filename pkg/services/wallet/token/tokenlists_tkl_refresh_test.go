package token

import (
	"context"
	"testing"
	"time"

	"github.com/status-im/nim-token-lists/go/tkl"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/pkg/services/wallet/token/tklmanager"
)

func TestTKLPutBatchRollsBackAndPreservesFetchTime(t *testing.T) {
	m, cleanup := setupTestTokenDB(t)
	defer cleanup()
	_, err := m.walletDB.Exec(`CREATE TRIGGER reject_bad BEFORE INSERT ON token_lists WHEN NEW.id = 'bad' BEGIN SELECT RAISE(ABORT, 'injected failure'); END`)
	require.NoError(t, err)
	writes := []tklmanager.Write{{ListContent: tkl.ListContent{ID: "good", Source: "https://example.org/list", ETag: "etag", FetchedAt: 123}, Body: []byte("body")}, {ListContent: tkl.ListContent{ID: "bad"}, Body: []byte("invalid")}}
	require.Error(t, putTKLBatch(context.Background(), m.walletDB, writes))
	rows, err := NewContentStore(m.walletDB).GetAll()
	require.NoError(t, err)
	require.Empty(t, rows)
	require.NoError(t, putTKLBatch(context.Background(), m.walletDB, writes[:1]))
	rows, err = NewContentStore(m.walletDB).GetAll()
	require.NoError(t, err)
	require.Equal(t, "body", string(rows["good"].Data))
	require.Equal(t, "etag", rows["good"].Etag)
	require.True(t, rows["good"].Fetched.Equal(time.Unix(123, 0)))
	body := []byte("body")
	require.NoError(t, putTKLBatch(context.Background(), m.walletDB, []tklmanager.Write{{ListContent: tkl.ListContent{ID: "unstamped"}, Body: body}, {ListContent: tkl.ListContent{ID: "embedded", FetchedTimestamp: "0001-01-01T00:00:00Z"}, Body: body}, {ListContent: tkl.ListContent{ID: "epoch", FetchedTimestamp: "1970-01-01T00:00:00Z"}, Body: body}}))
	rows, err = NewContentStore(m.walletDB).GetAll()
	require.NoError(t, err)
	require.True(t, rows["unstamped"].Fetched.IsZero())
	require.True(t, rows["embedded"].Fetched.IsZero())
	require.True(t, rows["epoch"].Fetched.Equal(time.Unix(0, 0)))
}
