package ext

import (
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/status-im/status-go/eth-node/types"

	"github.com/status-im/status-go/mailserver"

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
			s.given.SetDefaults(tnow)
			require.Equal(t, s.expected, s.given)
		})
	}
}

func TestMakeMessagesRequestPayload(t *testing.T) {
	var emptyTopic types.TopicType
	testCases := []struct {
		Name string
		Req  MessagesRequest
		Err  string
	}{
		{
			Name: "empty cursor",
			Req:  MessagesRequest{Cursor: ""},
			Err:  "",
		},
		{
			Name: "invalid cursor size",
			Req:  MessagesRequest{Cursor: hex.EncodeToString([]byte{0x01, 0x02, 0x03})},
			Err:  fmt.Sprintf("invalid cursor size: expected %d but got 3", mailserver.CursorLength),
		},
		{
			Name: "valid cursor",
			Req: MessagesRequest{
				Cursor: hex.EncodeToString(mailserver.NewDBKey(123, emptyTopic, types.Hash{}).Cursor()),
			},
			Err: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			_, err := MakeMessagesRequestPayload(tc.Req)
			if tc.Err == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tc.Err)
			}
		})
	}
}

func TestTopicsToBloom(t *testing.T) {
	t1 := stringToTopic("t1")
	b1 := types.TopicToBloom(t1)
	t2 := stringToTopic("t2")
	b2 := types.TopicToBloom(t2)
	t3 := stringToTopic("t3")
	b3 := types.TopicToBloom(t3)

	reqBloom := topicsToBloom(t1)
	assert.True(t, types.BloomFilterMatch(reqBloom, b1))
	assert.False(t, types.BloomFilterMatch(reqBloom, b2))
	assert.False(t, types.BloomFilterMatch(reqBloom, b3))

	reqBloom = topicsToBloom(t1, t2)
	assert.True(t, types.BloomFilterMatch(reqBloom, b1))
	assert.True(t, types.BloomFilterMatch(reqBloom, b2))
	assert.False(t, types.BloomFilterMatch(reqBloom, b3))

	reqBloom = topicsToBloom(t1, t2, t3)
	assert.True(t, types.BloomFilterMatch(reqBloom, b1))
	assert.True(t, types.BloomFilterMatch(reqBloom, b2))
	assert.True(t, types.BloomFilterMatch(reqBloom, b3))
}

func TestCreateBloomFilter(t *testing.T) {
	t1 := stringToTopic("t1")
	t2 := stringToTopic("t2")

	req := MessagesRequest{Topic: t1}
	bloom := createBloomFilter(req)
	assert.Equal(t, topicsToBloom(t1), bloom)

	req = MessagesRequest{Topics: []types.TopicType{t1, t2}}
	bloom = createBloomFilter(req)
	assert.Equal(t, topicsToBloom(t1, t2), bloom)
}

func stringToTopic(s string) types.TopicType {
	return types.BytesToTopic([]byte(s))
}

func TestExpiredOrCompleted(t *testing.T) {
	timeout := time.Millisecond
	events := make(chan types.EnvelopeEvent)
	errors := make(chan error, 1)
	hash := types.Hash{1}
	go func() {
		_, err := WaitForExpiredOrCompleted(hash, events, timeout)
		errors <- err
	}()
	select {
	case <-time.After(time.Second):
		require.FailNow(t, "timed out waiting for waitForExpiredOrCompleted to complete")
	case err := <-errors:
		require.EqualError(t, err, fmt.Sprintf("request %x expired", hash))
	}
}
