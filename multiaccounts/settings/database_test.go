package settings

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/errors"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/sqlite"
	"github.com/status-im/status-go/t/helpers"
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
		DefaultSyncPeriod:         777600,
		UseMailservers:            true,
		LinkPreviewRequestEnabled: true,
		SendStatusUpdates:         true,
		IncludeWatchOnlyAccount:   true,
		WalletRootAddress:         types.HexToAddress("0x3B591fd819F86D0A6a2EF2Bcb94f77807a7De1a6")}
)

func setupTestDB(t *testing.T) (*Database, func()) {
	db, stop, err := helpers.SetupTestSQLDB(appdatabase.DbInitializer{}, "settings-tests-")
	if err != nil {
		require.NoError(t, stop())
	}
	require.NoError(t, err)

	d, err := MakeNewDB(db)
	if err != nil {
		require.NoError(t, stop())
	}
	require.NoError(t, err)

	return d, func() {
		require.NoError(t, stop())
	}
}

func TestClosingsqlDB(t *testing.T) {
	testFileName := "./test.db"
	password := "settings-tests"

	// Make connection with sql.DB
	db, err := appdatabase.InitializeDB(testFileName, password, sqlite.ReducedKDFIterationsNumber)

	// handle removing the test file on any exit
	defer func() {
		require.NoError(t, os.Remove(testFileName))
		require.NoError(t, os.Remove(testFileName+"-shm"))
		require.NoError(t, os.Remove(testFileName+"-wal"))
	}()

	// Then check the first error
	require.NoError(t, err)

	// Init settings.Database struct
	d, err := MakeNewDB(db)
	require.NoError(t, err)

	// Add settings data to the db
	err = d.CreateSettings(settings, config)
	require.NoError(t, err)

	// close the sql.DB connection
	err = db.Close()
	require.NoError(t, err)

	// Make another connection with sql.DB
	db2, err := appdatabase.InitializeDB(testFileName, password, sqlite.ReducedKDFIterationsNumber)
	require.NoError(t, err)

	// Init settings.Database struct using second connection
	d, err = MakeNewDB(db2)
	require.NoError(t, err)

	// Ping db expect no error
	err = d.db.Ping()
	require.NoError(t, err)
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

	require.Equal(t, errors.ErrInvalidConfig, db.SaveSetting("a_column_that_does_n0t_exist", "random value"))
}

func TestDatabase_SetSettingLastSynced(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	tm := uint64(0)

	// Default value should be `0`
	ct, err := db.GetSettingLastSynced(Currency)
	require.NoError(t, err)
	require.Equal(t, tm, ct)

	// Test setting clock value to something greater than `0`
	tm += 123
	err = db.SetSettingLastSynced(Currency, tm)
	require.NoError(t, err)

	ct, err = db.GetSettingLastSynced(Currency)
	require.NoError(t, err)
	require.Equal(t, tm, ct)

	// Test setting clock to greater than `123`
	now := uint64(321)
	err = db.SetSettingLastSynced(Currency, now)
	require.NoError(t, err)

	ct, err = db.GetSettingLastSynced(Currency)
	require.NoError(t, err)
	require.Equal(t, now, ct)

	// Test setting clock to something less than `321`
	earlier := uint64(231)
	err = db.SetSettingLastSynced(Currency, earlier)
	require.NoError(t, err)

	ct, err = db.GetSettingLastSynced(Currency)
	require.NoError(t, err)
	require.Equal(t, now, ct)
}

func TestSyncColumnsSet(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	for _, sf := range SettingFieldRegister {
		if sf.SyncProtobufFactory() != nil {
			_, err := db.GetSettingLastSynced(sf)
			require.NoError(t, err)
		}
	}
}
