package commands

import (
	"testing"
	"time"

	"github.com/status-im/status-go/internal/crypto/types"
	persistence "github.com/status-im/status-go/services/connector/database"

	"github.com/stretchr/testify/assert"
)

func TestClientHandlerTimeout(t *testing.T) {
	db, cleanup := createWalletDB(t)
	t.Cleanup(cleanup)

	clientHandler := NewClientSideHandler(db, nil)

	backupWalletResponseMaxInterval := WalletResponseMaxInterval
	WalletResponseMaxInterval = 1 * time.Millisecond

	_, _, err := clientHandler.RequestShareAccountForDApp(testDAppData)
	assert.Equal(t, ErrWalletResponseTimeout, err)
	WalletResponseMaxInterval = backupWalletResponseMaxInterval
}

func TestRequestRejectedWhileWaiting(t *testing.T) {
	db, cleanup := createWalletDB(t)
	t.Cleanup(cleanup)

	clientHandler := NewClientSideHandler(db, nil)

	clientHandler.setRequestRunning()

	_, _, err := clientHandler.RequestShareAccountForDApp(testDAppData)
	assert.Equal(t, ErrAnotherConnectorOperationIsAwaitingFor, err)
}

func TestRecallDAppPermission(t *testing.T) {
	db, cleanup := createWalletDB(t)
	t.Cleanup(cleanup)

	dapp := persistence.DApp{
		Name:          "Test DApp",
		URL:           "http://testDAppURL",
		IconURL:       "http://testDAppIconUrl",
		ClientID:      "",
		SharedAccount: types.HexToAddress("0x1234567890"),
		ChainID:       0x1,
	}

	err := persistence.UpsertDApp(db, &dapp)
	assert.NoError(t, err)

	persistedDapp, err := persistence.SelectDApp(db, dapp.URL, dapp.ClientID)
	assert.Equal(t, persistedDapp, &dapp)
	assert.NoError(t, err)

	clientHandler := NewClientSideHandler(db, nil)
	err = clientHandler.RecallDAppPermissions(RecallDAppPermissionsArgs{URL: dapp.URL, ClientID: dapp.ClientID})
	assert.NoError(t, err)

	err = clientHandler.RecallDAppPermissions(RecallDAppPermissionsArgs{})
	assert.ErrorIs(t, err, ErrEmptyUrl)

	err = clientHandler.RecallDAppPermissions(RecallDAppPermissionsArgs{URL: dapp.URL, ClientID: dapp.ClientID})
	assert.ErrorIs(t, err, ErrDAppDoesNotHavePermissions)

	recalledDapp, err := persistence.SelectDApp(db, dapp.URL, dapp.ClientID)

	assert.Equal(t, recalledDapp, (*persistence.DApp)(nil))
	assert.NoError(t, err)
}
