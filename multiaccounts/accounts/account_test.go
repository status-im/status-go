package accounts

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsOwnAccount(t *testing.T) {
	account := Account{Wallet: true}
	require.True(t, account.IsOwnAccount())

	account = Account{
		Type: accountTypeGenerated,
	}
	require.True(t, account.IsOwnAccount())

	account = Account{
		Type: accountTypeKey,
	}
	require.True(t, account.IsOwnAccount())

	account = Account{
		Type: accountTypeSeed,
	}
	require.True(t, account.IsOwnAccount())

	account = Account{
		Type: AccountTypeWatch,
	}
	require.False(t, account.IsOwnAccount())

	account = Account{}
	require.False(t, account.IsOwnAccount())
}
