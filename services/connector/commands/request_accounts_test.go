package commands

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/accounts"
)

type RPCClientMock struct {
	response string
}

func (c *RPCClientMock) CallRaw(request string) string {
	return c.response
}

func (c *RPCClientMock) SetResponse(response string) {
	c.response = response
}

func TestFailToRequestAccountsWithMissingDAppFields(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	cmd := &RequestAccountsCommand{AccountsCommand: AccountsCommand{
		Db: db,
	}}

	request := RPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "eth_requestAccounts",
		Params:  []interface{}{},
	}

	result, err := cmd.Execute(request)
	assert.Equal(t, err, ErrRequestMissingDAppData)
	assert.Empty(t, result)
}

func TestRequestAccountsTwoTimes(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	rpcClient := &RPCClientMock{}

	cmd := &RequestAccountsCommand{
		RpcClient: rpcClient,
		AccountsCommand: AccountsCommand{
			Db: db,
		}}

	request := RPCRequest{
		JSONRPC:     "2.0",
		ID:          1,
		Method:      "eth_requestAccounts",
		Params:      []interface{}{},
		Origin:      "http://testDAppURL",
		DAppName:    "testDAppName",
		DAppIconUrl: "http://testDAppIconUrl",
	}

	accountAddress := types.BytesToAddress(types.FromHex("0x6d0aa2a774b74bb1d36f97700315adf962c69fcg"))

	acountsResponse := &RawAccountsResponse{
		JSONRPC: "2.0",
		ID:      1,
		Result:  []accounts.Account{{Address: accountAddress}},
	}
	acountsResponseJSON, err := json.Marshal(acountsResponse)
	assert.NoError(t, err)

	rpcClient.SetResponse(string(acountsResponseJSON))

	response, err := cmd.Execute(request)
	assert.NoError(t, err)

	result := &AccountsResponse{}
	err = json.Unmarshal([]byte(response), result)

	assert.NoError(t, err)
	assert.Len(t, result.Accounts, 1)
	assert.Equal(t, accountAddress.Hex(), result.Accounts[0])

	// Setting empty response here to ensure that the account is not requested again
	rpcClient.SetResponse("")

	response, err = cmd.Execute(request)
	assert.NoError(t, err)

	err = json.Unmarshal([]byte(response), result)

	assert.NoError(t, err)
	assert.Len(t, result.Accounts, 1)
	assert.Equal(t, accountAddress.Hex(), result.Accounts[0])
}
