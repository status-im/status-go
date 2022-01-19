package accounts

import (
	"database/sql"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
)

var (
	config = params.NodeConfig{
		NetworkID: 10,
		DataDir:   "test",
	}

	networks = json.RawMessage("{}")
	settings = Settings{
		Address:                   types.HexToAddress("0xdC540f3745Ff2964AFC1171a5A0DD726d1F6B472"),
		AnonMetricsShouldSend:     false,
		CurrentNetwork:            "mainnet_rpc",
		DappsAddress:              types.HexToAddress("0xD1300f99fDF7346986CbC766903245087394ecd0"),
		InstallationID:            "d3efcff6-cffa-560e-a547-21d3858cbc51",
		KeyUID:                    "0x4e8129f3edfc004875be17bf468a784098a9f69b53c095be1f52deff286935ab",
		BackupEnabled:             true,
		LatestDerivedPath:         0,
		Name:                      "Jittery Cornflowerblue Kingbird",
		Networks:                  &networks,
		PhotoPath:                 "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAADIAAAAyCAIAAACRXR/mAAAAjklEQVR4nOzXwQmFMBAAUZXUYh32ZB32ZB02sxYQQSZGsod55/91WFgSS0RM+SyjA56ZRZhFmEWYRRT6h+M6G16zrxv6fdJpmUWYRbxsYr13dKfanpN0WmYRZhGzXz6AWYRZRIfbaX26fT9Jk07LLMIsosPt9I/dTDotswizCG+nhFmEWYRZhFnEHQAA///z1CFkYamgfQAAAABJRU5ErkJggg==",
		PreviewPrivacy:            false,
		PublicKey:                 "0x04211fe0f69772ecf7eb0b5bfc7678672508a9fb01f2d699096f0d59ef7fe1a0cb1e648a80190db1c0f5f088872444d846f2956d0bd84069f3f9f69335af852ac0",
		SigningPhrase:             "yurt joey vibe",
		SendPushNotifications:     true,
		ProfilePicturesShowTo:     ProfilePicturesShowToContactsOnly,
		ProfilePicturesVisibility: ProfilePicturesVisibilityContactsOnly,
		DefaultSyncPeriod:         86400,
		UseMailservers:            true,
		LinkPreviewRequestEnabled: true,
		SendStatusUpdates:         true,
		WalletRootAddress:         types.HexToAddress("0x3B591fd819F86D0A6a2EF2Bcb94f77807a7De1a6")}
)

func setupTestDB(t *testing.T) (*Database, func()) {
	db, stop, err := appdatabase.SetupTestSQLDB("settings-tests-")
	if err != nil {
		require.NoError(t, stop())
	}
	require.NoError(t, err)

	d, err := NewDB(db)
	if err != nil {
		require.NoError(t, stop())
	}
	require.NoError(t, err)

	return d, func() {
		require.NoError(t, stop())
	}
}

func TestNewDB(t *testing.T) {
	// TODO test that
	//  - multiple different in memory dbs can be inited
	//  - only one instance per file name can be inited
}

func TestCreateSettings(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	require.NoError(t, db.CreateSettings(settings, config))

	s, err := db.GetSettings()
	require.NoError(t, err)
	require.Equal(t, settings, s)
}

func TestSaveSetting(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	require.NoError(t, db.CreateSettings(settings, config))
	require.NoError(t, db.SaveSetting(Currency.GetReactName(), "usd"))

	_, err := db.GetSettings()
	require.NoError(t, err)

	require.Equal(t, ErrInvalidConfig, db.SaveSetting("a_column_that_does_n0t_exist", "random value"))
}

func TestSaveAccounts(t *testing.T) {
	type testCase struct {
		description string
		accounts    []Account
		err         error
	}
	for _, tc := range []testCase{
		{
			description: "NoError",
			accounts: []Account{
				{Address: types.Address{0x01}, Chat: true, Wallet: true},
				{Address: types.Address{0x02}},
			},
		},
		{
			description: "UniqueChat",
			accounts: []Account{
				{Address: types.Address{0x01}, Chat: true},
				{Address: types.Address{0x02}, Chat: true},
			},
			err: ErrChatNotUnique,
		},
		{
			description: "UniqueWallet",
			accounts: []Account{
				{Address: types.Address{0x01}, Wallet: true},
				{Address: types.Address{0x02}, Wallet: true},
			},
			err: ErrWalletNotUnique,
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			db, stop := setupTestDB(t)
			defer stop()
			require.Equal(t, tc.err, db.SaveAccounts(tc.accounts))
		})
	}
}

func TestUpdateAccounts(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	accounts := []Account{
		{Address: types.Address{0x01}, Chat: true, Wallet: true},
		{Address: types.Address{0x02}},
	}
	require.NoError(t, db.SaveAccounts(accounts))
	accounts[0].Chat = false
	accounts[1].Chat = true
	require.NoError(t, db.SaveAccounts(accounts))
	rst, err := db.GetAccounts()
	require.NoError(t, err)
	require.Equal(t, accounts, rst)
}

func TestDeleteAccount(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	accounts := []Account{
		{Address: types.Address{0x01}, Chat: true, Wallet: true},
	}
	require.NoError(t, db.SaveAccounts(accounts))
	rst, err := db.GetAccounts()
	require.NoError(t, err)
	require.Equal(t, 1, len(rst))
	require.NoError(t, db.DeleteAccount(types.Address{0x01}))
	rst2, err := db.GetAccounts()
	require.NoError(t, err)
	require.Equal(t, 0, len(rst2))
}

func TestGetAddresses(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	accounts := []Account{
		{Address: types.Address{0x01}, Chat: true, Wallet: true},
		{Address: types.Address{0x02}},
	}
	require.NoError(t, db.SaveAccounts(accounts))
	addresses, err := db.GetAddresses()
	require.NoError(t, err)
	require.Equal(t, []types.Address{{0x01}, {0x02}}, addresses)
}

func TestGetWalletAddress(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	address := types.Address{0x01}
	_, err := db.GetWalletAddress()
	require.Equal(t, err, sql.ErrNoRows)
	require.NoError(t, db.SaveAccounts([]Account{{Address: address, Wallet: true}}))
	wallet, err := db.GetWalletAddress()
	require.NoError(t, err)
	require.Equal(t, address, wallet)
}

func TestGetChatAddress(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	address := types.Address{0x01}
	_, err := db.GetChatAddress()
	require.Equal(t, err, sql.ErrNoRows)
	require.NoError(t, db.SaveAccounts([]Account{{Address: address, Chat: true}}))
	chat, err := db.GetChatAddress()
	require.NoError(t, err)
	require.Equal(t, address, chat)
}

func TestGetAccounts(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	accounts := []Account{
		{Address: types.Address{0x01}, Chat: true, Wallet: true},
		{Address: types.Address{0x02}, PublicKey: types.HexBytes{0x01, 0x02}},
		{Address: types.Address{0x03}, PublicKey: types.HexBytes{0x02, 0x03}},
	}
	require.NoError(t, db.SaveAccounts(accounts))
	rst, err := db.GetAccounts()
	require.NoError(t, err)
	require.Equal(t, accounts, rst)
}

func TestGetAccountByAddress(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	address := types.Address{0x01}
	account := Account{Address: address, Chat: true, Wallet: true}
	dilute := []Account{
		{Address: types.Address{0x02}, PublicKey: types.HexBytes{0x01, 0x02}},
		{Address: types.Address{0x03}, PublicKey: types.HexBytes{0x02, 0x03}},
	}

	accounts := append(dilute, account)

	require.NoError(t, db.SaveAccounts(accounts))
	rst, err := db.GetAccountByAddress(address)
	require.NoError(t, err)
	require.Equal(t, &account, rst)
}

func TestAddressExists(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	accounts := []Account{
		{Address: types.Address{0x01}, Chat: true, Wallet: true},
	}
	require.NoError(t, db.SaveAccounts(accounts))

	exists, err := db.AddressExists(accounts[0].Address)
	require.NoError(t, err)
	require.True(t, exists)
}

func TestAddressDoesntExist(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	exists, err := db.AddressExists(types.Address{1, 1, 1})
	require.NoError(t, err)
	require.False(t, exists)
}

func TestDatabase_SetSettingLastSynced(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	tm := uint64(0)

	// Default value should be `0`
	ct, err := db.GetSettingLastSynced(Currency.GetDBName())
	require.NoError(t, err)
	require.Equal(t, tm, ct)

	// Test setting clock value to something greater than `0`
	tm += 123
	err = db.SetSettingLastSynced(Currency.GetDBName(), tm)
	require.NoError(t, err)

	ct, err = db.GetSettingLastSynced(Currency.GetDBName())
	require.NoError(t, err)
	require.Equal(t, tm, ct)

	// Test setting clock to greater than `123`
	now := uint64(321)
	err = db.SetSettingLastSynced(Currency.GetDBName(), now)
	require.NoError(t, err)

	ct, err = db.GetSettingLastSynced(Currency.GetDBName())
	require.NoError(t, err)
	require.Equal(t, now, ct)

	// Test setting clock to something less than `321`
	earlier := uint64(231)
	err = db.SetSettingLastSynced(Currency.GetDBName(), earlier)
	require.NoError(t, err)

	ct, err = db.GetSettingLastSynced(Currency.GetDBName())
	require.NoError(t, err)
	require.Equal(t, now, ct)
}
