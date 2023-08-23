package node

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"
)

type TestServiceAPI struct{}

func (api *TestServiceAPI) SomeMethod(_ context.Context) (string, error) {
	return "some method result", nil
}

func setupTestDBs() (appDB *sql.DB, walletDB *sql.DB, closeFn func() error, err error) {
	appDB, err = helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to setup app db: %w", err)
	}

	walletDB, err = helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to setup wallet db: %w", err)
	}
	return appDB, walletDB, func() error {
		appErr := appDB.Close()
		walletErr := walletDB.Close()
		if appErr != nil {
			return fmt.Errorf("failed to close app db: %w", appErr)
		}
		if walletErr != nil {
			return fmt.Errorf("failed to close wallet db: %w", walletErr)
		}
		return nil
	}, err
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

	appDB, walletDB, stop, err := setupTestDBs()
	defer func() {
		err := stop()
		if err != nil {
			statusNode.log.Error("stopping db", err)
		}
	}()
	if err != nil {
		return nil, err
	}
	statusNode.appDB = appDB
	statusNode.walletDB = walletDB

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
	appDB, walletDB, stop1, err := setupTestDBs()
	if err != nil {
		return nil, nil, nil, err
	}
	statusNode := New(nil)
	statusNode.SetAppDB(appDB)
	statusNode.SetWalletDB(walletDB)

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
