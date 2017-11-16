package api_test

import (
	"context"
	"io/ioutil"
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/status-im/status-go/cmd/api"
	gethapi "github.com/status-im/status-go/geth/api"
	"github.com/status-im/status-go/geth/params"
	"github.com/stretchr/testify/assert"
)

// TestConnectClient test starting the server and connecting it
// with a client.
func TestConnectClient(t *testing.T) {
	assert := assert.New(t)
	srv := mkServer(assert)

	clnt, err := api.NewClient("[::1]", "12345")
	assert.NoError(err)

	addrs, err := clnt.AdminGetAddresses()
	assert.NoError(err)
	assert.True(len(addrs) != 0)
	assert.NoError(srv.Err())
}

// TestStartNode tests starting a node on the server by a
// client command.
func TestStartStopNode(t *testing.T) {
	assert := assert.New(t)
	configJSON, cleanup, err := mkConfigJSON("status-start-stop-node")
	assert.NoError(err)
	defer cleanup()

	srv := mkServer(assert)

	clnt, err := api.NewClient("[::1]", "12345")
	assert.NoError(err)

	err = clnt.StatusStartNode(configJSON)
	assert.NoError(err)
	assert.NoError(srv.Err())

	err = clnt.StatusStopNode()
	assert.NoError(err)
	assert.NoError(srv.Err())
}

// TestCreateAccount tests creating an account on the server.
func TestCreateAccount(t *testing.T) {
	assert := assert.New(t)
	configJSON, cleanup, err := mkConfigJSON("status-create-account")
	assert.NoError(err)
	defer cleanup()

	srv := mkServer(assert)

	clnt, err := api.NewClient("[::1]", "12345")
	assert.NoError(err)

	err = clnt.StatusStartNode(configJSON)
	assert.NoError(err)
	assert.NoError(srv.Err())

	account, publicKey, mnemonic, err := clnt.StatusCreateAccount("password")
	assert.NoError(err)
	assert.NotEmpty(account)
	assert.NotEmpty(publicKey)
	assert.NotEmpty(mnemonic)

	err = clnt.StatusStopNode()
	assert.NoError(err)
	assert.NoError(srv.Err())
}

// TestSelectAccountLogout tests selecting an account on the server
// and logging out afterwards.
func TestSelectAccountLogout(t *testing.T) {
	assert := assert.New(t)
	configJSON, cleanup, err := mkConfigJSON("status-create-account")
	assert.NoError(err)
	defer cleanup()

	srv := mkServer(assert)

	clnt, err := api.NewClient("[::1]", "12345")
	assert.NoError(err)

	err = clnt.StatusStartNode(configJSON)
	assert.NoError(err)
	assert.NoError(srv.Err())

	address, publicKey, mnemonic, err := clnt.StatusCreateAccount("password")
	assert.NoError(err)
	assert.NotEmpty(address)
	assert.NotEmpty(publicKey)
	assert.NotEmpty(mnemonic)

	err = clnt.StatusSelectAccount(address, "password")
	assert.NoError(err)

	err = clnt.StatusLogout()
	assert.NoError(err)

	err = clnt.StatusStopNode()
	assert.NoError(err)
	assert.NoError(srv.Err())
}

//-----
// HELPERS
//-----

var (
	mu  sync.Mutex
	srv *api.Server
)

// mkServer lazily creates or reuses a server.
func mkServer(assert *assert.Assertions) *api.Server {
	mu.Lock()
	defer mu.Unlock()
	if srv == nil {
		var err error
		backend := gethapi.NewStatusBackend()
		srv, err = api.NewServer(context.Background(), backend, "[::1]", "12345")
		assert.NoError(err)
	}
	return srv
}

// mkConfigJSON creates a configuration matching to
// a temporary directory and a cleanup for that directory.
func mkConfigJSON(name string) (string, func(), error) {
	tmpDir, err := ioutil.TempDir(os.TempDir(), name)
	if err != nil {
		return "", nil, err
	}
	cleanup := func() {
		os.RemoveAll(tmpDir) //nolint: errcheck
	}
	configJSON := `{
		"NetworkId": ` + strconv.Itoa(params.RopstenNetworkID) + `,
		"DataDir": "` + tmpDir + `",
		"LogLevel": "INFO",
		"RPCEnabled": true
	}`
	return configJSON, cleanup, nil
}
