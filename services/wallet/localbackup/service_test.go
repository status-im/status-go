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
	service *Service
	close   func()
}

func setupTestService(tb testing.TB) (state testState) {
	appDB, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(tb, err)
	accountsDB, err := accounts.NewDB(appDB)
	require.NoError(tb, err)

	state.service = NewService(accountsDB, &event.Feed{})

	state.close = func() {
		require.NoError(tb, appDB.Close())
	}

	return state
}

func TestService_ExportAndImport(t *testing.T) {
	state1 := setupTestService(t)
	state2 := setupTestService(t)

	defer state1.close()
	defer state2.close()

	// Create watch-only accounts
	woAccounts := accounts.GetWatchOnlyAccountsForTest()
	err := state1.service.accountsDB.SaveOrUpdateAccounts(woAccounts, false)
	require.NoError(t, err)

	// Sanity check: ensure state2 has no accounts initially
	ogWoAccounts, err := state2.service.accountsDB.GetAllWatchOnlyAccounts()
	require.NoError(t, err)
	require.Empty(t, ogWoAccounts)

	// Backup
	marshalledBackup, err := state1.service.ExportBackup()
	require.NoError(t, err)
	require.NotEmpty(t, marshalledBackup)

	// Import the backup file and process it
	err = state2.service.ImportBackup(marshalledBackup)
	require.NoError(t, err)

	// Check if the accounts are imported correctly
	importedAccounts, err := state2.service.accountsDB.GetAllWatchOnlyAccounts()
	require.NoError(t, err)
	require.Len(t, importedAccounts, len(woAccounts))
}
