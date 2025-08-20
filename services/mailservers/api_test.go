package mailservers

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/appdatabase"
	messagingtypes "github.com/status-im/status-go/messaging/types"
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/t/helpers"
)

func setupTestDB(t *testing.T) *Database {
	db, cleanup, err := helpers.SetupTestSQLDB(appdatabase.DbInitializer{}, "maliservers-tests-")
	require.NoError(t, err)
	err = sqlite.Migrate(db) // migrate default
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, cleanup()) })
	return NewDB(db)
}

func TestTopic(t *testing.T) {
	db := setupTestDB(t)

	const topicA = "0x61000000"
	const topicD = "0x64000000"
	topics := []MailserverTopic{
		{PubsubTopic: messagingtypes.DefaultShardPubsubTopic(), ContentTopic: topicA, LastRequest: 1},
		{PubsubTopic: messagingtypes.DefaultShardPubsubTopic(), ContentTopic: "0x6200000", LastRequest: 2},
		{PubsubTopic: messagingtypes.DefaultShardPubsubTopic(), ContentTopic: "0x6300000", LastRequest: 3},
	}

	require.NoError(t, db.AddTopics(topics))

	topics, err := db.Topics()
	require.NoError(t, err)
	require.Len(t, topics, 3)

	filters := messagingtypes.ChatFilters{
		// Existing topic, is not updated
		messagingtypes.NewChatFilter(
			&messagingtypes.ChatFilterConfig{
				PubsubTopic:  messagingtypes.DefaultShardPubsubTopic(),
				ContentTopic: messagingtypes.BytesToContentTopic([]byte{0x61}),
			},
		),
		// Non existing topic is not inserted
		messagingtypes.NewChatFilter(
			&messagingtypes.ChatFilterConfig{
				Discovery:    true,
				Negotiated:   true,
				PubsubTopic:  messagingtypes.DefaultShardPubsubTopic(),
				ContentTopic: messagingtypes.BytesToContentTopic([]byte{0x64}),
			},
		),
	}

	require.NoError(t, db.SetTopics(filters))

	topics, err = db.Topics()
	require.NoError(t, err)
	require.Len(t, topics, 2)
	require.Equal(t, topics[0].ContentTopic, topicA)
	require.Equal(t, topics[0].LastRequest, 1)

	require.Equal(t, topics[0].ContentTopic, topicA)
	require.Equal(t, topics[0].LastRequest, 1)

	require.Equal(t, topics[1].ContentTopic, topicD)
	require.NotEmpty(t, topics[1].LastRequest)
	require.True(t, topics[1].Negotiated)
	require.True(t, topics[1].Discovery)
}
