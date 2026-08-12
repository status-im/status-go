package mailservers

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/db/appdatabase"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/pkg/messaging/types"
	"github.com/status-im/status-go/protocol/sqlite"
)

func setupTestDB(t *testing.T) *Database {
	db, cleanup, err := testutils.SetupTestSQLDB(appdatabase.DbInitializer{}, "maliservers-tests-")
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
		{PubsubTopic: types.DefaultShardPubsubTopic(), ContentTopic: topicA, LastRequest: 1},
		{PubsubTopic: types.DefaultShardPubsubTopic(), ContentTopic: "0x6200000", LastRequest: 2},
		{PubsubTopic: types.DefaultShardPubsubTopic(), ContentTopic: "0x6300000", LastRequest: 3},
	}

	require.NoError(t, db.AddTopics(topics))

	topics, err := db.Topics()
	require.NoError(t, err)
	require.Len(t, topics, 3)

	filters := types.ChatFilters{
		// Existing topic, is not updated
		types.NewChatFilter(
			&types.ChatFilterConfig{
				PubsubTopic:  types.DefaultShardPubsubTopic(),
				ContentTopic: types.BytesToContentTopic([]byte{0x61}),
			},
		),
		// Non existing topic is not inserted
		types.NewChatFilter(
			&types.ChatFilterConfig{
				Discovery:    true,
				Negotiated:   true,
				PubsubTopic:  types.DefaultShardPubsubTopic(),
				ContentTopic: types.BytesToContentTopic([]byte{0x64}),
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
	require.Zero(t, topics[1].LastRequest)
	require.True(t, topics[1].Negotiated)
	require.True(t, topics[1].Discovery)

	require.NoError(t, db.AdvanceHistoryCursors(10))
	topics, err = db.Topics()
	require.NoError(t, err)
	require.Equal(t, 10, topics[0].LastRequest)
	require.Zero(t, topics[1].LastRequest, "uninitialized topics must not be advanced")

	require.NoError(t, db.AddTopics([]MailserverTopic{{
		PubsubTopic:  types.DefaultShardPubsubTopic(),
		ContentTopic: topicA,
		LastRequest:  5,
	}}))
	topics, err = db.Topics()
	require.NoError(t, err)
	require.Equal(t, 10, topics[0].LastRequest, "completed cursors must never regress")
}
