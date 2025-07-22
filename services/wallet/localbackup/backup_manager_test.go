package localbackup

import (
	"testing"

	"github.com/ethereum/go-ethereum/event"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/t/helpers"

	"github.com/stretchr/testify/require"
)

type testState struct {
	backupManager *BackupManager
}

func setupTestService(tb testing.TB) (state testState) {
	appDB, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(tb, err)
	accountsDB, err := accounts.NewDB(appDB)
	require.NoError(tb, err)

	state.backupManager = NewBackupManager(accountsDB, &event.Feed{})

	tb.Cleanup(func() {
		require.NoError(tb, appDB.Close())
	})

	return state
}

func TestService_ExportAndImport(t *testing.T) {
	state1 := setupTestService(t)
	state2 := setupTestService(t)

	// Create watch-only accounts
	woAccounts := accounts.GetWatchOnlyAccountsForTest()
	err := state1.backupManager.accountsDB.SaveOrUpdateAccounts(woAccounts, false)
	require.NoError(t, err)

	// Sanity check: ensure state2 has no accounts initially
	ogWoAccounts, err := state2.backupManager.accountsDB.GetAllWatchOnlyAccounts()
	require.NoError(t, err)
	require.Empty(t, ogWoAccounts)

	// Backup
	marshalledBackup, err := state1.backupManager.ExportBackup()
	require.NoError(t, err)
	require.NotEmpty(t, marshalledBackup)

	// Import the backup file and process it
	err = state2.backupManager.ImportBackup(marshalledBackup)
	require.NoError(t, err)

	// Check if the accounts are imported correctly
	importedAccounts, err := state2.backupManager.accountsDB.GetAllWatchOnlyAccounts()
	require.NoError(t, err)
	require.Len(t, importedAccounts, len(woAccounts))
}
