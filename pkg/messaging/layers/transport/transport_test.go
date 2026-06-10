package transport

import (
	"crypto/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	bindata "github.com/status-im/migrate/v4/source/go_bindata"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/pkg/messaging/layers/transport/migrations"
	"github.com/status-im/status-go/pkg/messaging/layers/transport/rfc26"
	wakuv3 "github.com/status-im/status-go/pkg/messaging/waku"
	wakutypes "github.com/status-im/status-go/pkg/messaging/waku/types"
)

func TestNewTransport(t *testing.T) {
	db, err := testutils.SetupTestMemorySQLDB(testutils.NewTestDBInitializer([]*bindata.AssetSource{
		{
			Names:     migrations.AssetNames(),
			AssetFunc: migrations.Asset,
		},
	}))
	require.NoError(t, err)

	logger := testutils.MustCreateTestLogger()
	defer func() { _ = logger.Sync() }()

	tr, err := NewTransport(nil, nil, NewSQLiteKeysPersistence(db), NewSQLiteProcessedMessageIDsCachePersistence(db), nil, logger)
	require.NoError(t, err)
	// Stop the transport to prevent cleanFiltersLoop from leaking into later tests.
	t.Cleanup(func() { _ = tr.Stop() })
}

// TestReceivePushPath exercises the push receive pipeline end to end at the
// transport boundary: a neutral ReceivedMessage is routed by content topic to
// the matching filter, decoded via rfc26, and surfaced through RetrieveRawAll.
func TestReceivePushPath(t *testing.T) {
	db, err := testutils.SetupTestMemorySQLDB(testutils.NewTestDBInitializer([]*bindata.AssetSource{
		{Names: migrations.AssetNames(), AssetFunc: migrations.Asset},
	}))
	require.NoError(t, err)

	logger := testutils.MustCreateTestLogger()
	defer func() { _ = logger.Sync() }()

	waku, err := wakuv3.New(nil, &wakuv3.DefaultConfig, logger, nil, func([]byte, peer.AddrInfo, error) {}, nil)
	require.NoError(t, err)

	identity, err := crypto.GenerateKey()
	require.NoError(t, err)

	tr, err := NewTransport(waku, identity, NewSQLiteKeysPersistence(db), NewSQLiteProcessedMessageIDsCachePersistence(db), nil, logger)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tr.Stop() })

	// A public chat installs a symmetric, listening filter.
	filter, err := tr.JoinPublic("test-public-chat")
	require.NoError(t, err)
	require.True(t, filter.Listen)

	symKey, ok := tr.filters.SymKey(filter.FilterID)
	require.True(t, ok)

	payload := []byte("hello receive path")
	encoded, err := rfc26.Encode(payload, symKey, nil, identity)
	require.NoError(t, err)

	hash := []byte{0xaa, 0xbb, 0xcc}
	tr.handleReceivedMessage(&wakutypes.ReceivedMessage{
		Hash:         hash,
		ContentTopic: filter.ContentTopic.ContentTopic(),
		PubsubTopic:  wakuv3.DefaultShardPubsubTopic(),
		Payload:      encoded,
		Version:      1,
		Timestamp:    time.Now().UnixNano(),
	})

	result, err := tr.RetrieveRawAll()
	require.NoError(t, err)

	msgs := result[*filter]
	require.Len(t, msgs, 1)
	require.Equal(t, payload, msgs[0].Payload)
	require.Equal(t, filter.ContentTopic, msgs[0].Topic)
	require.Equal(t, hash, msgs[0].Hash)
	require.Equal(t, crypto.FromECDSAPub(&identity.PublicKey), msgs[0].Sig)

	// Draining again yields nothing: the buffer was emptied.
	result, err = tr.RetrieveRawAll()
	require.NoError(t, err)
	require.Empty(t, result)

	// A message on an unknown content topic routes to no filter.
	tr.handleReceivedMessage(&wakutypes.ReceivedMessage{
		Hash:         []byte{0x01},
		ContentTopic: "/waku/2/unknown/proto",
		PubsubTopic:  wakuv3.DefaultShardPubsubTopic(),
		Payload:      encoded,
		Version:      1,
		Timestamp:    time.Now().UnixNano(),
	})
	result, err = tr.RetrieveRawAll()
	require.NoError(t, err)
	require.Empty(t, result)

	// A message on the right (pubsub, content) topic but encrypted with the wrong
	// key is dropped: the authenticated decrypt fails. This is exactly how a
	// colliding chat (content topics are a 4-byte hash) is disambiguated.
	wrongKey := make([]byte, len(symKey))
	copy(wrongKey, symKey)
	wrongKey[0] ^= 0xFF
	badPayload, err := rfc26.Encode(payload, wrongKey, nil, identity)
	require.NoError(t, err)
	tr.handleReceivedMessage(&wakutypes.ReceivedMessage{
		Hash:         []byte{0x02},
		ContentTopic: filter.ContentTopic.ContentTopic(),
		PubsubTopic:  wakuv3.DefaultShardPubsubTopic(),
		Payload:      badPayload,
		Version:      1,
		Timestamp:    time.Now().UnixNano(),
	})
	result, err = tr.RetrieveRawAll()
	require.NoError(t, err)
	require.Empty(t, result)
}

// TestReceiveVersionZeroDecodesAsV1 covers the WakuMessage version-field
// migration (status-im/status-go#7499): an incoming message whose payload is
// rfc26-encrypted but whose envelope advertises version=0 must still be
// decrypted on a keyed filter (treated the same as version=1), now that senders
// advertise version=0. Previously version=0 was passed through as plaintext.
func TestReceiveVersionZeroDecodesAsV1(t *testing.T) {
	db, err := testutils.SetupTestMemorySQLDB(testutils.NewTestDBInitializer([]*bindata.AssetSource{
		{Names: migrations.AssetNames(), AssetFunc: migrations.Asset},
	}))
	require.NoError(t, err)

	logger := testutils.MustCreateTestLogger()
	defer func() { _ = logger.Sync() }()

	waku, err := wakuv3.New(nil, &wakuv3.DefaultConfig, logger, nil, func([]byte, peer.AddrInfo, error) {}, nil)
	require.NoError(t, err)

	identity, err := crypto.GenerateKey()
	require.NoError(t, err)

	tr, err := NewTransport(waku, identity, NewSQLiteKeysPersistence(db), NewSQLiteProcessedMessageIDsCachePersistence(db), nil, logger)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tr.Stop() })

	filter, err := tr.JoinPublic("test-public-chat")
	require.NoError(t, err)

	symKey, ok := tr.filters.SymKey(filter.FilterID)
	require.True(t, ok)

	payload := []byte("hello version zero")
	encoded, err := rfc26.Encode(payload, symKey, nil, identity)
	require.NoError(t, err)

	// The payload is rfc26-encrypted but the envelope is labeled version=0.
	tr.handleReceivedMessage(&wakutypes.ReceivedMessage{
		Hash:         []byte{0xde, 0xad},
		ContentTopic: filter.ContentTopic.ContentTopic(),
		PubsubTopic:  wakuv3.DefaultShardPubsubTopic(),
		Payload:      encoded,
		Version:      0,
		Timestamp:    time.Now().UnixNano(),
	})

	result, err := tr.RetrieveRawAll()
	require.NoError(t, err)

	msgs := result[*filter]
	require.Len(t, msgs, 1, "version=0 message should be decoded as WakuV1")
	require.Equal(t, payload, msgs[0].Payload)
	require.Equal(t, crypto.FromECDSAPub(&identity.PublicKey), msgs[0].Sig)
}

// TestReceiveSharedKeyFanOut covers the decode-once grouping in
// handleReceivedMessage: filters listening on the same (pubsub, content) topic
// with the same symmetric key form one key group — the payload is decrypted
// once and the decoded message fanned out to every filter in the group — while
// a colliding filter (same topic, different key) stays in its own group and
// receives nothing because its decrypt fails.
func TestReceiveSharedKeyFanOut(t *testing.T) {
	db, err := testutils.SetupTestMemorySQLDB(testutils.NewTestDBInitializer([]*bindata.AssetSource{
		{Names: migrations.AssetNames(), AssetFunc: migrations.Asset},
	}))
	require.NoError(t, err)

	logger := testutils.MustCreateTestLogger()
	defer func() { _ = logger.Sync() }()

	waku, err := wakuv3.New(nil, &wakuv3.DefaultConfig, logger, nil, func([]byte, peer.AddrInfo, error) {}, nil)
	require.NoError(t, err)

	identity, err := crypto.GenerateKey()
	require.NoError(t, err)

	tr, err := NewTransport(waku, identity, NewSQLiteKeysPersistence(db), NewSQLiteProcessedMessageIDsCachePersistence(db), nil, logger)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tr.Stop() })

	sharedKey := make([]byte, 32)
	_, err = rand.Read(sharedKey)
	require.NoError(t, err)

	otherKey := make([]byte, 32)
	_, err = rand.Read(otherKey)
	require.NoError(t, err)

	// Install three listening filters on the same (pubsub, content) topic
	// directly in the manager: two sharing a key, one colliding with its own.
	contentTopic := wakutypes.BytesToTopic(ToTopic("shared-chat"))
	mkFilter := func(filterID, chatID string, symKey []byte) *Filter {
		require.NoError(t, tr.filters.addSymKey(filterID, symKey))
		return &Filter{
			ChatID:       chatID,
			FilterID:     filterID,
			ContentTopic: contentTopic,
			PubsubTopic:  wakuv3.DefaultShardPubsubTopic(),
			Listen:       true,
		}
	}
	sharedA := mkFilter("filter-shared-a", "chat-a", sharedKey)
	sharedB := mkFilter("filter-shared-b", "chat-b", sharedKey)
	collider := mkFilter("filter-collider", "chat-c", otherKey)
	for _, f := range []*Filter{sharedA, sharedB, collider} {
		tr.filters.filters[f.FilterID] = f
	}

	payload := []byte("fan out once")
	encoded, err := rfc26.Encode(payload, sharedKey, nil, identity)
	require.NoError(t, err)

	tr.handleReceivedMessage(&wakutypes.ReceivedMessage{
		Hash:         []byte{0xfa, 0x07},
		ContentTopic: contentTopic.ContentTopic(),
		PubsubTopic:  wakuv3.DefaultShardPubsubTopic(),
		Payload:      encoded,
		Version:      1,
		Timestamp:    time.Now().UnixNano(),
	})

	result, err := tr.RetrieveRawAll()
	require.NoError(t, err)

	// Both shared-key filters got the decoded message; the collider got nothing.
	require.Len(t, result[*sharedA], 1)
	require.Equal(t, payload, result[*sharedA][0].Payload)
	require.Len(t, result[*sharedB], 1)
	require.Equal(t, payload, result[*sharedB][0].Payload)
	require.Empty(t, result[*collider])
}

func TestCleanFiltersLoopPausesAndResumesByLifecycle(t *testing.T) {
	originalInterval := cleanFiltersLoopInterval
	cleanFiltersLoopInterval = 20 * time.Millisecond
	t.Cleanup(func() { cleanFiltersLoopInterval = originalInterval })

	logger := testutils.MustCreateTestLogger()
	defer func() { _ = logger.Sync() }()

	quit := make(chan struct{})
	var quitOnce sync.Once
	shutdown := func() {
		quitOnce.Do(func() { close(quit) })
	}
	defer shutdown()

	var cleanCalls atomic.Int32
	tr := &Transport{
		logger: logger,
		quit:   quit,
		cleanFiltersFn: func() error {
			cleanCalls.Add(1)
			return nil
		},
	}
	tr.MarkPaused()

	tr.cleanFiltersLoop()

	// While paused the ticker must not run: Never fails if cleanCalls becomes non-zero during the window.
	pausedSoak := 3 * cleanFiltersLoopInterval
	require.Never(t, func() bool { return cleanCalls.Load() > 0 }, pausedSoak, 5*time.Millisecond,
		"ticks must not run while paused")

	tr.MarkResumed()
	require.Eventually(t, func() bool {
		return cleanCalls.Load() > 0
	}, time.Second, 20*time.Millisecond, "expected at least one tick after resume")

	shutdown()
}
