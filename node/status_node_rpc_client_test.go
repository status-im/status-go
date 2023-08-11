package node

import (
	"context"
	"database/sql"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/t/helpers"
)

type TestServiceAPI struct{}

func (api *TestServiceAPI) SomeMethod(_ context.Context) (string, error) {
	return "some method result", nil
}

func setupTestDB() (*sql.DB, func() error, error) {
	return helpers.SetupTestSQLDB(appdatabase.DbInitializer{}, "tests")
}

func setupTestMultiDB() (*multiaccounts.Database, func() error, error) {
	tmpfile, err := ioutil.TempFile("", "tests")
	if err != nil {
		return nil, nil, err
	}
	db, err := multiaccounts.InitializeDB(tmpfile.Name())
	if err != nil {
		return nil, nil, err
	}
	return db, func() error {
		err := db.Close()
		if err != nil {
			return err
		}
		return os.Remove(tmpfile.Name())
	}, nil
}

func createAndStartStatusNode(config *params.NodeConfig) (*StatusNode, error) {
	statusNode := New(nil)

	db, stop, err := setupTestDB()
	defer func() {
		err := stop()
		if err != nil {
			statusNode.log.Error("stopping db", err)
		}
	}()
	if err != nil {
		return nil, err
	}
	statusNode.appDB = db

	ma, stop2, err := setupTestMultiDB()
	defer func() {
		err := stop2()
		if err != nil {
			statusNode.log.Error("stopping multiaccount db", err)
		}
	}()
	if err != nil {
		return nil, err
	}
	statusNode.multiaccountsDB = ma

	err = statusNode.Start(config, nil)
	if err != nil {
		return nil, err
	}

	return statusNode, nil
}

func createStatusNode() (*StatusNode, func() error, func() error, error) {
	db, stop1, err := setupTestDB()
	if err != nil {
		return nil, nil, nil, err
	}
	statusNode := New(nil)
	statusNode.SetAppDB(db)

	ma, stop2, err := setupTestMultiDB()
	statusNode.SetMultiaccountsDB(ma)

	return statusNode, stop1, stop2, err
}

func TestNodeRPCClientCallOnlyPublicAPIs(t *testing.T) {
	var err error

	statusNode, err := createAndStartStatusNode(&params.NodeConfig{
		APIModules: "", // no whitelisted API modules; use only public APIs
		UpstreamConfig: params.UpstreamRPCConfig{
			URL:     "https://infura.io",
			Enabled: true},
		WakuConfig: params.WakuConfig{
			Enabled: true,
		},
	})
	require.NoError(t, err)
	defer func() {
		err := statusNode.Stop()
		require.NoError(t, err)
	}()

	client := statusNode.RPCClient()
	require.NotNil(t, client)

	// call public API with public RPC Client
	result, err := statusNode.CallRPC(`{"jsonrpc": "2.0", "id": 1, "method": "eth_uninstallFilter", "params": ["id"]}`)
	require.NoError(t, err)

	// the call is successful
	require.False(t, strings.Contains(result, "error"))

	result, err = statusNode.CallRPC(`{"jsonrpc": "2.0", "id": 1, "method": "waku_info"}`)
	require.NoError(t, err)

	// call private API with public RPC client
	require.Equal(t, ErrRPCMethodUnavailable, result)

}

func TestNodeRPCPrivateClientCallPrivateService(t *testing.T) {
	var err error

	statusNode, err := createAndStartStatusNode(&params.NodeConfig{
		WakuConfig: params.WakuConfig{
			Enabled: true,
		},
	})
	require.NoError(t, err)
	defer func() {
		err := statusNode.Stop()
		require.NoError(t, err)
	}()

	result, err := statusNode.CallPrivateRPC(`{"jsonrpc": "2.0", "id": 1, "method": "waku_info"}`)
	require.NoError(t, err)

	// the call is successful
	require.False(t, strings.Contains(result, "error"))

	_, err = statusNode.CallPrivateRPC(`{"jsonrpc": "2.0", "id": 1, "method": "settings_getSettings"}`)
	require.NoError(t, err)
}
