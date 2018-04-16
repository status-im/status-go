package node

import (
	"errors"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/les"
	gethnode "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"

	"github.com/status-im/status-go/geth/params"
	"github.com/stretchr/testify/require"
)

func TestStatusNodeStart(t *testing.T) {
	var err error

	config := params.NodeConfig{}
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
	require.NoError(t, n.Start(&config))

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
	require.EqualError(t, n.Start(&config), ErrNodeRunning.Error())

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

func TestStatusNodeServiceGetters(t *testing.T) {
	config := params.NodeConfig{
		WhisperConfig: &params.WhisperConfig{
			Enabled: true,
		},
		LightEthConfig: &params.LightEthConfig{
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
	})
	require.NoError(t, err)
	require.NoError(t, peer.Start())
	defer func() {
		require.NoError(t, peer.Stop())
	}()
	peerURL := peer.Server().Self().String()

	n := New()

	// checks before node is started
	require.EqualError(t, n.AddPeer(peerURL), ErrNoRunningNode.Error())

	// start status node
	config := params.NodeConfig{
		MaxPeers: math.MaxInt32,
	}
	require.NoError(t, n.Start(&config))
	defer func() {
		require.NoError(t, n.Stop())
	}()

	errCh := waitForPeerAsync(n, peerURL, time.Second*5)

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
	})
	require.NoError(t, err)
	require.NoError(t, peer.Start())
	defer func() {
		require.NoError(t, peer.Stop())
	}()
	peerURL := peer.Server().Self().String()

	n := New()

	var errCh <-chan error

	// checks before node is started
	require.EqualError(t, n.ReconnectStaticPeers(), ErrNoRunningNode.Error())

	// start status node
	config := params.NodeConfig{
		MaxPeers: math.MaxInt32,
		ClusterConfig: &params.ClusterConfig{
			Enabled:     true,
			StaticNodes: []string{peerURL},
		},
	}
	errCh = waitForPeerAsync(n, peerURL, time.Second*30)
	require.NoError(t, n.Start(&config))
	defer func() {
		require.NoError(t, n.Stop())
	}()

	// checks after node is started
	require.NoError(t, <-errCh)
	require.Equal(t, 1, n.PeerCount())

	// reconnect static peers
	// it takes at least 30 seconds to bring back previously connected peer
	errCh = waitForPeerAsync(n, peerURL, time.Second*60)
	require.NoError(t, n.ReconnectStaticPeers(), ErrNoRunningNode.Error())
	require.NoError(t, <-errCh)
	require.Equal(t, 1, n.PeerCount())
}

func waitForPeer(node *StatusNode, peerURL string, timeout time.Duration) error {
	if !node.IsRunning() {
		return ErrNoRunningNode
	}

	parsedPeer, err := discover.ParseNode(peerURL)
	if err != nil {
		return err
	}

	server := node.GethNode().Server()
	ch := make(chan *p2p.PeerEvent)
	subscription := server.SubscribeEvents(ch)
	defer subscription.Unsubscribe()

	for {
		select {
		case ev := <-ch:
			if ev.Type == p2p.PeerEventTypeAdd && ev.Peer == parsedPeer.ID {
				return nil
			}
		case err := <-subscription.Err():
			if err != nil {
				return err
			}
		case <-time.After(timeout):
			// it may happen that the peer is already connected
			// but even was not received
			for _, p := range node.GethNode().Server().Peers() {
				if p.ID() == parsedPeer.ID {
					return nil
				}
			}

			return errors.New("wait for peer: timeout")
		}
	}
}

func waitForPeerAsync(node *StatusNode, peerURL string, timeout time.Duration) <-chan error {
	errCh := make(chan error)
	go func() {
		errCh <- waitForPeer(node, peerURL, timeout)
	}()

	return errCh
}
