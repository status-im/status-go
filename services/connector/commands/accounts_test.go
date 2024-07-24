package commands

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/status-im/status-go/eth-node/types"
)

func TestFailToGetAccountWithMissingDAppFields(t *testing.T) {
	db, close := SetupTestDB(t)
	defer close()

	cmd := &AccountsCommand{Db: db}

	// Missing DApp fields
	request, err := ConstructRPCRequest("eth_accounts", []interface{}{}, nil)
	assert.NoError(t, err)

	result, err := cmd.Execute(request)
	assert.Equal(t, ErrRequestMissingDAppData, err)
	assert.Empty(t, result)
}

func TestFailToGetAccountForUnpermittedDApp(t *testing.T) {
	db, close := SetupTestDB(t)
	defer close()

	cmd := &AccountsCommand{Db: db}

	request, err := ConstructRPCRequest("eth_accounts", []interface{}{}, &testDAppData)
	assert.NoError(t, err)

	result, err := cmd.Execute(request)
	assert.Equal(t, ErrDAppIsNotPermittedByUser, err)
	assert.Empty(t, result)
}

func TestGetAccountForPermittedDApp(t *testing.T) {
	db, close := SetupTestDB(t)
	defer close()

	cmd := &AccountsCommand{Db: db}

	sharedAccount := types.HexToAddress("0x6d0aa2a774b74bb1d36f97700315adf962c69fcg")

	err := PersistDAppData(db, testDAppData, sharedAccount, 0x123)
	assert.NoError(t, err)

	request, err := ConstructRPCRequest("eth_accounts", []interface{}{}, &testDAppData)
	assert.NoError(t, err)

	response, err := cmd.Execute(request)
	assert.NoError(t, err)

	// Unmarshal the response into a slice of addresses
	var result []types.Address
	err = json.Unmarshal([]byte(response), &result)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, sharedAccount, result[0])
}
