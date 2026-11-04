package node

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/peermanager"
	"github.com/waku-org/go-waku/waku/v2/peerstore"
	"github.com/waku-org/go-waku/waku/v2/protocol"
)

const pubsubTopic = "/waku/2/rs/16/1000"

func goCheckConnectedness(t *testing.T, wg *sync.WaitGroup, topicHealthStatusChan chan peermanager.TopicHealthStatus,
	healthStatus peermanager.TopicHealth) {
	wg.Add(1)
	go checkConnectedness(t, wg, topicHealthStatusChan, healthStatus)
}

func checkConnectedness(t *testing.T, wg *sync.WaitGroup, topicHealthStatusChan chan peermanager.TopicHealthStatus,
	healthStatus peermanager.TopicHealth) {
	defer wg.Done()

	timeout := time.After(5 * time.Second)

	select {
	case topicHealthStatus := <-topicHealthStatusChan:
		require.Equal(t, healthStatus, topicHealthStatus.Health)
		t.Log("received health status update ", topicHealthStatus.Health, "expected is ", healthStatus)
		return
	case <-timeout:
		require.Fail(t, "health status should have changed")
	}
}

func TestConnectionStatusChanges(t *testing.T) {
	t.Skip("TODO: figure out how the mesh is managed in go-libp2p")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	topicHealthStatusChan := make(chan peermanager.TopicHealthStatus, 100)

	// Node1: Only Relay
	hostAddr1, err := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	require.NoError(t, err)
	node1, err := New(
		WithHostAddress(hostAddr1),
		WithWakuRelay(),
		WithClusterID(16),
		WithTopicHealthStatusChannel(topicHealthStatusChan),
	)
	require.NoError(t, err)
	err = node1.Start(ctx)
	require.NoError(t, err)
	_, err = node1.Relay().Subscribe(ctx, protocol.NewContentFilter(pubsubTopic))
	require.NoError(t, err)

	// Node2: Relay
	node2 := startNodeAndSubscribe(t, ctx)

	// Node3: Relay + Store
	node3 := startNodeAndSubscribe(t, ctx)

	// Node4: Relay
	node4 := startNodeAndSubscribe(t, ctx)

	// Node5: Relay
	node5 := startNodeAndSubscribe(t, ctx)

	var wg sync.WaitGroup

	goCheckConnectedness(t, &wg, topicHealthStatusChan, peermanager.MinimallyHealthy)

	node1.AddDiscoveredPeer(node2.host.ID(), node2.ListenAddresses(), peerstore.Static, []string{pubsubTopic}, nil, true)

	wg.Wait()

	err = node1.DialPeer(ctx, node3.ListenAddresses()[0].String())
	require.NoError(t, err)

	err = node1.DialPeer(ctx, node4.ListenAddresses()[0].String())
	require.NoError(t, err)
	goCheckConnectedness(t, &wg, topicHealthStatusChan, peermanager.SufficientlyHealthy)

	err = node1.DialPeer(ctx, node5.ListenAddresses()[0].String())
	require.NoError(t, err)

	wg.Wait()

	goCheckConnectedness(t, &wg, topicHealthStatusChan, peermanager.MinimallyHealthy)

	node3.Stop()

	wg.Wait()

	goCheckConnectedness(t, &wg, topicHealthStatusChan, peermanager.UnHealthy)

	err = node1.ClosePeerById(node2.Host().ID())
	require.NoError(t, err)

	node4.Stop()
	node5.Stop()

	wg.Wait()

	goCheckConnectedness(t, &wg, topicHealthStatusChan, peermanager.MinimallyHealthy)

	err = node1.DialPeerByID(ctx, node2.Host().ID())
	require.NoError(t, err)
	wg.Wait()
}

func startNodeAndSubscribe(t *testing.T, ctx context.Context) *WakuNode {
	hostAddr, err := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	require.NoError(t, err)
	node, err := New(
		WithHostAddress(hostAddr),
		WithWakuRelay(),
		WithClusterID(16),
	)
	require.NoError(t, err)
	err = node.Start(ctx)
	require.NoError(t, err)
	_, err = node.Relay().Subscribe(ctx, protocol.NewContentFilter(pubsubTopic))
	require.NoError(t, err)
	return node
}
