package accounts

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func MockTestAccounts(tb testing.TB, db *sql.DB, accounts []*Account) {
	d, err := NewDB(db)
	require.NoError(tb, err)

	err = d.SaveOrUpdateAccounts(accounts, false)
	require.NoError(tb, err)
	res, err := d.GetActiveAccounts()
	require.NoError(tb, err)
	require.Equal(tb, accounts[0].Address, res[0].Address)
}
