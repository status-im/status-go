package whisper

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/shhext"
	"github.com/status-im/status-go/t/utils"
	"github.com/stretchr/testify/require"
)

func createClientsAndServers(t *testing.T, clientCount, serverCount int) ([]*node.StatusNode, []*node.StatusNode) {
	clientConfig := &params.NodeConfig{
		MaxPendingPeers:      10,
		MaxPeers:             10,
		StatusServiceEnabled: true,
		NoDiscovery:          true,
		ListenAddr:           "127.0.0.1:0",
		StatusServiceConfig: &shhext.ServiceConfig{
			MailServerConfirmations: true,
			EnableConnectionManager: true,
			ConnectionTarget:        1,
			EnableLastUsedMonitor:   true,
		},
		WhisperConfig: params.WhisperConfig{
			Enabled: true,
		},
	}
	serverConfig := &params.NodeConfig{
		MaxPendingPeers: 10,
		MaxPeers:        10,
		NoDiscovery:     true,
		ListenAddr:      "127.0.0.1:0",
		WhisperConfig: params.WhisperConfig{
			Enabled: true,
		},
	}
	return createAndStartNodes(t, clientConfig, clientCount), createAndStartNodes(t, serverConfig, serverCount)
}

func createAndStartNodes(t *testing.T, config *params.NodeConfig, count int) []*node.StatusNode {
	nodes := make([]*node.StatusNode, count)
	for i := range nodes {
		nodes[i] = createAndStartNode(t, config)
	}
	return nodes
}

func createAndStartNode(t *testing.T, config *params.NodeConfig) *node.StatusNode {
	n := node.New()
	require.NoError(t, n.Start(config))
	return n
}

func TestConnectionManagerOnConnectionProblems(t *testing.T) {
	clients, servers := createClientsAndServers(t, 1, 2)
	client := clients[0]

	// get any connected peer
	connected := addNodesAndWaitForAnyConnected(t, client, servers)
	var notconnected enode.ID
	// stop a node which we got as a connection
	for _, srv := range servers {
		if connected == srv.Server().Self().ID() {
			require.NoError(t, srv.Stop())
		} else {
			notconnected = srv.Server().Self().ID()
		}
	}
	require.NotEqual(t, enode.ID{}, notconnected)
	waitEstablishedConnectionWithPeer(t, client, time.Second, notconnected)
}

func TestConnectionManagerOnMailserverExpiry(t *testing.T) {
	clients, servers := createClientsAndServers(t, 2, 1)
	mainClient := clients[0]
	fakeServer := clients[1]
	server := servers[0]

	// add a connection with fake mail server. do not include actual mail server in order
	// to deterministically connect with fake one
	connected := addNodesAndWaitForAnyConnected(t, mainClient, []*node.StatusNode{fakeServer})
	require.Equal(t, fakeServer.Server().Self().ID(), connected)

	shhextService, err := mainClient.ShhExtService()
	require.NoError(t, err)

	// update mail servers so that manager can connect with actual mail server if fake one will expire
	require.NoError(t, shhextService.UpdateMailservers([]*enode.Node{fakeServer.Server().Self(), server.Server().Self()}))
	shhapi := shhext.NewPublicAPI(shhextService)
	_, err = shhapi.RequestMessages(context.TODO(), shhext.MessagesRequest{
		Timeout: 1,
	})
	require.NoError(t, err)
	waitEstablishedConnectionWithPeer(t, mainClient, 2*time.Second, server.Server().Self().ID())
}

func addNodesAndWaitForAnyConnected(t *testing.T, client *node.StatusNode, nodes []*node.StatusNode) enode.ID {
	shhextService, err := client.ShhExtService()
	require.NoError(t, err)
	serversEnodes := []*enode.Node{}
	for _, n := range nodes {
		serversEnodes = append(serversEnodes, n.Server().Self())
	}
	events := make(chan *p2p.PeerEvent, 10)
	sub := client.Server().SubscribeEvents(events)
	defer sub.Unsubscribe()
	require.NoError(t, err)
	require.NoError(t, shhextService.UpdateMailservers(serversEnodes))
	// get any first connected peer
	return waitForAnyConnected(t, events, 2*time.Second)
}

func waitForAnyConnected(t *testing.T, events chan *p2p.PeerEvent, timeout time.Duration) enode.ID {
	timer := time.After(timeout)
	for {
		select {
		case <-timer:
			require.FailNow(t, "failed waiting for connection after mailservers were updated")
		case ev := <-events:
			if ev.Type == p2p.PeerEventTypeAdd {
				return ev.Peer
			}
		}
	}
}

func waitDisconnected(t *testing.T, node *node.StatusNode, timeout time.Duration, nodeID enode.ID) {
	server := node.Server()
	events := make(chan *p2p.PeerEvent, 2)
	server.SubscribeEvents(events)
	timer := time.After(timeout)
	for {
		select {
		case <-timer:
			require.FailNowf(t, "error by timeout", "waiting for %s to be dropped", nodeID.String())
		case ev := <-events:
			if ev.Type != p2p.PeerEventTypeDrop && ev.Peer != nodeID {
				continue
			}

		}
	}
}

func waitEstablishedConnectionWithPeer(t *testing.T, client *node.StatusNode, timeout time.Duration, nodeID enode.ID) {
	require.NoError(t, utils.Eventually(func() error {
		peers := client.Server().Peers()
		if len(peers) != 1 {
			return fmt.Errorf("too many peers")
		}
		if peers[0].ID() != nodeID {
			return fmt.Errorf("connection establish with wrong peer. expect %s got %s", nodeID, peers[0].ID())
		}
		return nil
	}, timeout, 200*time.Millisecond))
}
