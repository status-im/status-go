//go:build tkl

package token

import (
	"context"
	"github.com/status-im/nim-token-lists/go/tkl"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestTKLPutBatchRollsBackAndPreservesFetchTime(t *testing.T) {
	m, cleanup := setupTestTokenDB(t)
	defer cleanup()
	_, err := m.walletDB.Exec(`CREATE TRIGGER reject_bad BEFORE INSERT ON token_lists WHEN NEW.id = 'bad' BEGIN SELECT RAISE(ABORT, 'injected failure'); END`)
	require.NoError(t, err)
	writes := []tkl.ListContent{{ID: "good", Source: "https://example.org/list", Body: "body", ETag: "etag", FetchedAt: 123}, {ID: "bad", Body: "invalid"}}
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
	require.NoError(t, putTKLBatch(context.Background(), m.walletDB, []tkl.ListContent{{ID: "unstamped", Body: "body"}, {ID: "embedded", Body: "body", FetchedTimestamp: "0001-01-01T00:00:00Z"}, {ID: "epoch", Body: "body", FetchedTimestamp: "1970-01-01T00:00:00Z"}}))
	rows, err = NewContentStore(m.walletDB).GetAll()
	require.NoError(t, err)
	require.True(t, rows["unstamped"].Fetched.IsZero())
	require.True(t, rows["embedded"].Fetched.IsZero())
	require.True(t, rows["epoch"].Fetched.Equal(time.Unix(0, 0)))
}
