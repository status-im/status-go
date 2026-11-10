package messaging

import (
	"errors"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/status-im/status-go/pkg/messaging/common"
	"github.com/status-im/status-go/pkg/messaging/events"
	"github.com/status-im/status-go/pkg/messaging/layers/reliability"
	reliabilitypb "github.com/status-im/status-go/pkg/messaging/layers/reliability/protobuf"
	"github.com/status-im/status-go/pkg/pubsub"
)

type sdsEnvelopeHashesTrackerStub struct {
	called         bool
	receivedID     []byte
	trackedHashes  []string
	trackedHashErr error
}

func (s *sdsEnvelopeHashesTrackerStub) TrackedEnvelopeHashes(identifier []byte) ([]string, error) {
	s.called = true
	s.receivedID = identifier
	return s.trackedHashes, s.trackedHashErr
}

func TestBuildSDSRetrievalHint_InvalidMessageID(t *testing.T) {
	tracker := &sdsEnvelopeHashesTrackerStub{}

	hint := buildSDSRetrievalHint(zap.NewNop(), tracker, "not-hex")

	require.Nil(t, hint)
	require.False(t, tracker.called)
}

func TestBuildSDSRetrievalHint_NoTrackedEnvelopeHashes(t *testing.T) {
	tracker := &sdsEnvelopeHashesTrackerStub{
		trackedHashErr: errors.New("not found"),
	}

	hint := buildSDSRetrievalHint(zap.NewNop(), tracker, "0x0102")

	require.Nil(t, hint)
	require.True(t, tracker.called)
	require.Equal(t, []byte{0x01, 0x02}, tracker.receivedID)
}

func TestBuildSDSRetrievalHint_Success(t *testing.T) {
	tracker := &sdsEnvelopeHashesTrackerStub{
		trackedHashes: []string{"0xabc", "0xdef"},
	}

	hintBytes := buildSDSRetrievalHint(zap.NewNop(), tracker, "0x0102")

	require.True(t, tracker.called)
	require.Equal(t, []byte{0x01, 0x02}, tracker.receivedID)
	require.NotNil(t, hintBytes)

	var hint reliabilitypb.RetrievalHint
	require.NoError(t, proto.Unmarshal(hintBytes, &hint))
	require.Equal(t, [][]byte{[]byte("0xabc"), []byte("0xdef")}, hint.EnvelopeHashes)
}

type sdsRetrievalHintPersistenceStub struct {
	hashes map[string][][]byte
}

func (*sdsRetrievalHintPersistenceStub) SaveSDSMessageReceived(string, string) error {
	return nil
}

func (*sdsRetrievalHintPersistenceStub) IsSDSMessageReceived(string, string) (bool, error) {
	return false, nil
}

func (*sdsRetrievalHintPersistenceStub) DeleteExpiredSDSReceivedMessages() error {
	return nil
}

func (s *sdsRetrievalHintPersistenceStub) SaveSDSRetrievalHints(channelID, messageID string, hashes [][]byte) error {
	key := channelID + ":" + messageID
	s.hashes[key] = append(s.hashes[key], hashes...)
	return nil
}

func (s *sdsRetrievalHintPersistenceStub) LoadSDSRetrievalHints(channelID, messageID string) ([][]byte, error) {
	return s.hashes[channelID+":"+messageID], nil
}

type sdsApplicationMessageIDTrackerStub struct {
	called        bool
	receivedID    []byte
	applicationID []byte
	applicationOK bool
}

func (s *sdsApplicationMessageIDTrackerStub) TakeApplicationMessageIDForSDS(sdsMessageID []byte) ([]byte, bool) {
	s.called = true
	s.receivedID = sdsMessageID
	return s.applicationID, s.applicationOK
}

func TestResolveSDSRetrievalHintUsesPersistedHashesAfterDeliveryConfirmation(t *testing.T) {
	messageID := []byte{0x01, 0x02}
	envelopeHash := []byte{0xaa, 0xbb}
	hintStore := &sdsRetrievalHintPersistenceStub{hashes: make(map[string][][]byte)}
	reliabilityLayer := &reliability.Reliability{}
	reliabilityLayer.SetSDSRetrievalHintPersistence(hintStore)
	require.NoError(t, reliabilityLayer.RecordSDSRetrievalHints("channel", messageID, [][]byte{envelopeHash}))

	tracker := &sdsApplicationMessageIDTrackerStub{
		applicationID: []byte{0xcc},
		applicationOK: true,
	}
	publishSDSMessageDelivered(zap.NewNop(), tracker, pubsub.NewPublisher(), "0x0102")
	require.True(t, tracker.called, "delivery confirmation should consume the in-memory alias")

	core := &Core{
		config: config{logger: zap.NewNop()},
		stack:  &common.MessagingStack{Reliability: reliabilityLayer},
	}
	hintBytes := core.resolveSDSRetrievalHintForChannel("channel", "0x0102")

	var hint reliabilitypb.RetrievalHint
	require.NoError(t, proto.Unmarshal(hintBytes, &hint))
	require.Equal(t, [][]byte{[]byte("0xaabb")}, hint.EnvelopeHashes)
}

func TestPublishSDSMessageDelivered_TrackedAliasPublishesApplicationMessageID(t *testing.T) {
	publisher := pubsub.NewPublisher()
	deliveredMessages, unsubscribe := pubsub.Subscribe[events.DeliveredMessage](publisher, 1)
	defer unsubscribe()

	tracker := &sdsApplicationMessageIDTrackerStub{
		applicationID: []byte{0xaa, 0xbb},
		applicationOK: true,
	}

	publishSDSMessageDelivered(zap.NewNop(), tracker, publisher, "0x0102")

	require.True(t, tracker.called)
	require.Equal(t, []byte{0x01, 0x02}, tracker.receivedID)
	require.Equal(t, events.DeliveredMessage{MessageIDs: [][]byte{{0xaa, 0xbb}}}, <-deliveredMessages)
}

func TestPublishSDSMessageDelivered_IgnoresInvalidAndUntrackedIDs(t *testing.T) {
	publisher := pubsub.NewPublisher()
	deliveredMessages, unsubscribe := pubsub.Subscribe[events.DeliveredMessage](publisher, 1)
	defer unsubscribe()

	tracker := &sdsApplicationMessageIDTrackerStub{}
	publishSDSMessageDelivered(zap.NewNop(), tracker, publisher, "not-hex")
	require.False(t, tracker.called)

	publishSDSMessageDelivered(zap.NewNop(), tracker, publisher, "0x0102")
	require.True(t, tracker.called)
	require.Equal(t, []byte{0x01, 0x02}, tracker.receivedID)

	select {
	case delivered := <-deliveredMessages:
		t.Fatalf("unexpected delivered message: %#v", delivered)
	default:
	}
}
