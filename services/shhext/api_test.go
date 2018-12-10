package shhext

import (
	"context"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/mailserver"

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

func TestMakeMessagesRequestPayload(t *testing.T) {
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
			Err:  fmt.Sprintf("invalid cursor size: expected %d but got 3", mailserver.DBKeyLength),
		},
		{
			Name: "valid cursor",
			Req: MessagesRequest{
				Cursor: hex.EncodeToString(mailserver.NewDBKey(123, common.Hash{}).Bytes()),
			},
			Err: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			_, err := makeMessagesRequestPayload(tc.Req)
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

func TestCreateSyncMailRequest(t *testing.T) {
	testCases := []struct {
		Name   string
		Req    SyncMessagesRequest
		Verify func(*testing.T, whisper.SyncMailRequest)
		Error  string
	}{
		{
			Name: "no topics",
			Req:  SyncMessagesRequest{},
			Verify: func(t *testing.T, r whisper.SyncMailRequest) {
				require.Equal(t, whisper.MakeFullNodeBloom(), r.Bloom)
			},
		},
		{
			Name: "some topics",
			Req: SyncMessagesRequest{
				Topics: []whisper.TopicType{{0x01, 0xff, 0xff, 0xff}},
			},
			Verify: func(t *testing.T, r whisper.SyncMailRequest) {
				expectedBloom := whisper.TopicToBloom(whisper.TopicType{0x01, 0xff, 0xff, 0xff})
				require.Equal(t, expectedBloom, r.Bloom)
			},
		},
		{
			Name: "decode cursor",
			Req: SyncMessagesRequest{
				Cursor: hex.EncodeToString([]byte{0x01, 0x02, 0x03}),
			},
			Verify: func(t *testing.T, r whisper.SyncMailRequest) {
				require.Equal(t, []byte{0x01, 0x02, 0x03}, r.Cursor)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			r, err := createSyncMailRequest(tc.Req)
			if tc.Error != "" {
				require.EqualError(t, err, tc.Error)
			}
			tc.Verify(t, r)
		})
	}
}

func TestSyncMessagesErrors(t *testing.T) {
	validEnode := "enode://e8a7c03b58911e98bbd66accb2a55d57683f35b23bf9dfca89e5e244eb5cc3f25018b4112db507faca34fb69ffb44b362f79eda97a669a8df29c72e654416784@127.0.0.1:30404"

	testCases := []struct {
		Name  string
		Req   SyncMessagesRequest
		Error string
	}{
		{
			Name:  "invalid MailServerPeer",
			Req:   SyncMessagesRequest{MailServerPeer: "invalid-scheme://"},
			Error: `invalid MailServerPeer: invalid URL scheme, want "enode"`,
		},
		{
			Name: "failed to create SyncMailRequest",
			Req: SyncMessagesRequest{
				MailServerPeer: validEnode,
				Cursor:         "a", // odd number of characters is an invalid hex representation
			},
			Error: "failed to create a sync mail request: encoding/hex: odd length hex string",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			api := PublicAPI{}
			err := api.SyncMessages(context.TODO(), tc.Req)
			if tc.Error != "" {
				require.EqualError(t, err, tc.Error)
			}
		})
	}
}
