package shhext

import (
	"fmt"
	"testing"
	"time"

	whisper "github.com/status-im/whisper/whisperv6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessagesRequest_setDefaults(t *testing.T) {
	daysAgo := func(now time.Time, days int) uint32 {
		return uint32(now.UTC().Add(-24 * time.Hour * time.Duration(days)).Unix())
	}

	tnow := time.Now()
	now := uint32(tnow.UTC().Unix())
	yesterday := daysAgo(tnow, 1)

	scenarios := []struct {
		given    *MessagesRequest
		expected *MessagesRequest
	}{
		{
			&MessagesRequest{From: 0, To: 0},
			&MessagesRequest{From: yesterday, To: now, Timeout: defaultRequestTimeout},
		},
		{
			&MessagesRequest{From: 1, To: 0},
			&MessagesRequest{From: uint32(1), To: now, Timeout: defaultRequestTimeout},
		},
		{
			&MessagesRequest{From: 0, To: yesterday},
			&MessagesRequest{From: daysAgo(tnow, 2), To: yesterday, Timeout: defaultRequestTimeout},
		},
		// 100 - 1 day would be invalid, so we set From to 0
		{
			&MessagesRequest{From: 0, To: 100},
			&MessagesRequest{From: 0, To: 100, Timeout: defaultRequestTimeout},
		},
		// set Timeout
		{
			&MessagesRequest{From: 0, To: 0, Timeout: 100},
			&MessagesRequest{From: yesterday, To: now, Timeout: 100},
		},
	}

	for i, s := range scenarios {
		t.Run(fmt.Sprintf("Scenario %d", i), func(t *testing.T) {
			s.given.setDefaults(tnow)
			require.Equal(t, s.expected, s.given)
		})
	}
}

func TestTopicsToBloom(t *testing.T) {
	t1 := stringToTopic("t1")
	b1 := whisper.TopicToBloom(t1)
	t2 := stringToTopic("t2")
	b2 := whisper.TopicToBloom(t2)
	t3 := stringToTopic("t3")
	b3 := whisper.TopicToBloom(t3)

	reqBloom := topicsToBloom(t1)
	assert.True(t, whisper.BloomFilterMatch(reqBloom, b1))
	assert.False(t, whisper.BloomFilterMatch(reqBloom, b2))
	assert.False(t, whisper.BloomFilterMatch(reqBloom, b3))

	reqBloom = topicsToBloom(t1, t2)
	assert.True(t, whisper.BloomFilterMatch(reqBloom, b1))
	assert.True(t, whisper.BloomFilterMatch(reqBloom, b2))
	assert.False(t, whisper.BloomFilterMatch(reqBloom, b3))

	reqBloom = topicsToBloom(t1, t2, t3)
	assert.True(t, whisper.BloomFilterMatch(reqBloom, b1))
	assert.True(t, whisper.BloomFilterMatch(reqBloom, b2))
	assert.True(t, whisper.BloomFilterMatch(reqBloom, b3))
}

func TestCreateBloomFilter(t *testing.T) {
	t1 := stringToTopic("t1")
	t2 := stringToTopic("t2")

	req := MessagesRequest{Topic: t1}
	bloom := createBloomFilter(req)
	assert.Equal(t, topicsToBloom(t1), bloom)

	req = MessagesRequest{Topics: []whisper.TopicType{t1, t2}}
	bloom = createBloomFilter(req)
	assert.Equal(t, topicsToBloom(t1, t2), bloom)
}

func stringToTopic(s string) whisper.TopicType {
	return whisper.BytesToTopic([]byte(s))
}
