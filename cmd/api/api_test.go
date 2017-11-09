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
	"github.com/stretchr/testify/suite"
)

// TestAPI runs the whole API test suite.
func TestAPI(t *testing.T) {
	suite.Run(t, new(APITestSuite))
}

// APITestSuite contains all tests of the exposed API.
type APITestSuite struct {
	suite.Suite
}

// TestStartStopServer tests starting the server without any client
// connection. It is actively killed by using a cancel context.
func (s *APITestSuite) TestStartStopServer() {
	ctx, cancel := context.WithCancel(context.Background())
	srv, err := api.NewServer(ctx, "localhost", "12345")
	s.NoError(err)
	s.NotNil(srv)
	s.NoError(srv.Err())

	// Terminate and wait so that background goroutine can end.
	cancel()
	time.Sleep(1 * time.Millisecond)

	s.Equal(srv.Err(), context.Canceled)
}

// TestConnectClient test starting the server and connecting it
// with a client.
func (s *APITestSuite) TestConnectClient() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv, err := api.NewServer(ctx, "[::1]", "12345")
	s.NoError(err)

	clnt, err := api.NewClient("[::1]", "12345")
	s.NoError(err)

	addrs, err := clnt.AdminGetAddresses()
	s.NoError(err)
	s.True(len(addrs) != 0)
	s.NoError(srv.Err())
}

// TestStartNode tests starting a node on the server by a
// client command.
func (s *APITestSuite) TestStartStopNode() {
	tmpDir, err := ioutil.TempDir(os.TempDir(), "status-start-stop-node")
	s.NoError(err)
	defer os.RemoveAll(tmpDir) //nolint: errcheck

	configJSON := `{
		"NetworkId": ` + strconv.Itoa(params.RopstenNetworkID) + `,
		"DataDir": "` + tmpDir + `",
		"LogLevel": "INFO",
		"RPCEnabled": true
	}`

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv, err := api.NewServer(ctx, "[::1]", "12345")
	s.NoError(err)

	clnt, err := api.NewClient("[::1]", "12345")
	s.NoError(err)

	err = clnt.StatusStartNode(configJSON)
	s.NoError(err)
	s.NoError(srv.Err())

	err = clnt.StatusStopNode()
	s.NoError(err)
}
