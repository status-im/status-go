package wakuv2

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
	"golang.org/x/time/rate"
	"google.golang.org/protobuf/proto"

	"github.com/status-im/status-go/wakuv2/common"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
)

func TestThrottle(t *testing.T) {
	// Configuration for the first Waku node
	config1 := &Config{
		Port:                0,
		UseThrottledPublish: true,
	}

	// Start the first Waku node
	w1, err := New(nil, "", config1, nil, nil, nil, nil, nil)
	// Adding rate limiter back, since it's disabled by default in test context
	w1.limiter = rate.NewLimiter(3, 5)

	require.NoError(t, err)
	require.NoError(t, w1.Start())
	defer func() {
		require.NoError(t, w1.Stop())
	}()

	filter := &common.Filter{
		Messages:      common.NewMemoryMessageStore(),
		PubsubTopic:   config1.DefaultShardPubsubTopic,
		ContentTopics: common.NewTopicSetFromBytes([][]byte{{1, 2, 3, 4}}),
	}

	config2 := &Config{
		Port:                0,
		UseThrottledPublish: true,
	}
	w2, err := New(nil, "", config2, nil, nil, nil, nil, nil)
	// Adding rate limiter back, since it's disabled by default in test context
	w2.limiter = rate.NewLimiter(3, 5)

	require.NoError(t, err)
	require.NoError(t, w2.Start())

	_, err = w2.Subscribe(filter)
	require.NoError(t, err)

	w2EnvelopeCh := make(chan common.EnvelopeEvent, 100)
	w2.SubscribeEnvelopeEvents(w2EnvelopeCh)
	defer func() {
		require.NoError(t, w2.Stop())
		close(w2EnvelopeCh)
	}()

	// Connect the two nodes directly
	peer2Addr := w2.node.ListenAddresses()[0].String()
	err = w1.node.DialPeer(context.Background(), peer2Addr)
	require.NoError(t, err)

	time.Sleep(2 * time.Second) // Wait for mesh to form

	wg := sync.WaitGroup{}

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			msgTimestamp := w1.CurrentTime().UnixNano()
			contentTopic := maps.Keys(filter.ContentTopics)[0]
			_, err = w1.Send(config1.DefaultShardPubsubTopic, &pb.WakuMessage{
				Payload:      []byte{1, 2, 3, 4, 5},
				ContentTopic: contentTopic.ContentTopic(),
				Version:      proto.Uint32(0),
				Timestamp:    &msgTimestamp,
			}, nil)
			require.NoError(t, err)
			waitForEnvelope(t, contentTopic.ContentTopic(), w2EnvelopeCh)
		}()
	}

	wg.Wait()
}
