package wakuv2

import (
	"context"
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/timesource"
)

func TestSetTopicInterest(t *testing.T) {
	w := &Waku{
		ctx:           context.TODO(),
		timesource:    timesource.Default(),
		topicInterest: make(map[string]TopicInterest),
	}

	peerID, err := peer.Decode("16Uiu2HAm3xVDaz6SRJ6kErwC21zBJEZjavVXg7VSkoWzaV1aMA3F")
	require.NoError(t, err)

	pubsubTopic1 := "topic1"
	contentTopics1 := []string{"A", "B", "C"}
	contentTopics1_1 := []string{"C", "D", "E", "F"}

	w.SetTopicsToVerifyForMissingMessages(peerID, pubsubTopic1, contentTopics1)

	storedTopicInterest, ok := w.topicInterest[pubsubTopic1]
	require.True(t, ok)
	require.Equal(t, storedTopicInterest.contentTopics, contentTopics1)
	require.Equal(t, storedTopicInterest.pubsubTopic, pubsubTopic1)

	w.SetTopicsToVerifyForMissingMessages(peerID, pubsubTopic1, contentTopics1_1)
	storedTopicInterest_2, ok := w.topicInterest[pubsubTopic1]
	require.True(t, ok)
	require.Equal(t, storedTopicInterest_2.contentTopics, contentTopics1_1)
	require.Equal(t, storedTopicInterest_2.pubsubTopic, pubsubTopic1)

	require.Error(t, storedTopicInterest.ctx.Err(), context.Canceled)
	require.NoError(t, w.topicInterest[pubsubTopic1].ctx.Err())

}
