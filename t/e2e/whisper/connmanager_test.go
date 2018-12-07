package whisper

import (
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

func TestConnectionManager(t *testing.T) {
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

	client := node.New()
	require.NoError(t, client.Start(clientConfig))

	servers := [2]*node.StatusNode{node.New(), node.New()}
	for _, srv := range servers {
		require.NoError(t, srv.Start(serverConfig))
	}

	serversEnodes := []*enode.Node{}
	for _, srv := range servers {
		serversEnodes = append(serversEnodes, srv.Server().Self())
	}

	events := make(chan *p2p.PeerEvent, 10)
	sub := client.Server().SubscribeEvents(events)
	defer sub.Unsubscribe()
	shhextService, err := client.ShhExtService()
	require.NoError(t, err)
	require.NoError(t, shhextService.UpdateMailservers(serversEnodes))

	// get any first connected peer
	connected := waitForAnyConnected(t, events, 2*time.Second)
	var notconnected enode.ID
	for _, srv := range servers {
		if connected == srv.Server().Self().ID() {
			require.NoError(t, srv.Stop())
		} else {
			notconnected = srv.Server().Self().ID()
		}
	}
	require.NotEqual(t, enode.ID{}, notconnected)
	require.NoError(t, utils.Eventually(func() error {
		peers := client.Server().Peers()
		if len(peers) != 1 {
			return fmt.Errorf("too many peers")
		}
		if peers[0].ID() != notconnected {
			return fmt.Errorf("connection establish with wrong peer. expect %s got %s", notconnected, peers[0].ID())
		}
		return nil
	}, 2*time.Second, 200*time.Millisecond))
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
