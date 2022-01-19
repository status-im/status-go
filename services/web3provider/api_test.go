package web3provider

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/permissions"
	"github.com/status-im/status-go/t/utils"
	"github.com/status-im/status-go/transactions/fake"

	gethrpc "github.com/ethereum/go-ethereum/rpc"
	statusRPC "github.com/status-im/status-go/rpc"
)

func createDB(t *testing.T) (*sql.DB, func()) {
	tmpfile, err := ioutil.TempFile("", "provider-tests-")
	require.NoError(t, err)
	db, err := appdatabase.InitializeDB(tmpfile.Name(), "provider-tests")
	require.NoError(t, err)
	return db, func() {
		require.NoError(t, db.Close())
		require.NoError(t, os.Remove(tmpfile.Name()))
	}
}

func setupTestAPI(t *testing.T) (*API, func()) {
	db, cancel := createDB(t)

	keyStoreDir, err := ioutil.TempDir(os.TempDir(), "accounts")
	require.NoError(t, err)

	// Creating a dummy status node to simulate what it's done in get_status_node.go
	upstreamConfig := params.UpstreamRPCConfig{
		URL:     "https://mainnet.infura.io/v3/800c641949d64d768a5070a1b0511938",
		Enabled: true,
	}

	txServiceMockCtrl := gomock.NewController(t)
	server, _ := fake.NewTestServer(txServiceMockCtrl)
	client := gethrpc.DialInProc(server)

	rpcClient, err := statusRPC.NewClient(client, 1, upstreamConfig, nil, db)
	require.NoError(t, err)

	// import account keys
	utils.Init()
	require.NoError(t, utils.ImportTestAccount(keyStoreDir, utils.GetAccount1PKFile()))

	accManager := account.NewGethManager()

	nodeConfig := &params.NodeConfig{
		KeyStoreDir: keyStoreDir,
		NetworkID:   1,
	}

	service, err := NewService(db, rpcClient, nodeConfig, accManager, nil, nil)
	require.NoError(t, err)

	networks := json.RawMessage("{}")
	settings := settings.Settings{
		DappsAddress: types.HexToAddress(utils.TestConfig.Account1.WalletAddress),
		Networks:     &networks,
	}

	accounts := []accounts.Account{
		{Address: types.HexToAddress(utils.TestConfig.Account1.WalletAddress), Chat: true, Wallet: true},
	}
	require.NoError(t, service.accountsDB.SaveAccounts(accounts))

	require.NoError(t, service.accountsDB.CreateSettings(settings, *nodeConfig))

	return &API{
		s: service,
	}, cancel
}

func TestRequestPermission(t *testing.T) {
	api, cancel := setupTestAPI(t)
	defer cancel()

	request := APIRequest{
		Hostname: "www.status.im",
	}

	_, err := api.ProcessAPIRequest(request)
	require.Error(t, err)

	request.Permission = PermissionWeb3

	response, err := api.ProcessAPIRequest(request)
	require.NoError(t, err)
	require.False(t, response.IsAllowed)
	require.Equal(t, ResponseAPI, response.ProviderResponse.ResponseType)

	_ = api.s.permissionsDB.AddPermissions(permissions.DappPermissions{Name: "www.status.im", Permissions: []string{PermissionWeb3, PermissionContactCode, "RandomPermission"}})

	response, err = api.ProcessAPIRequest(request)
	require.NoError(t, err)
	require.True(t, response.IsAllowed)

	d := make([]interface{}, 1)
	d[0] = types.HexToAddress(utils.TestConfig.Account1.WalletAddress)
	var data interface{} = d
	require.Equal(t, data, response.Data)

	request.Permission = PermissionContactCode
	response, err = api.ProcessAPIRequest(request)
	require.NoError(t, err)
	require.True(t, response.IsAllowed)

	pubKey, _ := api.s.accountsDB.GetPublicKey()
	data = pubKey
	require.Equal(t, data, response.Data)

	request.Permission = "RandomPermission"
	_, err = api.ProcessAPIRequest(request)
	require.Error(t, err)
}

func TestWeb3Call(t *testing.T) {
	api, cancel := setupTestAPI(t)
	defer cancel()

	request := Web3SendAsyncReadOnlyRequest{
		Hostname:  "www.status.im",
		MessageID: 1,
		Payload: ETHPayload{
			ID:      1,
			JSONRPC: "2.0",
			From:    types.HexToAddress(utils.TestConfig.Account1.WalletAddress).String(),
			Method:  "net_version",
			Params:  []interface{}{},
		},
	}

	response, err := api.ProcessWeb3ReadOnlyRequest(request)
	require.NoError(t, err)
	require.Equal(t, `{"jsonrpc":"2.0","id":1,"result":"1"}`, string(response.Result.(json.RawMessage)))

	request.Payload.Method = "eth_accounts"
	response, err = api.ProcessWeb3ReadOnlyRequest(request)
	require.NoError(t, err)
	require.Equal(t, uint(4100), response.Error.(Web3SendAsyncReadOnlyError).Code)

	_ = api.s.permissionsDB.AddPermissions(permissions.DappPermissions{Name: "www.status.im", Permissions: []string{PermissionWeb3}})

	response, err = api.ProcessWeb3ReadOnlyRequest(request)
	require.NoError(t, err)

	d := make([]types.Address, 1)
	d[0] = types.HexToAddress(utils.TestConfig.Account1.WalletAddress)
	var data interface{} = d // eth_account is an array of addresses
	require.Equal(t, data, response.Result.(JSONRPCResponse).Result)

	request.Payload.Method = "eth_coinbase"
	data = d[0] // eth_coinbase is an address
	response, err = api.ProcessWeb3ReadOnlyRequest(request)
	require.NoError(t, err)
	require.Equal(t, data, response.Result.(JSONRPCResponse).Result)
}

func TestWeb3Signature(t *testing.T) {
	api, cancel := setupTestAPI(t)
	defer cancel()

	_ = api.s.permissionsDB.AddPermissions(permissions.DappPermissions{Name: "www.status.im", Permissions: []string{PermissionWeb3}})

	request := Web3SendAsyncReadOnlyRequest{
		Hostname:  "www.status.im",
		MessageID: 1,
		Payload: ETHPayload{
			ID:       1,
			JSONRPC:  "2.0",
			From:     types.HexToAddress(utils.TestConfig.Account1.WalletAddress).String(),
			Method:   "personal_sign",
			Params:   []interface{}{types.HexBytes{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}},
			Password: "wrong-password",
		},
	}

	response, err := api.ProcessWeb3ReadOnlyRequest(request)
	require.NoError(t, err)
	require.Equal(t, uint(4100), response.Error.(Web3SendAsyncReadOnlyError).Code)
	require.Equal(t, "could not decrypt key with given password", response.Error.(Web3SendAsyncReadOnlyError).Message)

	request.Payload.Password = utils.TestConfig.Account1.Password
	response, err = api.ProcessWeb3ReadOnlyRequest(request)
	require.NoError(t, err)
	require.Equal(t, types.HexBytes(types.Hex2Bytes("0xc113a94f201334da86b8237c676951932d2b0ee2b539d941736da5b736f0f224448be6435846a9df9ea0085d92b107b6e49b1786e90d6604d3ef7d6f6ec19d531c")), response.Result.(JSONRPCResponse).Result.(types.HexBytes))
}
