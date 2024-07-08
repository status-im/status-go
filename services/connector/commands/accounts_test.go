package commands

import (
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"
)

func setupTestDB(t *testing.T) (db *sql.DB, close func()) {
	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)
	return db, func() {
		require.NoError(t, db.Close())
	}
}

func TestFailToGetAccountWithMissingDAppFields(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	cmd := &AccountsCommand{Db: db}

	// Missing DApp fields
	request := constructRPCRequest("eth_accounts", []interface{}{}, nil)

	result, err := cmd.Execute(request)
	assert.Equal(t, ErrRequestMissingDAppData, err)
	assert.Empty(t, result)
}

func TestFailToGetAccountForUnpermittedDApp(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	cmd := &AccountsCommand{Db: db}

	request := constructRPCRequest("eth_accounts", []interface{}{}, &testDAppData)

	result, err := cmd.Execute(request)
	assert.Equal(t, ErrDAppIsNotPermittedByUser, err)
	assert.Empty(t, result)
}

func TestGetAccountForPermittedDApp(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	cmd := &AccountsCommand{Db: db}

	sharedAccount := types.HexToAddress("0x6d0aa2a774b74bb1d36f97700315adf962c69fcg")

	err := persistDAppData(db, testDAppData, sharedAccount, 0x123)
	assert.NoError(t, err)

	request := constructRPCRequest("eth_accounts", []interface{}{}, &testDAppData)

	response, err := cmd.Execute(request)
	assert.NoError(t, err)

	result := &AccountsResponse{}
	err = json.Unmarshal([]byte(response), result)

	assert.NoError(t, err)
	assert.Len(t, result.Accounts, 1)
	assert.Equal(t, sharedAccount, result.Accounts[0])
}
