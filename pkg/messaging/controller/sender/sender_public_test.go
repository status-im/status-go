package sender

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/event"
	bindata "github.com/status-im/migrate/v4/source/go_bindata"
	mvdsnode "github.com/status-im/mvds/node"
	mvdsmigrations "github.com/status-im/mvds/persistenceutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/status-im/status-go/internal/crypto"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/instrumentation/trace"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/pkg/messaging/common"
	"github.com/status-im/status-go/pkg/messaging/layers/encryption"
	encryptionmigrations "github.com/status-im/status-go/pkg/messaging/layers/encryption/migrations"
	"github.com/status-im/status-go/pkg/messaging/layers/reliability"
	"github.com/status-im/status-go/pkg/messaging/layers/segmentation"
	"github.com/status-im/status-go/pkg/messaging/layers/transport"
	transportmigrations "github.com/status-im/status-go/pkg/messaging/layers/transport/migrations"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
	wakutypes "github.com/status-im/status-go/pkg/messaging/waku/types"
	"github.com/status-im/status-go/pkg/pubsub"
)

type publicSendWakuStub struct {
	wakutypes.Waku
	hashes [][]byte
}

func (w *publicSendWakuStub) Send(_ context.Context, _, _ string, payload []byte, _ bool, _ *int) ([]byte, error) {
	hash := crypto.Keccak256(payload)
	w.hashes = append(w.hashes, hash)
	return hash, nil
}

func (*publicSendWakuStub) MaxMessageSize() uint32 { return 1024 * 1024 }

func (*publicSendWakuStub) Subscribe(context.Context, string, []wakutypes.TopicType) error {
	return nil
}

func (*publicSendWakuStub) Unsubscribe(context.Context, string, []wakutypes.TopicType) error {
	return nil
}

func (*publicSendWakuStub) SubscribeEnvelopeEvents(chan<- wakutypes.EnvelopeEvent) wakutypes.Subscription {
	return event.NewSubscription(func(quit <-chan struct{}) error {
		<-quit
		return nil
	})
}

type failingSentSDSHintStore struct {
	err       error
	channelID string
	messageID string
	hashes    [][]byte
}

func (p *failingSentSDSHintStore) SaveSDSRetrievalHints(channelID, messageID string, hashes [][]byte) error {
	p.channelID, p.messageID, p.hashes = channelID, messageID, hashes
	return p.err
}

func (*failingSentSDSHintStore) LoadSDSRetrievalHints(string, string) ([][]byte, error) {
	return nil, nil
}

func TestSendPublicSDSHintSaveFailurePreservesSendCompletion(t *testing.T) {
	db, err := testutils.SetupTestMemorySQLDB(testutils.NewTestDBInitializer([]*bindata.AssetSource{
		{Names: transportmigrations.AssetNames(), AssetFunc: transportmigrations.Asset},
		{Names: encryptionmigrations.AssetNames(), AssetFunc: encryptionmigrations.Asset},
	}))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, mvdsmigrations.Migrate(db))
	identity, err := crypto.GenerateKey()
	require.NoError(t, err)
	core, logs := observer.New(zap.ErrorLevel)
	logger := zap.New(core)
	backend := &publicSendWakuStub{}
	tr, err := transport.NewTransport(
		backend, identity,
		transport.NewSQLiteKeysPersistence(db),
		transport.NewSQLiteProcessedMessageIDsCachePersistence(db),
		&transport.EnvelopesMonitorConfig{Logger: logger},
		logger,
	)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tr.Stop()) })
	reliabilityLayer, err := reliability.NewReliability(
		mvdsnode.NewSQLitePersistence(db), identity,
		func(string, []string, string) error { return nil }, logger,
	)
	require.NoError(t, err)
	t.Cleanup(reliabilityLayer.Close)
	hints := &failingSentSDSHintStore{err: errors.New("injected hint save failure")}
	reliabilityLayer.SetSDSRetrievalHintPersistence(hints)
	sender := NewSender(identity, &common.MessagingStack{
		Transport:   tr,
		Reliability: reliabilityLayer,
		Segmentation: segmentation.NewSegmenter(
			segmentation.NewSQLitePersistence(db), logger,
		),
		Encryption: encryption.New(encryption.NewSQLitePersistence(db), "installation-1", logger, trace.NewNoopTracer()),
	}, logger, trace.NewNoopTracer())
	sent, unsubscribe := pubsub.Subscribe[SentMessage](sender.Publisher(), 1)
	defer unsubscribe()
	params := messagingtypes.SendPublicParams{
		Sender:       &identity.PublicKey,
		Payload:      []byte("published SDS message"),
		CommunityID:  "community-id",
		ContentTopic: "community-chat",
	}
	messageID := messagingtypes.MessageID(params.Sender, params.Payload)
	sdsMessageID := crypto.Keccak256(params.Payload)

	require.NoError(t, sender.SendPublic(context.Background(), params))
	require.Len(t, backend.hashes, 1)
	require.Equal(t, publicSDSChannelID(params), hints.channelID)
	require.Equal(t, cryptotypes.EncodeHex(sdsMessageID), hints.messageID)
	require.Equal(t, backend.hashes, hints.hashes)
	for _, id := range [][]byte{[]byte(messageID), sdsMessageID} {
		hashes, err := tr.TrackedEnvelopeHashes(id)
		require.NoError(t, err)
		require.Equal(t, cryptotypes.EncodeHexes(backend.hashes), hashes)
	}
	applicationMessageID, ok := tr.TakeApplicationMessageIDForSDS(sdsMessageID)
	require.True(t, ok)
	require.Equal(t, []byte(messageID), applicationMessageID)
	select {
	case notification := <-sent:
		require.False(t, notification.Private)
		require.Equal(t, [][]byte{[]byte(messageID)}, notification.MessageIDs)
	default:
		t.Fatal("successful publication must emit SentMessage despite hint-save failure")
	}
	entries := logs.FilterMessage("failed to persist sent SDS retrieval hints").All()
	require.Len(t, entries, 1)
	require.Equal(t, hints.channelID, entries[0].ContextMap()["channelID"])
	require.Equal(t, hints.messageID, entries[0].ContextMap()["sdsMessageID"])
	require.Equal(t, hints.err.Error(), entries[0].ContextMap()["error"])
}

type sdsWrapperStub struct {
	called         bool
	wantPayload    []byte
	wantChannelID  string
	wrappedPayload []byte
	messageID      []byte
	err            error
}

func (s *sdsWrapperStub) WrapPayloadForSDS(payload []byte, channelID string) ([]byte, []byte, error) {
	s.called = true
	s.wantPayload = payload
	s.wantChannelID = channelID
	return s.wrappedPayload, s.messageID, s.err
}

func TestWrapPayloadForPublicSDS_MissingCommunityID(t *testing.T) {
	logger := zap.NewNop()
	payload := []byte("payload")
	stub := &sdsWrapperStub{}

	wrapped, messageID, err := wrapPayloadForPublicSDS(logger, stub, payload, "")

	require.NoError(t, err)
	require.Equal(t, payload, wrapped)
	require.Nil(t, messageID)
	require.False(t, stub.called)
}

func TestWrapPayloadForPublicSDS_OversizedPayload(t *testing.T) {
	logger := zap.NewNop()
	payload := []byte("payload")
	communityID := "community-id"
	stub := &sdsWrapperStub{
		err: errors.New("reMessageTooLarge: message exceeds allowed size"),
	}

	channelID := reliability.BuildChannelID(communityID, "")
	wrapped, messageID, err := wrapPayloadForPublicSDS(logger, stub, payload, channelID)

	require.NoError(t, err)
	require.Equal(t, payload, wrapped)
	require.Nil(t, messageID)
	require.True(t, stub.called)
	require.Equal(t, payload, stub.wantPayload)
	require.Equal(t, channelID, stub.wantChannelID)
}

func TestWrapPayloadForPublicSDS_WrapError(t *testing.T) {
	logger := zap.NewNop()
	payload := []byte("payload")
	stub := &sdsWrapperStub{
		err: errors.New("boom"),
	}

	channelID := reliability.BuildChannelID("community-id", "")
	wrapped, messageID, err := wrapPayloadForPublicSDS(logger, stub, payload, channelID)

	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to wrap payload for SDS")
	require.Contains(t, err.Error(), "boom")
	require.Equal(t, payload, wrapped)
	require.Nil(t, messageID)
	require.True(t, stub.called)
}

func TestWrapPayloadForPublicSDS_Success(t *testing.T) {
	logger := zap.NewNop()
	payload := []byte("payload")
	wantedWrapped := []byte("wrapped-payload")
	wantedMessageID := []byte{0x01, 0x02, 0x03}
	stub := &sdsWrapperStub{
		wrappedPayload: wantedWrapped,
		messageID:      wantedMessageID,
	}

	wrapped, messageID, err := wrapPayloadForPublicSDS(logger, stub, payload, "community-id")

	require.NoError(t, err)
	require.Equal(t, wantedWrapped, wrapped)
	require.Equal(t, wantedMessageID, messageID)
	require.True(t, stub.called)
}

func TestWrapPublicParamsForSDS_SkipSDS(t *testing.T) {
	payload := []byte("payload")
	stub := &sdsWrapperStub{wrappedPayload: []byte("wrapped-payload")}

	wrapped, messageID, err := wrapPublicParamsForSDS(zap.NewNop(), stub, messagingtypes.SendPublicParams{
		Payload:     payload,
		CommunityID: "community-id",
		SkipSDS:     true,
	})

	require.NoError(t, err)
	require.Equal(t, payload, wrapped)
	require.Nil(t, messageID)
	require.False(t, stub.called)
}

func TestWrapPublicParamsForSDS_WrapsCommunityPayload(t *testing.T) {
	stub := &sdsWrapperStub{wrappedPayload: []byte("wrapped-payload"), messageID: []byte{0x01}}

	wrapped, messageID, err := wrapPublicParamsForSDS(zap.NewNop(), stub, messagingtypes.SendPublicParams{
		Payload:     []byte("payload"),
		CommunityID: "community-id",
	})

	require.NoError(t, err)
	require.Equal(t, stub.wrappedPayload, wrapped)
	require.Equal(t, stub.messageID, messageID)
	require.Equal(t, reliability.BuildChannelID("community-id", ""), stub.wantChannelID)
}

func TestWrapPublicParamsForSDS_ScopesPayloadToContentTopic(t *testing.T) {
	stub := &sdsWrapperStub{wrappedPayload: []byte("wrapped-payload")}

	_, _, err := wrapPublicParamsForSDS(zap.NewNop(), stub, messagingtypes.SendPublicParams{
		Payload:      []byte("payload"),
		CommunityID:  "community-id",
		ContentTopic: "community-chat",
	})

	require.NoError(t, err)
	require.Equal(t, reliability.BuildChannelID("community-id", "community-chat"), stub.wantChannelID)
}

func TestWrapPublicParamsForSDS_DerivesCommunityIDFromPublicKey(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	stub := &sdsWrapperStub{wrappedPayload: []byte("wrapped-payload")}

	_, _, err = wrapPublicParamsForSDS(zap.NewNop(), stub, messagingtypes.SendPublicParams{
		Payload:            []byte("payload"),
		CommunityPublicKey: &key.PublicKey,
	})

	require.NoError(t, err)
	communityID := cryptotypes.EncodeHex(crypto.CompressPubkey(&key.PublicKey))
	require.Equal(t, reliability.BuildChannelID(communityID, ""), stub.wantChannelID)
}
