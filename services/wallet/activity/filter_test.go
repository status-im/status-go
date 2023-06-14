package activity

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/services/wallet/transfer"

	"github.com/stretchr/testify/require"
)

func setupTestFilterDB(t *testing.T) (db *sql.DB, close func()) {
	db, err := appdatabase.SetupTestMemorySQLDB("wallet-activity-tests-filter")
	require.NoError(t, err)

	return db, func() {
		require.NoError(t, db.Close())
	}
}

func TestGetRecipientsEmptyDB(t *testing.T) {
	db, close := setupTestFilterDB(t)
	defer close()

	entries, hasMore, err := GetRecipients(context.Background(), db, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 0, len(entries))
	require.False(t, hasMore)
}

func TestGetRecipients(t *testing.T) {
	db, close := setupTestFilterDB(t)
	defer close()

	// Add 6 extractable transactions
	trs, _, toTrs := transfer.GenerateTestTransfers(t, db, 0, 6)
	for i := range trs {
		transfer.InsertTestTransfer(t, db, &trs[i])
	}

	entries, hasMore, err := GetRecipients(context.Background(), db, 0, 15)
	require.NoError(t, err)
	require.False(t, hasMore)
	require.Equal(t, 6, len(entries))
	for i := range entries {
		found := false
		for j := range toTrs {
			if entries[i] == toTrs[j] {
				found = true
				break
			}
		}
		require.True(t, found, fmt.Sprintf("recipient %s not found in toTrs", entries[i].Hex()))
	}

	entries, hasMore, err = GetRecipients(context.Background(), db, 0, 4)
	require.NoError(t, err)
	require.Equal(t, 4, len(entries))
	require.True(t, hasMore)
}
