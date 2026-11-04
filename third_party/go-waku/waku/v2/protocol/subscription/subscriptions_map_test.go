package subscription

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"google.golang.org/protobuf/proto"
)

const PUBSUB_TOPIC = "/test/topic"

func createPeerID(t *testing.T) peer.ID {
	peerId, err := test.RandPeerID()
	assert.NoError(t, err)
	return peerId
}

func TestSubscriptionMapAppend(t *testing.T) {
	fmap := NewSubscriptionMap(utils.Logger())
	peerID := createPeerID(t)
	contentTopics := protocol.NewContentTopicSet("ct1", "ct2")

	sub := fmap.NewSubscription(peerID, protocol.ContentFilter{PubsubTopic: PUBSUB_TOPIC, ContentTopics: contentTopics})
	_, found := sub.ContentFilter.ContentTopics["ct1"]
	require.True(t, found)
	_, found = sub.ContentFilter.ContentTopics["ct2"]
	require.True(t, found)
	require.False(t, sub.Closed)
	require.Equal(t, sub.PeerID, peerID)
	require.Equal(t, sub.ContentFilter.PubsubTopic, PUBSUB_TOPIC)

	sub.Add("ct3")
	_, found = sub.ContentFilter.ContentTopics["ct3"]
	require.True(t, found)

	sub.Remove("ct3")
	_, found = sub.ContentFilter.ContentTopics["ct3"]
	require.False(t, found)

	err := sub.Close()
	require.NoError(t, err)
	require.True(t, sub.Closed)
}

func TestSubscriptionClear(t *testing.T) {
	fmap := NewSubscriptionMap(utils.Logger())
	contentTopics := protocol.NewContentTopicSet("ct1", "ct2")

	var subscriptions = []*SubscriptionDetails{
		fmap.NewSubscription(createPeerID(t), protocol.ContentFilter{PubsubTopic: PUBSUB_TOPIC + "1", ContentTopics: contentTopics}),
		fmap.NewSubscription(createPeerID(t), protocol.ContentFilter{PubsubTopic: PUBSUB_TOPIC + "2", ContentTopics: contentTopics}),
		fmap.NewSubscription(createPeerID(t), protocol.ContentFilter{PubsubTopic: PUBSUB_TOPIC + "3", ContentTopics: contentTopics}),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	wg := sync.WaitGroup{}
	wg.Add(len(subscriptions))
	for _, s := range subscriptions {
		go func(s *SubscriptionDetails) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				t.Fail()
				return
			case <-s.C:
				return
			}
		}(s)
	}

	fmap.Clear()

	wg.Wait()

	require.True(t, subscriptions[0].Closed)
	require.True(t, subscriptions[1].Closed)
	require.True(t, subscriptions[2].Closed)
}

func TestSubscriptionsNotify(t *testing.T) {
	fmap := NewSubscriptionMap(utils.Logger())
	p1 := createPeerID(t)
	p2 := createPeerID(t)
	var subscriptions = []*SubscriptionDetails{
		fmap.NewSubscription(p1, protocol.ContentFilter{PubsubTopic: PUBSUB_TOPIC + "1", ContentTopics: protocol.NewContentTopicSet("ct1", "ct2")}),
		fmap.NewSubscription(p2, protocol.ContentFilter{PubsubTopic: PUBSUB_TOPIC + "1", ContentTopics: protocol.NewContentTopicSet("ct1")}),
		fmap.NewSubscription(p1, protocol.ContentFilter{PubsubTopic: PUBSUB_TOPIC + "2", ContentTopics: protocol.NewContentTopicSet("ct1", "ct2")}),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	successChan := make(chan struct{}, 10)
	wg := sync.WaitGroup{}

	successOnReceive := func(ctx context.Context, i int) {
		defer wg.Done()

		if subscriptions[i].Closed {
			successChan <- struct{}{}
			return
		}

		select {
		case <-ctx.Done():
			panic("should have failed1")
		case c := <-subscriptions[i].C:
			if c == nil {
				panic("should have failed2")
			}
			successChan <- struct{}{}
			return
		}
	}

	failOnReceive := func(ctx context.Context, i int) {
		defer wg.Done()

		if subscriptions[i].Closed {
			successChan <- struct{}{}
			return
		}

		select {
		case <-ctx.Done():
			successChan <- struct{}{}
			return
		case c := <-subscriptions[i].C:
			if c != nil {
				panic("should have failed")
			}
			successChan <- struct{}{}
			return
		}
	}

	wg.Add(3)
	go successOnReceive(ctx, 0)
	go successOnReceive(ctx, 1)
	go failOnReceive(ctx, 2)
	time.Sleep(200 * time.Millisecond)

	envTopic1Ct1 := protocol.NewEnvelope(tests.CreateWakuMessage("ct1", nil), 0, PUBSUB_TOPIC+"1")
	wg.Add(1)
	go func() {
		defer wg.Done()
		fmap.Notify(ctx, p1, envTopic1Ct1)
		fmap.Notify(ctx, p2, envTopic1Ct1)
	}()

	<-successChan
	<-successChan
	cancel()
	wg.Wait()
	<-successChan

	//////////////////////////////////////

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)

	wg.Add(3)
	go successOnReceive(ctx, 0)
	go failOnReceive(ctx, 1)
	go failOnReceive(ctx, 2)
	time.Sleep(200 * time.Millisecond)

	envTopic1Ct2 := protocol.NewEnvelope(tests.CreateWakuMessage("ct2", nil), 0, PUBSUB_TOPIC+"1")
	wg.Add(1)
	go func() {
		defer wg.Done()
		fmap.Notify(ctx, p1, envTopic1Ct2)
		fmap.Notify(ctx, p2, envTopic1Ct2)
	}()

	<-successChan
	cancel()
	wg.Wait()
	<-successChan
	<-successChan

	//////////////////////////////////////

	// Testing after closing the subscription

	subscriptions[0].Close()
	time.Sleep(200 * time.Millisecond)

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)

	wg.Add(3)
	go failOnReceive(ctx, 0)
	go successOnReceive(ctx, 1)
	go failOnReceive(ctx, 2)
	time.Sleep(200 * time.Millisecond)

	envTopic1Ct1_2 := protocol.NewEnvelope(tests.CreateWakuMessage("ct1", proto.Int64(1)), 1, PUBSUB_TOPIC+"1")

	wg.Add(1)
	go func() {
		defer wg.Done()
		fmap.Notify(ctx, p1, envTopic1Ct1_2)
		fmap.Notify(ctx, p2, envTopic1Ct1_2)
	}()

	<-successChan // One of these successes is for closing the subscription
	<-successChan
	cancel()
	wg.Wait()
	<-successChan
}

// TestSetClosingDoesNotHoldInnerLock verifies that SetClosing does not leave
// the SubscriptionDetails RWMutex held when the Closing channel has no ready
// receiver
func TestSetClosingDoesNotHoldInnerLock(t *testing.T) {
	fmap := NewSubscriptionMap(utils.Logger())
	peerID := createPeerID(t)
	sub := fmap.NewSubscription(peerID, protocol.ContentFilter{
		PubsubTopic:   PUBSUB_TOPIC,
		ContentTopics: protocol.NewContentTopicSet("ct1"),
	})

	// Intentionally do NOT spawn a receiver on sub.Closing — reproduces the
	// scenario where the api/filter multiplex goroutine or its downstream
	// apiSub.closing consumer is stalled (needing outer mapRef.Lock that
	// another goroutine holds as outer RLock).
	setClosingDone := make(chan struct{})
	go func() {
		sub.SetClosing()
		close(setClosingDone)
	}()

	// Give SetClosing time to reach the blocking send (if unpatched).
	time.Sleep(50 * time.Millisecond)

	// A parallel reader that exercises the real GetSubscriptionsForPeer ->
	// isPartOf path. isPartOf takes s.RLock() on the SubscriptionDetails.
	readerDone := make(chan []*SubscriptionDetails, 1)
	go func() {
		readerDone <- fmap.GetSubscriptionsForPeer(peerID, protocol.ContentFilter{})
	}()

	select {
	case subs := <-readerDone:
		require.Len(t, subs, 1)
	case <-time.After(time.Second):
		t.Fatal("reader blocked: SetClosing is holding SubscriptionDetails lock while sending on unbuffered Closing channel (deadlock)")
	}

	// After the fix, SetClosing itself should also complete — either because
	// the channel is buffered(1) and the send is instant, or because a select-
	// default drops the send when no one is reading. Either is acceptable.
	select {
	case <-setClosingDone:
	case <-time.After(time.Second):
		t.Fatal("SetClosing never returned without a receiver — inner lock or channel send is still blocking")
	}
}
