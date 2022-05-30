package accounts

import (
	"encoding/json"
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

func TestUnmarshal(t *testing.T) {
	data := `
{
    "public-key": "0x0465f6d4f1172524fc057954c8a3f8e34f991558b3d1097189975062f67adda7835da61acb5cda3348b41d211ed0cb07aba668eb12e19e29d98745bebf68d93b61",
    "address": "0xf09c9f5Fb9faa22d0C6C593e7157Ceac8B2b0fe4",
    "color": "#4360df",
    "wallet": true,
    "path": "m/44'/60'/0'/0/0",
    "name": "Status account",
    "derived-from": "0x6f015A79890Dcb38eFeC1D83772d57159D2eb58b"
}
`
	var account Account
	err := json.Unmarshal([]byte(data), &account)
	require.NoError(t, err)

	require.Equal(t, []byte("0x0465f6d4f1172524fc057954c8a3f8e34f991558b3d1097189975062f67adda7835da61acb5cda3348b41d211ed0cb07aba668eb12e19e29d98745bebf68d93b61"), account.PublicKey.Bytes())
	require.Equal(t, "0xf09c9f5Fb9faa22d0C6C593e7157Ceac8B2b0fe4", account.Address.String())
	require.Equal(t, "#4360df", account.Color)
	require.Equal(t, true, account.Wallet)
	require.Equal(t, "m/44'/60'/0'/0/0", account.Path)
	require.Equal(t, "Status account", account.Name)
	require.Equal(t, "0x6f015A79890Dcb38eFeC1D83772d57159D2eb58b", account.DerivedFrom)
}
