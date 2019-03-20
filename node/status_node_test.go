package node

import (
	"fmt"
	"io/ioutil"
	"math"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/les"
	gethnode "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	whisper "github.com/status-im/whisper/whisperv6"

	"github.com/status-im/status-go/discovery"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/t/utils"
	"github.com/stretchr/testify/require"
)

func TestStatusNodeStart(t *testing.T) {
	config, err := utils.MakeTestNodeConfigWithDataDir("", "", params.StatusChainNetworkID)
	require.NoError(t, err)
	n := New()

	// checks before node is started
	require.Nil(t, n.GethNode())
	require.Nil(t, n.Config())
	require.Nil(t, n.RPCClient())
	require.Equal(t, 0, n.PeerCount())
	_, err = n.AccountManager()
	require.EqualError(t, err, ErrNoGethNode.Error())
	_, err = n.AccountKeyStore()
	require.EqualError(t, err, ErrNoGethNode.Error())

	// start node
	require.NoError(t, n.Start(config))

	// checks after node is started
	require.True(t, n.IsRunning())
	require.NotNil(t, n.GethNode())
	require.NotNil(t, n.Config())
	require.NotNil(t, n.RPCClient())
	require.Equal(t, 0, n.PeerCount())
	accountManager, err := n.AccountManager()
	require.Nil(t, err)
	require.NotNil(t, accountManager)
	keyStore, err := n.AccountKeyStore()
	require.Nil(t, err)
	require.NotNil(t, keyStore)
	// try to start already started node
	require.EqualError(t, n.Start(config), ErrNodeRunning.Error())

	// stop node
	require.NoError(t, n.Stop())
	// try to stop already stopped node
	require.EqualError(t, n.Stop(), ErrNoRunningNode.Error())

	// checks after node is stopped
	require.Nil(t, n.GethNode())
	require.Nil(t, n.RPCClient())
	require.Equal(t, 0, n.PeerCount())
	_, err = n.AccountManager()
	require.EqualError(t, err, ErrNoGethNode.Error())
	_, err = n.AccountKeyStore()
	require.EqualError(t, err, ErrNoGethNode.Error())
}

func TestStatusNodeWithDataDir(t *testing.T) {
	var err error

	dir, err := ioutil.TempDir("", "status-node-test")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.RemoveAll(dir))
	}()

	// keystore directory
	keyStoreDir := path.Join(dir, "keystore")
	err = os.MkdirAll(keyStoreDir, os.ModePerm)
	require.NoError(t, err)

	config := params.NodeConfig{
		DataDir:     dir,
		KeyStoreDir: keyStoreDir,
	}
	n := New()

	require.NoError(t, n.Start(&config))
	require.NoError(t, n.Stop())
}

func TestStatusNodeServiceGetters(t *testing.T) {
	config := params.NodeConfig{
		WhisperConfig: params.WhisperConfig{
			Enabled: true,
		},
		LightEthConfig: params.LightEthConfig{
			Enabled: true,
		},
	}
	n := New()

	var (
		instance interface{}
		err      error
	)

	services := []struct {
		getter func() (interface{}, error)
		typ    reflect.Type
	}{
		{
			getter: func() (interface{}, error) {
				return n.WhisperService()
			},
			typ: reflect.TypeOf(&whisper.Whisper{}),
		},
		{
			getter: func() (interface{}, error) {
				return n.LightEthereumService()
			},
			typ: reflect.TypeOf(&les.LightEthereum{}),
		},
	}

	for _, service := range services {
		t.Run(service.typ.String(), func(t *testing.T) {
			// checks before node is started
			instance, err = service.getter()
			require.EqualError(t, err, ErrNoRunningNode.Error())
			require.Nil(t, instance)

			// start node
			require.NoError(t, n.Start(&config))

			// checks after node is started
			instance, err = service.getter()
			require.NoError(t, err)
			require.NotNil(t, instance)
			require.Equal(t, service.typ, reflect.TypeOf(instance))

			// stop node
			require.NoError(t, n.Stop())

			// checks after node is stopped
			instance, err = service.getter()
			require.EqualError(t, err, ErrNoRunningNode.Error())
			require.Nil(t, instance)
		})
	}
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
	defer func() { require.NoError(t, peer.Stop()) }()
	peerURL := peer.Server().Self().String()

	n := New()

	// checks before node is started
	require.EqualError(t, n.AddPeer(peerURL), ErrNoRunningNode.Error())

	// start status node
	config := params.NodeConfig{
		MaxPeers: math.MaxInt32,
	}
	require.NoError(t, n.Start(&config))
	defer func() { require.NoError(t, n.Stop()) }()

	errCh := helpers.WaitForPeerAsync(n.Server(), peerURL, p2p.PeerEventTypeAdd, time.Second*5)

	// checks after node is started
	require.NoError(t, n.AddPeer(peerURL))
	require.NoError(t, <-errCh)
	require.Equal(t, 1, n.PeerCount())
}

func TestStatusNodeReconnectStaticPeers(t *testing.T) {
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
	defer func() { require.NoError(t, peer.Stop()) }()

	var errCh <-chan error

	peerURL := peer.Server().Self().String()
	n := New()

	// checks before node is started
	require.EqualError(t, n.ReconnectStaticPeers(), ErrNoRunningNode.Error())

	// start status node
	config := params.NodeConfig{
		MaxPeers: math.MaxInt32,
		ClusterConfig: params.ClusterConfig{
			Enabled:     true,
			StaticNodes: []string{peerURL},
		},
	}
	require.NoError(t, n.Start(&config))
	defer func() { require.NoError(t, n.Stop()) }()

	// checks after node is started
	// it may happen that the peer is already connected
	// because it was already added to `StaticNodes`
	connected, err := isPeerConnected(n, peerURL)
	require.NoError(t, err)
	if !connected {
		errCh = helpers.WaitForPeerAsync(n.Server(), peerURL, p2p.PeerEventTypeAdd, time.Second*30)
		require.NoError(t, <-errCh)
	}
	require.Equal(t, 1, n.PeerCount())
	require.Equal(t, peer.Server().Self().ID().String(), n.GethNode().Server().PeersInfo()[0].ID)

	// reconnect static peers
	errDropCh := helpers.WaitForPeerAsync(n.Server(), peerURL, p2p.PeerEventTypeDrop, time.Second*30)

	// it takes at least 30 seconds to bring back previously connected peer
	errAddCh := helpers.WaitForPeerAsync(n.Server(), peerURL, p2p.PeerEventTypeAdd, time.Second*60)
	require.NoError(t, n.ReconnectStaticPeers())
	// first check if a peer gets disconnected
	require.NoError(t, <-errDropCh)
	require.NoError(t, <-errAddCh)
}

func isPeerConnected(node *StatusNode, peerURL string) (bool, error) {
	if !node.IsRunning() {
		return false, ErrNoRunningNode
	}

	parsedPeer, err := enode.ParseV4(peerURL)
	if err != nil {
		return false, err
	}

	server := node.GethNode().Server()

	for _, peer := range server.PeersInfo() {
		if peer.ID == parsedPeer.ID().String() {
			return true, nil
		}
	}

	return false, nil
}

func TestStatusNodeRendezvousDiscovery(t *testing.T) {
	config := params.NodeConfig{
		Rendezvous:  true,
		NoDiscovery: true,
		ClusterConfig: params.ClusterConfig{
			Enabled: true,
			// not necessarily with id, just valid multiaddr
			RendezvousNodes: []string{"/ip4/127.0.0.1/tcp/34012", "/ip4/127.0.0.1/tcp/34011"},
		},
		// use custom address to test the all possibilities
		AdvertiseAddr: "127.0.0.1",
	}
	n := New()
	require.NoError(t, n.Start(&config))
	require.NotNil(t, n.discovery)
	require.True(t, n.discovery.Running())
	require.IsType(t, &discovery.Rendezvous{}, n.discovery)
}

func TestStatusNodeStartDiscoveryManual(t *testing.T) {
	config := params.NodeConfig{
		Rendezvous:  true,
		NoDiscovery: true,
		ClusterConfig: params.ClusterConfig{
			Enabled: true,
			// not necessarily with id, just valid multiaddr
			RendezvousNodes: []string{"/ip4/127.0.0.1/tcp/34012", "/ip4/127.0.0.1/tcp/34011"},
		},
		// use custom address to test the all possibilities
		AdvertiseAddr: "127.0.0.1",
	}
	n := New()
	require.NoError(t, n.StartWithOptions(&config, StartOptions{}))
	require.Nil(t, n.discovery)
	// start discovery manually
	require.NoError(t, n.StartDiscovery())
	require.NotNil(t, n.discovery)
	require.True(t, n.discovery.Running())
	require.IsType(t, &discovery.Rendezvous{}, n.discovery)
}

func TestStatusNodeDiscoverNode(t *testing.T) {
	config := params.NodeConfig{
		NoDiscovery: true,
		ListenAddr:  "127.0.0.1:0",
	}
	n := New()
	require.NoError(t, n.Start(&config))
	node, err := n.discoverNode()
	require.NoError(t, err)
	require.Equal(t, net.ParseIP("127.0.0.1").To4(), node.IP())

	config = params.NodeConfig{
		NoDiscovery:   true,
		AdvertiseAddr: "127.0.0.2",
		ListenAddr:    "127.0.0.1:0",
	}
	n = New()
	require.NoError(t, n.Start(&config))
	node, err = n.discoverNode()
	require.NoError(t, err)
	require.Equal(t, net.ParseIP("127.0.0.2").To4(), node.IP())
}

func TestChaosModeChangeRPCClientsUpstreamURL(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{
			"id": 1,
			"jsonrpc": "2.0",
			"result": "0x234234e22b9ffc2387e18636e0534534a3d0c56b0243567432453264c16e78a2adc"
		}`)
	}))
	defer ts.Close()

	config := params.NodeConfig{
		NoDiscovery: true,
		ListenAddr:  "127.0.0.1:0",
		UpstreamConfig: params.UpstreamRPCConfig{
			Enabled: true,
			URL:     ts.URL,
		},
	}
	n := New()
	require.NoError(t, n.Start(&config))
	defer func() { require.NoError(t, n.Stop()) }()
	require.NotNil(t, n.RPCClient())

	client := n.RPCClient()
	require.NotNil(t, client)

	var err error

	err = client.Call(nil, "net_version")
	require.NoError(t, err)

	// act
	err = n.ChaosModeChangeRPCClientsUpstreamURL(params.MainnetEthereumNetworkURL)
	require.NoError(t, err)

	// assert
	err = client.Call(nil, "net_version")
	require.EqualError(t, err, `500 Internal Server Error "500 Internal Server Error"`)
}
