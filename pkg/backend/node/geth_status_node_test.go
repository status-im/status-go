package node

import (
	"os"
	"path"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/rpc/network/testutil"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/params"
	walletcommon "github.com/status-im/status-go/services/wallet/common"
)

func TestStatusNodeStart(t *testing.T) {
	config, err := params.NewNodeConfig("", walletcommon.EthereumSepolia)
	require.NoError(t, err)

	// StatusNode startup creates TokenManager, which requires at least one active network.
	config.Networks = testutil.MinimalActiveNetworks()

	n := New(nil, nil, testutils.MustCreateTestLogger())

	// checks before node is started
	require.Nil(t, n.Config())
	require.Nil(t, n.RPCClient())

	appDB, walletDB, stop, err := setupTestDBs()
	defer func() {
		err := stop()
		if err != nil {
			n.logger.Error("stopping db", zap.Error(err))
		}
	}()
	require.NoError(t, err)
	n.appDB = appDB
	n.walletDB = walletDB

	// start node
	require.NoError(t, n.Start(config))

	// checks after node is started
	require.True(t, n.IsRunning())
	require.NotNil(t, n.Config())
	require.NotNil(t, n.RPCClient())
	require.NotNil(t, n.TokenManager())
	require.NotNil(t, n.tokenManagerStartDone)
	nativeToken, err := n.TokenManager().GetTokenByChainAddress(walletcommon.EthereumMainnet, common.Address{})
	require.NoError(t, err)
	require.NotNil(t, nativeToken)

	// try to start already started node
	require.EqualError(t, n.Start(config), ErrNodeRunning.Error())

	n.StartTokenManager()
	startDone := n.tokenManagerStartDone
	require.NotNil(t, startDone)

	n.StartTokenManager()
	require.Equal(t, startDone, n.tokenManagerStartDone)
	<-startDone

	// stop node
	require.NoError(t, n.Stop())
	require.Nil(t, n.tokenManagerStartDone)
	// try to stop already stopped node
	require.EqualError(t, n.Stop(), ErrNoRunningNode.Error())

	// checks after node is stopped
	require.Nil(t, n.RPCClient())
}

func TestStatusNodeWithDataDir(t *testing.T) {
	dir := t.TempDir()

	// keystore directory
	keyStoreDir := path.Join(dir, "keystore")
	err := os.MkdirAll(keyStoreDir, os.ModePerm)
	require.NoError(t, err)

	// Start requires at least one active network because TokenManager is created during node startup.
	config := params.NodeConfig{
		RootDataDir: dir,
		Networks:    testutil.MinimalActiveNetworks(),
	}

	n, stop1, stop2, err := createStatusNode()
	defer func() {
		err := stop1()
		if err != nil {
			n.logger.Error("stopping db", zap.Error(err))
		}
	}()
	defer func() {
		err := stop2()
		if err != nil {
			n.logger.Error("stopping multiaccount db", zap.Error(err))
		}
	}()
	require.NoError(t, err)

	require.NoError(t, n.Start(&config))
	require.NoError(t, n.Stop())
}
