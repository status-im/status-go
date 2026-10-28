package transport

import (
	"context"
	"crypto/rand"
	"sync"
	"testing"
	"time"

	bindata "github.com/status-im/migrate/v4/source/go_bindata"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/pkg/messaging/layers/transport/migrations"
	"github.com/status-im/status-go/pkg/messaging/layers/transport/rfc26"
	"github.com/status-im/status-go/pkg/messaging/waku"
	wakutypes "github.com/status-im/status-go/pkg/messaging/waku/types"
)

type subscriptionCall struct {
	subscribe    bool
	pubsubTopic  string
	contentTopic string
}

type contentTopicOnlyAPI struct {
	mu    sync.Mutex
	calls []subscriptionCall
}

func (a *contentTopicOnlyAPI) ContentTopicOnly() {}

func (a *contentTopicOnlyAPI) Send(context.Context, string, string, []byte, bool, *int) ([]byte, error) {
	return nil, nil
}

func (a *contentTopicOnlyAPI) SubscribeEnvelopeEvents(chan<- wakutypes.EnvelopeEvent) wakutypes.Subscription {
	return nil
}

func (a *contentTopicOnlyAPI) Subscribe(_ context.Context, pubsubTopic string, contentTopics []wakutypes.TopicType) error {
	a.record(true, pubsubTopic, contentTopics)
	return nil
}

func (a *contentTopicOnlyAPI) Unsubscribe(_ context.Context, pubsubTopic string, contentTopics []wakutypes.TopicType) error {
	a.record(false, pubsubTopic, contentTopics)
	return nil
}

func (a *contentTopicOnlyAPI) record(subscribe bool, pubsubTopic string, contentTopics []wakutypes.TopicType) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, topic := range contentTopics {
		a.calls = append(a.calls, subscriptionCall{subscribe, pubsubTopic, topic.ContentTopic()})
	}
}

func TestFilterSubscriptionsCountByContentTopicOnly(t *testing.T) {
	api := &contentTopicOnlyAPI{}
	subs := newFilterSubscriptions(api, testutils.MustCreateTestLogger())

	contentTopic := wakutypes.BytesToTopic(ToTopic("admin"))
	onDefault := &Filter{FilterID: "a", ContentTopic: contentTopic, PubsubTopic: waku.DefaultShardPubsubTopic()}
	onNonProtected := &Filter{FilterID: "b", ContentTopic: contentTopic, PubsubTopic: waku.DefaultNonProtectedPubsubTopic()}

	require.NoError(t, subs.Acquire(onDefault, onNonProtected))
	require.Equal(t, []subscriptionCall{{true, "", contentTopic.ContentTopic()}}, api.calls)

	subs.Release(onNonProtected)
	require.Len(t, api.calls, 1)

	subs.Release(onDefault)
	require.Equal(t, subscriptionCall{false, "", contentTopic.ContentTopic()}, api.calls[1])
}

// A message without a pubsub topic reaches a chat registered on several pubsub
// topics once, through its default-shard filters, which still fan out.
func TestReceiveWithoutPubsubTopicOnOnePubsubTopic(t *testing.T) {
	db, err := testutils.SetupTestMemorySQLDB(testutils.NewTestDBInitializer([]*bindata.AssetSource{
		{Names: migrations.AssetNames(), AssetFunc: migrations.Asset},
	}))
	require.NoError(t, err)

	logger := testutils.MustCreateTestLogger()
	defer func() { _ = logger.Sync() }()

	wakuNode, err := waku.New(nil, &waku.DefaultConfig, logger, nil)
	require.NoError(t, err)

	identity, err := crypto.GenerateKey()
	require.NoError(t, err)

	tr, err := NewTransport(wakuNode, identity, NewSQLiteKeysPersistence(db), NewSQLiteProcessedMessageIDsCachePersistence(db), nil, logger)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tr.Stop() })

	key := make([]byte, 32)
	_, err = rand.Read(key)
	require.NoError(t, err)

	contentTopic := wakutypes.BytesToTopic(ToTopic("community-admin"))
	mkFilter := func(filterID, pubsubTopic string) *Filter {
		require.NoError(t, tr.filters.addSymKey(filterID, key))
		f := &Filter{
			ChatID:       filterID,
			FilterID:     filterID,
			ContentTopic: contentTopic,
			PubsubTopic:  pubsubTopic,
			Listen:       true,
		}
		tr.filters.filters[filterID] = f
		return f
	}
	defaultA := mkFilter("default-a", waku.DefaultShardPubsubTopic())
	defaultB := mkFilter("default-b", "")
	nonProtected := mkFilter("non-protected", waku.DefaultNonProtectedPubsubTopic())

	payload := []byte("once per pubsub topic")
	encoded, err := rfc26.Encode(payload, key, nil, identity)
	require.NoError(t, err)

	tr.handleReceivedMessage(&wakutypes.ReceivedMessage{
		Hash:         []byte{0x0c, 0x70},
		ContentTopic: contentTopic.ContentTopic(),
		Payload:      encoded,
		Version:      1,
		Timestamp:    time.Now().UnixNano(),
	})

	result, err := tr.RetrieveRawAll()
	require.NoError(t, err)

	require.Len(t, result[*defaultA], 1)
	require.Equal(t, payload, result[*defaultA][0].Payload)
	require.Len(t, result[*defaultB], 1)
	require.Empty(t, result[*nonProtected])
}
