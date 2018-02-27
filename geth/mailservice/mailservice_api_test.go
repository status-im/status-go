package mailservice

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	mailServerPeer = "enode://b7e65e1bedc2499ee6cbd806945af5e7df0e59e4070c96821570bd581473eade24a489f5ec95d060c0db118c879403ab88d827d3766978f28708989d35474f87@[::]:51920"
)

func TestRequestMessagesDefaults(t *testing.T) {
	r := MessagesRequest{}
	setMessagesRequestDefaults(&r)
	require.NotZero(t, r.From)
	require.InEpsilon(t, uint32(time.Now().UTC().Unix()), r.To, 1.0)
}

func TestRequestMessagesNoPeers(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := NewMockServiceProvider(ctrl)
	service := New(provider)
	require.NoError(t, service.Start(nil))
	api := NewPublicAPI(service)
	defer func() { require.NoError(t, service.Stop()) }()
	shh := whisper.New(nil)
	// Node is ephemeral (only in memory).
	nodeA, err := node.New(&node.Config{
		NoUSB: true,
	})
	require.NoError(t, err)
	require.NoError(t, nodeA.Start())

	// without peers
	provider.EXPECT().WhisperService().Return(shh, nil)
	provider.EXPECT().Server().Return(nodeA.Server(), nil).AnyTimes()
	result, err := api.RequestMessages(context.TODO(), MessagesRequest{})
	assert.False(t, result)
	assert.EqualError(t, err, "no mailservers are available")
	require.NoError(t, nodeA.Stop())
}

func TestRequestMessagesFailedToAddPeer(t *testing.T) {
	mailNode, err := discover.ParseNode(mailServerPeer)
	require.NoError(t, err)

	shh := whisper.New(nil)
	ctrl := gomock.NewController(t)
	provider := NewMockServiceProvider(ctrl)
	service := New(provider)
	require.NoError(t, service.Start(nil))
	api := NewPublicAPI(service)
	defer func() { require.NoError(t, service.Stop()) }()

	// with peers but peer is not reachable
	nodeA, err := node.New(&node.Config{
		NoUSB: true,
		P2P:   p2p.Config{TrustedNodes: []*discover.Node{mailNode}},
	})
	require.NoError(t, err)
	require.NoError(t, nodeA.Start())
	provider.EXPECT().WhisperService().Return(shh, nil)
	provider.EXPECT().Server().Return(nodeA.Server(), nil).AnyTimes()
	result, err := api.RequestMessages(context.TODO(), MessagesRequest{})
	assert.False(t, result)
	assert.Contains(t, err.Error(), "failed to add a peer")
	require.NoError(t, nodeA.Stop())
}

func TestRequestMessagesSuccess(t *testing.T) {
	// TODO(adam): next step would be to run a successful test, however,
	// it requires to set up emepheral nodes that can discover each other
	// without syncing blockchain. It requires a bit research how to do that.
	t.Skip()
}

func TestGarbageCollector(t *testing.T) {
	port := 30303
	servers := make([]*p2p.Server, 0, 2)
	setupFailed := false
	for i := 0; i < cap(servers); i++ {
		key, err := crypto.GenerateKey()
		assert.NoError(t, err)
		shh := whisper.New(nil)
		name := common.MakeName("test", "1.0")
		server := &p2p.Server{
			Config: p2p.Config{
				MaxPeers:   10,
				Name:       name,
				ListenAddr: fmt.Sprintf(":%d", port),
				PrivateKey: key,
				Protocols:  shh.Protocols(),
			},
		}
		err = server.Start()
		assert.NoError(t, err)
		defer server.Stop()
		servers = append(servers, server)
		port++
		setupFailed = err != nil
	}
	if setupFailed {
		require.Fail(t, "test failed while preparing p2p servers")
	}
	servers[0].AddPeer(servers[1].Self())
	// let server to dial a peer
	time.Sleep(200 * time.Millisecond)
	for i := range servers {
		require.Equal(t, 1, servers[i].PeerCount())
	}

	ctrl := gomock.NewController(t)
	provider := NewMockServiceProvider(ctrl)
	service := New(provider)
	require.NoError(t, service.Start(nil))
	api := &PublicAPI{
		service:           service,
		newConnectedPeers: make(chan *discover.Node, 1),
	}
	go api.runTrustedPeersGC(200*time.Millisecond, 100*time.Millisecond)
	defer func() { require.NoError(t, service.Stop()) }()
	provider.EXPECT().Server().Return(servers[0], nil).AnyTimes()
	events := make(chan *p2p.PeerEvent, 1)
	sub := servers[0].SubscribeEvents(events)
	defer sub.Unsubscribe()
	// update gc with a peer
	api.newConnectedPeers <- servers[1].Self()
	// let gc do the work
	select {
	case ev := <-events:
		assert.Equal(t, p2p.PeerEventTypeDrop, ev.Type)
		assert.Equal(t, servers[1].Self().ID, ev.Peer)
	case <-time.After(300 * time.Millisecond):
		assert.Fail(t, "timeout out waiting for peer removal")
	}
}
