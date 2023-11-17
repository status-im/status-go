package accounts

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/multiaccounts/common"
)

func TestIsOwnAccount(t *testing.T) {
	account := Account{Wallet: true}
	require.True(t, account.IsWalletNonWatchOnlyAccount())

	account = Account{
		Type: AccountTypeGenerated,
	}
	require.True(t, account.IsWalletNonWatchOnlyAccount())

	account = Account{
		Type: AccountTypeKey,
	}
	require.True(t, account.IsWalletNonWatchOnlyAccount())

	account = Account{
		Type: AccountTypeSeed,
	}
	require.True(t, account.IsWalletNonWatchOnlyAccount())

	account = Account{
		Type: AccountTypeWatch,
	}
	require.False(t, account.IsWalletNonWatchOnlyAccount())

	account = Account{}
	require.False(t, account.IsWalletNonWatchOnlyAccount())
}

func TestUnmarshal(t *testing.T) {
	data := `
{
		"key-uid": "0xbc14c321b74652e57c7f26eb30d597ea27cbdf36cba5c85d24f12748153a035e",
    "public-key": "0x0465f6d4f1172524fc057954c8a3f8e34f991558b3d1097189975062f67adda7835da61acb5cda3348b41d211ed0cb07aba668eb12e19e29d98745bebf68d93b61",
    "address": "0xf09c9f5Fb9faa22d0C6C593e7157Ceac8B2b0fe4",
    "colorId": "primary",
    "wallet": true,
		"chat": true,
    "path": "m/44'/60'/0'/0/0",
    "name": "Status account",
		"type": "generated",
		"emoji": "some-emoji",
		"hidden": true,
		"clock": 1234,
		"removed": true,
		"operable": "fully"
}
`
	var account Account
	err := json.Unmarshal([]byte(data), &account)
	require.NoError(t, err)

	require.Equal(t, "0xbc14c321b74652e57c7f26eb30d597ea27cbdf36cba5c85d24f12748153a035e", account.KeyUID)
	require.Equal(t, []byte("0x0465f6d4f1172524fc057954c8a3f8e34f991558b3d1097189975062f67adda7835da61acb5cda3348b41d211ed0cb07aba668eb12e19e29d98745bebf68d93b61"), account.PublicKey.Bytes())
	require.Equal(t, "0xf09c9f5Fb9faa22d0C6C593e7157Ceac8B2b0fe4", account.Address.String())
	require.Equal(t, common.CustomizationColorPrimary, account.ColorID)
	require.Equal(t, true, account.Wallet)
	require.Equal(t, true, account.Chat)
	require.Equal(t, "m/44'/60'/0'/0/0", account.Path)
	require.Equal(t, "Status account", account.Name)
	require.Equal(t, "generated", account.Type.String())
	require.Equal(t, "some-emoji", account.Emoji)
	require.Equal(t, true, account.Hidden)
	require.Equal(t, uint64(1234), account.Clock)
	require.Equal(t, true, account.Removed)
	require.Equal(t, "fully", account.Operable.String())
}
