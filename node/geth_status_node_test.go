package node

import (
	"math"
	"net"
	"os"
	"path"
	"testing"
	"time"

	"go.uber.org/zap"

	gethnode "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/t/helpers"
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
	require.Equal(t, 0, n.PeerCount())

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
	require.Equal(t, 0, n.PeerCount())
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
	require.Equal(t, 0, n.PeerCount())
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

func TestStatusNodeAddPeer(t *testing.T) {
	var err error

	peer, err := gethnode.New(&gethnode.Config{
		P2P: p2p.Config{
			MaxPeers:    math.MaxInt32,
			NoDiscovery: true,
			ListenAddr:  ":0",
		},
		NoUSB: true,
	})
	require.NoError(t, err)
	require.NoError(t, peer.Start())
	defer func() { require.NoError(t, peer.Close()) }()
	peerURL := peer.Server().Self().URLv4()

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

	// checks before node is started
	require.EqualError(t, n.AddPeer(peerURL), ErrNoRunningNode.Error())

	// start status node
	config := params.NodeConfig{
		MaxPeers: math.MaxInt32,
	}
	require.NoError(t, n.Start(&config, nil))
	defer func() { require.NoError(t, n.Stop()) }()

	errCh := helpers.WaitForPeerAsync(n.Server(), peerURL, p2p.PeerEventTypeAdd, time.Second*5)

	// checks after node is started
	require.NoError(t, n.AddPeer(peerURL))
	require.NoError(t, <-errCh)
	require.Equal(t, 1, n.PeerCount())
}

func TestStatusNodeDiscoverNode(t *testing.T) {
	config := params.NodeConfig{
		NoDiscovery: true,
		ListenAddr:  "127.0.0.1:0",
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
	node, err := n.discoverNode()
	require.NoError(t, err)
	require.Equal(t, net.ParseIP("127.0.0.1").To4(), node.IP())

	config = params.NodeConfig{
		NoDiscovery:   true,
		AdvertiseAddr: "127.0.0.2",
		ListenAddr:    "127.0.0.1:0",
	}

	n1, stop11, stop12, err := createStatusNode()
	defer func() {
		err := stop11()
		if err != nil {
			n1.logger.Error("stopping db", zap.Error(err))
		}
	}()
	defer func() {
		err := stop12()
		if err != nil {
			n1.logger.Error("stopping multiaccount db", zap.Error(err))
		}
	}()
	require.NoError(t, err)

	require.NoError(t, n1.Start(&config, nil))
	node, err = n1.discoverNode()
	require.NoError(t, err)
	require.Equal(t, net.ParseIP("127.0.0.2").To4(), node.IP())
}
