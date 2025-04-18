package node

import (
	"os"
	"path"
	"testing"

	"go.uber.org/zap"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/t/utils"
)

func TestStatusNodeStart(t *testing.T) {
	config, err := utils.MakeTestNodeConfigWithDataDir("", "", params.StatusChainNetworkID)
	require.NoError(t, err)
	n := New(nil, tt.MustCreateTestLogger())

	// checks before node is started
	require.Nil(t, n.GethNode())
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
	require.NoError(t, n.Start(config, nil))

	// checks after node is started
	require.True(t, n.IsRunning())
	require.NotNil(t, n.GethNode())
	require.NotNil(t, n.Config())
	require.NotNil(t, n.RPCClient())
	accountManager, err := n.AccountManager()
	require.Nil(t, err)
	require.NotNil(t, accountManager)
	// try to start already started node
	require.EqualError(t, n.Start(config, nil), ErrNodeRunning.Error())

	// stop node
	require.NoError(t, n.Stop())
	// try to stop already stopped node
	require.EqualError(t, n.Stop(), ErrNoRunningNode.Error())

	// checks after node is stopped
	require.Nil(t, n.GethNode())
	require.Nil(t, n.RPCClient())
}

func TestStatusNodeWithDataDir(t *testing.T) {
	dir := t.TempDir()

	// keystore directory
	keyStoreDir := path.Join(dir, "keystore")
	err := os.MkdirAll(keyStoreDir, os.ModePerm)
	require.NoError(t, err)

	config := params.NodeConfig{
		DataDir:     dir,
		KeyStoreDir: keyStoreDir,
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

	require.NoError(t, n.Start(&config, nil))
	require.NoError(t, n.Stop())
}
