package api_test

import (
	"context"
	"io/ioutil"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/status-im/status-go/cmd/api"
	"github.com/status-im/status-go/geth/params"
	"github.com/stretchr/testify/assert"
)

// TestStartStopServer tests starting the server without any client
// connection. It is actively killed by using a cancel context.
func TestStartStopServer(t *testing.T) {
	assert := assert.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	srv, err := api.NewServer(ctx, "localhost", "12345")
	assert.NoError(err)
	assert.NotNil(srv)
	assert.NoError(srv.Err())

	// Terminate and wait so that background goroutine can end.
	cancel()
	time.Sleep(1 * time.Millisecond)

	assert.Equal(srv.Err(), context.Canceled)
}

// TestConnectClient test starting the server and connecting it
// with a client.
func TestConnectClient(t *testing.T) {
	assert := assert.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv, err := api.NewServer(ctx, "[::1]", "12345")
	assert.NoError(err)

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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv, err := api.NewServer(ctx, "[::1]", "12345")
	assert.NoError(err)

	clnt, err := api.NewClient("[::1]", "12345")
	assert.NoError(err)

	err = clnt.StatusStartNode(configJSON)
	assert.NoError(err)
	assert.NoError(srv.Err())

	err = clnt.StatusStopNode()
	assert.NoError(err)
	assert.NoError(srv.Err())
}

//-----
// HELPERS
//-----

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
