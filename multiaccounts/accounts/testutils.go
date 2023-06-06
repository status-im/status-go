package accounts

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func AddTestAccounts(t *testing.T, db *sql.DB, accounts []*Account) {
	d, err := NewDB(db)
	require.NoError(t, err)

	err = d.SaveOrUpdateAccounts(accounts)
	require.NoError(t, err)
	res, err := d.GetAccounts()
	require.NoError(t, err)
	require.Equal(t, accounts[0].Address, res[0].Address)
}
