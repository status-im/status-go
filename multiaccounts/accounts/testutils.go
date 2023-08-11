package accounts

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func MockTestAccounts(tb testing.TB, d *Database, accounts []*Account) {
	err := d.SaveOrUpdateAccounts(accounts, false)
	require.NoError(tb, err)
	res, err := d.GetActiveAccounts()
	require.NoError(tb, err)
	require.Equal(tb, accounts[0].Address, res[0].Address)
}
