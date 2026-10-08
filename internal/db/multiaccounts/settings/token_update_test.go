package settings

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSaveLastTokensUpdateCancellation(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	require.NoError(t, db.CreateSettings(settings, config))
	when := time.Unix(123, 0).UTC()
	require.NoError(t, db.SaveLastTokensUpdate(context.Background(), when))
	got, err := db.LastTokensUpdate()
	require.NoError(t, err)
	require.True(t, got.Equal(when))
	// Exhaust the pool to reproduce shutdown while a settings write waits for DB.
	db.db.SetMaxOpenConns(1)
	conn, err := db.db.Conn(context.Background())
	require.NoError(t, err)
	defer conn.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	require.ErrorIs(t, db.SaveLastTokensUpdate(ctx, when.Add(time.Second)), context.DeadlineExceeded)
}
