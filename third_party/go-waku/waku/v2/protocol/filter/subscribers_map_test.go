package filter

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const PUBSUB_TOPIC = "/test/topic"

func createPeerID(t *testing.T) peer.ID {
	peerId, err := test.RandPeerID()
	assert.NoError(t, err)
	return peerId
}

func firstSubscriber(subs *SubscribersMap, pubsubTopic string, contentTopic string) peer.ID {
	for sub := range subs.Items(pubsubTopic, contentTopic) {
		return sub
	}
	return ""
}

func TestAppend(t *testing.T) {
	subs := NewSubscribersMap(5 * time.Second)
	peerId := createPeerID(t)

	subs.Set(peerId, PUBSUB_TOPIC, []string{"topic1"})

	sub := firstSubscriber(subs, PUBSUB_TOPIC, "topic1")
	assert.NotEmpty(t, sub)

	// Adding to existing peer
	subs.Set(peerId, PUBSUB_TOPIC, []string{"topic2"})

	sub = firstSubscriber(subs, PUBSUB_TOPIC, "topic2")
	assert.NotEmpty(t, sub)

	subs.Set(peerId, PUBSUB_TOPIC+"2", []string{"topic1"})

	sub = firstSubscriber(subs, PUBSUB_TOPIC+"2", "topic1")
	assert.NotEmpty(t, sub)

	sub = firstSubscriber(subs, PUBSUB_TOPIC+"2", "topic2")
	assert.Empty(t, sub)
}

func TestRemove(t *testing.T) {
	subs := NewSubscribersMap(5 * time.Second)
	peerId := createPeerID(t)

	subs.Set(peerId, PUBSUB_TOPIC+"1", []string{"topic1", "topic2"})
	subs.Set(peerId, PUBSUB_TOPIC+"2", []string{"topic1"})

	err := subs.DeleteAll(peerId)
	assert.Empty(t, err)

	sub := firstSubscriber(subs, PUBSUB_TOPIC+"1", "topic1")
	assert.Empty(t, sub)

	sub = firstSubscriber(subs, PUBSUB_TOPIC+"1", "topic2")
	assert.Empty(t, sub)

	sub = firstSubscriber(subs, PUBSUB_TOPIC+"2", "topic1")
	assert.Empty(t, sub)

	assert.False(t, subs.Has(peerId))

	_, found := subs.Get(peerId)
	assert.False(t, found)

	_, ok := subs.items[peerId]
	assert.False(t, ok)
}

func TestRemovePartial(t *testing.T) {
	subs := NewSubscribersMap(5 * time.Second)
	peerId := createPeerID(t)

	subs.Set(peerId, PUBSUB_TOPIC, []string{"topic1", "topic2"})
	err := subs.Delete(peerId, PUBSUB_TOPIC, []string{"topic1"})
	require.NoError(t, err)

	sub := firstSubscriber(subs, PUBSUB_TOPIC, "topic2")
	assert.NotEmpty(t, sub)
}

func TestRemoveBogus(t *testing.T) {
	subs := NewSubscribersMap(5 * time.Second)
	peerId := createPeerID(t)

	subs.Set(peerId, PUBSUB_TOPIC, []string{"topic1", "topic2"})
	err := subs.Delete(peerId, PUBSUB_TOPIC, []string{"does not exist", "topic1"})
	require.NoError(t, err)

	sub := firstSubscriber(subs, PUBSUB_TOPIC, "topic1")
	assert.Empty(t, sub)
	sub = firstSubscriber(subs, PUBSUB_TOPIC, "does not exist")
	assert.Empty(t, sub)

	err = subs.Delete(peerId, "DOES_NOT_EXIST", []string{"topic1"})
	require.Error(t, err)
}

// TestCleanup verifies the idle-subscription cleanup lifecycle: a subscriber
// that keeps getting refreshed (e.g. via pings) survives the periodic cleanup
// while active, and is removed only once it goes idle past the timeout. This
// also guards against a regression where the idle check was inverted, deleting
// active subscribers while keeping idle ones forever.
func TestCleanup(t *testing.T) {
	subs := NewSubscribersMap(1 * time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go subs.cleanUp(ctx, 100*time.Millisecond)

	peerId := createPeerID(t)
	subs.Set(peerId, PUBSUB_TOPIC, []string{"topic1", "topic2"})

	hasSubs := subs.Has(peerId)
	require.True(t, hasSubs)

	_, exists := subs.Get(peerId)
	require.True(t, exists)

	// Keep the subscriber active for well over the timeout by refreshing it
	// periodically. It must never be cleaned up while active.
	for i := 0; i < 15; i++ {
		time.Sleep(100 * time.Millisecond)
		subs.Refresh(peerId)
		require.True(t, subs.Has(peerId), "active subscriber was removed while being refreshed")
	}

	// Stop refreshing; once idle past the timeout it must be removed.
	require.Eventually(t, func() bool {
		return !subs.Has(peerId)
	}, 3*time.Second, 100*time.Millisecond, "idle subscriber was not cleaned up")

	_, exists = subs.Get(peerId)
	require.False(t, exists)
}
