package multiaccounts

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*Database, func()) {
	tmpfile, err := ioutil.TempFile("", "accounts-tests-")
	require.NoError(t, err)
	db, err := InitializeDB(tmpfile.Name())
	require.NoError(t, err)
	return db, func() {
		require.NoError(t, db.Close())
		require.NoError(t, os.Remove(tmpfile.Name()))
	}
}

func TestAccounts(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	expected := Account{Name: "string", KeyUID: "string"}
	require.NoError(t, db.SaveAccount(expected))
	accounts, err := db.GetAccounts()
	require.NoError(t, err)
	require.Len(t, accounts, 1)
	require.Equal(t, expected, accounts[0])
}

func TestAccountsUpdate(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	expected := Account{KeyUID: "string"}
	require.NoError(t, db.SaveAccount(expected))
	expected.PhotoPath = "chars"
	require.NoError(t, db.UpdateAccount(expected))
	rst, err := db.GetAccounts()
	require.NoError(t, err)
	require.Len(t, rst, 1)
	require.Equal(t, expected, rst[0])
}

func TestLoginUpdate(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	accounts := []Account{{Name: "first", KeyUID: "0x1"}, {Name: "second", KeyUID: "0x2"}}
	for _, acc := range accounts {
		require.NoError(t, db.SaveAccount(acc))
	}
	require.NoError(t, db.UpdateAccountTimestamp(accounts[0].KeyUID, 100))
	require.NoError(t, db.UpdateAccountTimestamp(accounts[1].KeyUID, 10))
	accounts[0].Timestamp = 100
	accounts[1].Timestamp = 10
	rst, err := db.GetAccounts()
	require.NoError(t, err)
	require.Equal(t, accounts, rst)
}
