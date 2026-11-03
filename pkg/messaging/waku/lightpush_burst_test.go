package waku

import (
	"context"
	"crypto/rand"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/status-im/status-go/internal/panics"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/lightpush"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"golang.org/x/time/rate"
)

func TestLightpushServerBurstAllowsAllGroupRecipients(t *testing.T) {
	const recipientCount = 19

	tests := []struct {
		name          string
		burst         int
		expectedSends int
	}{
		{name: "burst 10 rejects part of 20-member group send", burst: 10, expectedSends: 10},
		{name: "configured burst 20 accepts all recipients", burst: lightpushServerRateLimitBurst, expectedSends: recipientCount},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			successCount, rateLimitedCount := sendLightpushBurst(t, tt.burst, recipientCount)
			require.Equal(t, tt.expectedSends, successCount)
			require.Equal(t, recipientCount-tt.expectedSends, rateLimitedCount)
		})
	}
}

func sendLightpushBurst(t *testing.T, burst, sendCount int) (successCount, rateLimitedCount int) {
	t.Helper()

	const pubsubTopic = "/waku/2/lightpush/burst-test"
	const contentTopic = "/status/1/lightpush-burst/proto"

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	relayNode1, subscription1, host1 := newLightpushBurstRelay(t, pubsubTopic)
	defer relayNode1.Stop()
	defer subscription1.Unsubscribe()

	relayNode2, subscription2, host2 := newLightpushBurstRelay(t, pubsubTopic)
	defer relayNode2.Stop()
	defer subscription2.Unsubscribe()

	server := lightpush.NewWakuLightPush(
		relayNode2,
		nil,
		prometheus.DefaultRegisterer,
		utils.Logger(),
		// A zero refill rate isolates the burst allowance from request timing.
		lightpush.WithRateLimiter(rate.Limit(0), burst),
	)
	server.SetHost(host2)
	require.NoError(t, server.Start(ctx))
	defer server.Stop()

	host2.Peerstore().AddAddr(host1.ID(), tests.GetHostAddress(host1), peerstore.PermanentAddrTTL)
	require.NoError(t, host2.Peerstore().AddProtocols(host1.ID(), relay.WakuRelayID_v200))
	require.NoError(t, host2.Connect(ctx, host2.Peerstore().PeerInfo(host1.ID())))

	require.Eventually(t, func() bool {
		return relayNode2.EnoughPeersToPublishToTopic(pubsubTopic)
	}, 5*time.Second, 50*time.Millisecond)

	clientHost, err := tests.MakeHost(ctx, 0, rand.Reader)
	require.NoError(t, err)
	defer clientHost.Close()

	client := lightpush.NewWakuLightPush(nil, nil, prometheus.DefaultRegisterer, utils.Logger())
	client.SetHost(clientHost)
	clientHost.Peerstore().AddAddr(host2.ID(), tests.GetHostAddress(host2), peerstore.PermanentAddrTTL)
	require.NoError(t, clientHost.Peerstore().AddProtocols(host2.ID(), lightpush.LightPushID_v20beta1))

	results := make(chan error, sendCount)
	var wg sync.WaitGroup
	for i := 0; i < sendCount; i++ {
		wg.Add(1)
		go func(i int) {
			defer panics.LogOnPanic()
			defer wg.Done()
			message := tests.CreateWakuMessage(contentTopic, utils.GetUnixEpoch(), strconv.Itoa(i))
			_, err := client.Publish(ctx, message, lightpush.WithPubSubTopic(pubsubTopic), lightpush.WithPeer(host2.ID()))
			results <- err
		}(i)
	}
	wg.Wait()
	close(results)

	for err := range results {
		switch {
		case err == nil:
			successCount++
		case isLightpushRateLimited(err):
			rateLimitedCount++
		default:
			require.NoError(t, err)
		}
	}

	return successCount, rateLimitedCount
}

func newLightpushBurstRelay(t *testing.T, pubsubTopic string) (*relay.WakuRelay, *relay.Subscription, host.Host) {
	t.Helper()

	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)

	nodeHost, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)

	broadcaster := relay.NewBroadcaster(10)
	require.NoError(t, broadcaster.Start(context.Background()))

	relayNode := relay.NewWakuRelay(broadcaster, 0, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	relayNode.SetHost(nodeHost)
	require.NoError(t, relayNode.Start(context.Background()))

	subscriptions, err := relayNode.Subscribe(context.Background(), protocol.NewContentFilter(pubsubTopic))
	require.NoError(t, err)
	require.Len(t, subscriptions, 1)

	return relayNode, subscriptions[0], nodeHost
}
