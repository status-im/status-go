package commands

import (
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	persistence "github.com/status-im/status-go/services/connector/database"
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

	cmd := &AccountsCommand{
		Db: db,
	}

	request := RPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "eth_accounts",
		Params:  []interface{}{},
	}

	result, err := cmd.Execute(request)
	assert.Equal(t, err, ErrRequestMissingDAppData)
	assert.Empty(t, result)
}

func TestFailToGetAccountForUnpermittedDApp(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	cmd := &AccountsCommand{
		Db: db,
	}

	request := RPCRequest{
		JSONRPC:     "2.0",
		ID:          1,
		Method:      "eth_accounts",
		Params:      []interface{}{},
		Origin:      "http://testDAppURL",
		DAppName:    "testDAppName",
		DAppIconUrl: "http://testDAppIconUrl",
	}

	result, err := cmd.Execute(request)
	assert.Equal(t, err, ErrDAppIsNotPermittedByUser)
	assert.Empty(t, result)
}

func TestGetAccountForPermittedDApp(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	cmd := &AccountsCommand{
		Db: db,
	}

	request := RPCRequest{
		JSONRPC:     "2.0",
		ID:          1,
		Method:      "eth_accounts",
		Params:      []interface{}{},
		Origin:      "http://testDAppURL",
		DAppName:    "testDAppName",
		DAppIconUrl: "http://testDAppIconUrl",
	}

	dApp := persistence.DApp{
		URL:           request.Origin,
		Name:          request.DAppName,
		IconURL:       request.DAppIconUrl,
		SharedAccount: "0x6d0aa2a774b74bb1d36f97700315adf962c69fcg",
	}

	err := persistence.UpsertDApp(db, &dApp)
	assert.NoError(t, err)

	response, err := cmd.Execute(request)
	assert.NoError(t, err)

	result := &AccountsResponse{}
	err = json.Unmarshal([]byte(response), result)

	assert.NoError(t, err)
	assert.Len(t, result.Accounts, 1)
	assert.Equal(t, dApp.SharedAccount, result.Accounts[0])
}
