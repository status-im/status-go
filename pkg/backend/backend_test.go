package backend

import (
	"fmt"
	"path"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/pkg/backend/requests"
	"github.com/status-im/status-go/pkg/testutils"
	"github.com/status-im/status-go/sqlite"
)

func TestNewDataDir(t *testing.T) {
	b, err := NewStatusBackend(
		t.TempDir(),
		WithLogger(testutils.MustCreateTestLogger().Named("backend")),
	)
	require.NoError(t, err)
	require.NotNil(t, b)

	// No accounts yet
	accs, err := b.API().ListAccounts(t.Context())
	require.NoError(t, err)
	require.Empty(t, accs)
}

func TestCreateAccount(t *testing.T) {
	dataDir := t.TempDir()
	b, err := NewStatusBackend(
		dataDir,
		WithLogger(testutils.MustCreateTestLogger().Named("backend")),
	)
	require.NoError(t, err)
	require.NotNil(t, b)

	// Create new account
	request := &requests.CreateAccount{
		Password:                  gofakeit.LetterN(10),
		KdfIterations:             gofakeit.Number(1, 10),
		DeviceName:                gofakeit.LetterN(10),
		DisplayName:               gofakeit.LetterN(10),
		WakuV2LightClient:         gofakeit.Bool(),
		ThirdpartyServicesEnabled: gofakeit.Bool(),
	}
	acc, err := b.API().CreateAccount(t.Context(), request)
	require.NoError(t, err)

	accs, err := b.API().ListAccounts(t.Context())
	require.NoError(t, err)
	require.Len(t, accs, 1)
	assert.Equal(t, acc.KeyUID, accs[0].KeyUID)
	assert.Equal(t, request.KdfIterations, accs[0].KDFIterations)

	// Check wallet database was created
	walletDBPath := path.Join(dataDir, fmt.Sprintf("%s-wallet.db", acc.KeyUID))
	require.FileExists(t, walletDBPath)

	walletDB, err := sqlite.OpenDB(walletDBPath, request.Password, request.KdfIterations)
	require.NoError(t, err)
	require.NotNil(t, walletDB)

	err = walletDB.Ping()
	require.NoError(t, err)

	err = walletDB.Close()
	require.NoError(t, err)

	// Check app database was created
	appDBPath := path.Join(dataDir, fmt.Sprintf("%s-v4.db", acc.KeyUID))
	require.FileExists(t, appDBPath)

	appDB, err := sqlite.OpenDB(appDBPath, request.Password, request.KdfIterations)
	require.NoError(t, err)
	require.NotNil(t, appDB)

	t.Cleanup(func() {
		err = appDB.Close()
		assert.NoError(t, err)
	})

	err = appDB.Ping()
	require.NoError(t, err)

	// Read created account settings, without login
	accountsDB, err := accounts.NewDB(appDB)
	require.NoError(t, err)
	require.NotNil(t, accountsDB)

	accountSettings, err := accountsDB.GetSettings()
	require.NoError(t, err)
	require.NotNil(t, accountSettings)

	assert.Equal(t, acc.KeyUID, accountSettings.KeyUID)
	assert.Equal(t, request.DisplayName, accountSettings.DisplayName)
	assert.Equal(t, request.ThirdpartyServicesEnabled, accountSettings.ThirdpartyServicesEnabled)

	deviceName, err := accountsDB.DeviceName()
	require.NoError(t, err)
	assert.Equal(t, request.DeviceName, deviceName)

	nodeConfig, err := accountsDB.GetNodeConfig()
	require.NoError(t, err)
	require.NotNil(t, nodeConfig)
	assert.Equal(t, request.WakuV2LightClient, nodeConfig.WakuV2Config.LightClient)
}

func TestLoginWrongPassword(t *testing.T) {
	b, err := NewStatusBackend(
		t.TempDir(),
		WithLogger(testutils.MustCreateTestLogger().Named("backend")),
	)
	require.NoError(t, err)
	require.NotNil(t, b)

	request := &requests.CreateAccount{
		Password:      gofakeit.LetterN(10),
		KdfIterations: gofakeit.Number(1, 10),
	}
	acc, err := b.API().CreateAccount(t.Context(), request)
	require.NoError(t, err)

	loginRequest := &requests.Login{
		KeyUID:   acc.KeyUID,
		Password: request.Password + "wrong",
	}
	err = b.API().Login(loginRequest)
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid password or corrupted database")
}
