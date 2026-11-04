package protocol

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestContentTopicAndSharding(t *testing.T) {
	ct, err := NewContentTopic("waku", "2", "test", "proto")
	require.NoError(t, err)
	require.Equal(t, ct.String(), "/waku/2/test/proto")

	_, err = StringToContentTopic("/waku/-1/a/b")
	require.NoError(t, err)

	_, err = StringToContentTopic("waku/1/a/b")
	require.Error(t, err, ErrInvalidFormat)

	_, err = StringToContentTopic("////")
	require.Error(t, err, ErrInvalidFormat)

	_, err = StringToContentTopic("/waku/1/a")
	require.Error(t, err, ErrInvalidFormat)

	ct2, err := StringToContentTopic("/waku/2/test/proto")
	require.NoError(t, err)
	require.Equal(t, ct.String(), ct2.String())
	require.True(t, ct.Equal(ct2))

	ct3, err := NewContentTopic("waku", "2a", "test2", "proto")
	require.NoError(t, err)
	require.False(t, ct.Equal(ct3))

	ct4, err := StringToContentTopic("/0/toychat/2/huilong/proto")
	require.NoError(t, err)
	require.Equal(t, ct4.Generation, 0)

	ct6, err := StringToContentTopic("/toychat/2/huilong/proto")
	require.NoError(t, err)

	nsPubSubT1 := GetShardFromContentTopic(ct6, GenerationZeroShardsCount)
	require.Equal(t, NewStaticShardingPubsubTopic(ClusterIndex, 3), nsPubSubT1)

	_, err = StringToContentTopic("/abc/toychat/2/huilong/proto")
	require.Error(t, err, ErrInvalidGeneration)

	_, err = StringToContentTopic("/1/toychat/2/huilong/proto")
	require.Error(t, err, ErrInvalidGeneration)

	ct5, err := NewContentTopic("waku", "2b", "test2", "proto", WithGeneration(0))
	require.NoError(t, err)
	require.Equal(t, ct5.Generation, 0)
}

func randomContentTopic() (ContentTopic, error) {
	var app = ""
	const WordLength = 5
	rand.New(rand.NewSource(time.Now().Unix()))

	//Generate a random character between lowercase a to z
	for i := 0; i < WordLength; i++ {
		randomChar := 'a' + rune(rand.Intn(26))
		app = app + string(randomChar)
	}
	version := "1"

	var name = ""

	for i := 0; i < WordLength; i++ {
		randomChar := 'a' + rune(rand.Intn(26))
		name = name + string(randomChar)
	}
	var enc = "proto"

	return NewContentTopic(app, version, name, enc)
}

func TestShardChoiceSimulation(t *testing.T) {
	//Given
	var topics []ContentTopic
	for i := 0; i < 100000; i++ {
		ct, err := randomContentTopic()
		require.NoError(t, err)
		topics = append(topics, ct)
	}

	var counts [GenerationZeroShardsCount]int

	// When
	for _, topic := range topics {
		pubsub := GetShardFromContentTopic(topic, GenerationZeroShardsCount)
		counts[pubsub.Shard()]++
	}

	t.Logf("Total number of topics simulated %d", len(topics))
	for i := 0; i < GenerationZeroShardsCount; i++ {
		t.Logf("Topics assigned to shard %d is %d", i, counts[i])
	}

	// Then
	for i := 1; i < GenerationZeroShardsCount; i++ {
		//t.Logf("float64(counts[%d]) %f float64(counts[%d]) %f", i-1, float64(counts[i-1]), i, float64(counts[i]))
		if float64(counts[i-1]) <= (float64(counts[i])*1.05) &&
			float64(counts[i]) <= (float64(counts[i-1])*1.05) &&
			float64(counts[i-1]) >= (float64(counts[i])*0.95) &&
			float64(counts[i]) >= (float64(counts[i-1])*0.95) {
			t.Logf("Shard choice simulation successful")
		} else {
			t.FailNow()
		}
	}
}

func TestShardPubsubTopic(t *testing.T) {
	{ // not waku topci
		topic := "/waku/1/2/3"
		_, err := ToWakuPubsubTopic(topic)
		require.Error(t, ErrNotWakuPubsubTopic, err)
	}

	{ // check default pubsub topic
		topic := defaultPubsubTopic
		wakuTopic, err := ToWakuPubsubTopic(topic)
		require.NoError(t, err)
		require.Equal(t, defaultPubsubTopic, wakuTopic.String())
	}

	{ // check behavior of waku topic
		topic := "/waku/2/rs/16/42"
		wakuTopic, err := ToWakuPubsubTopic(topic)
		require.NoError(t, err)
		require.Equal(t, topic, wakuTopic.String())
		require.Equal(t, uint16(16), wakuTopic.(StaticShardingPubsubTopic).Cluster())
		require.Equal(t, uint16(42), wakuTopic.(StaticShardingPubsubTopic).Shard())
	}

	{ // check if shard pubtopic checks for prefix
		topic := "/waku/1/rs/16/42"
		err := (&StaticShardingPubsubTopic{}).Parse(topic)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidShardedTopicPrefix)
	}

	{ // check if cluster/index is missing
		topic := "/waku/2/rs//02"
		_, err := ToWakuPubsubTopic(topic)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrMissingClusterIndex)

		topic = "/waku/2/rs/1/"
		_, err = ToWakuPubsubTopic(topic)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrMissingShardNumber)
	}

	{ // check if the cluster/index are number
		topic := "/waku/2/rs/xx/77"
		_, err := ToWakuPubsubTopic(topic)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidNumberFormat)
	}

}
