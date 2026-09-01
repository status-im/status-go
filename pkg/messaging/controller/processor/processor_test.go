package processor

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"math"
	"sync"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/golang/protobuf/proto"
	bindata "github.com/status-im/migrate/v4/source/go_bindata"
	mvdsnode "github.com/status-im/mvds/node"
	mvdsmigrations "github.com/status-im/mvds/persistenceutil"
	mvdsproto "github.com/status-im/mvds/protobuf"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/crypto"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/instrumentation/trace"
	"github.com/status-im/status-go/internal/testutils"
	common "github.com/status-im/status-go/pkg/messaging/common"
	commonmigrations "github.com/status-im/status-go/pkg/messaging/common/migrations"
	encryption "github.com/status-im/status-go/pkg/messaging/layers/encryption"
	encryptionmigrations "github.com/status-im/status-go/pkg/messaging/layers/encryption/migrations"
	"github.com/status-im/status-go/pkg/messaging/layers/reliability"
	reliabilitypb "github.com/status-im/status-go/pkg/messaging/layers/reliability/protobuf"
	segmentation "github.com/status-im/status-go/pkg/messaging/layers/segmentation"
	segmentationmigrations "github.com/status-im/status-go/pkg/messaging/layers/segmentation/migrations"
	transport "github.com/status-im/status-go/pkg/messaging/layers/transport"
	transportmigrations "github.com/status-im/status-go/pkg/messaging/layers/transport/migrations"
	"github.com/status-im/status-go/pkg/messaging/types"
	"github.com/status-im/status-go/pkg/messaging/waku"
)

func TestProcessorSuite(t *testing.T) {
	suite.Run(t, new(ProcessorSuite))
}

type ProcessorSuite struct {
	suite.Suite

	processor   *Processor
	testPayload []byte
	logger      *zap.Logger
}

func (s *ProcessorSuite) newStandaloneReliability() *reliability.Reliability {
	db, err := testutils.SetupTestMemorySQLDB(testutils.NewTestDBInitializer(nil))
	s.Require().NoError(err)

	s.T().Cleanup(func() {
		s.Require().NoError(db.Close())
	})

	err = mvdsmigrations.Migrate(db)
	s.Require().NoError(err)

	identity, err := crypto.GenerateKey()
	s.Require().NoError(err)

	r, err := reliability.NewReliability(
		mvdsnode.NewSQLitePersistence(db),
		identity,
		func(string, []string, string) error { return nil },
		s.logger,
	)
	s.Require().NoError(err)

	s.T().Cleanup(r.Close)

	return r
}

func (s *ProcessorSuite) SetupTest() {
	s.testPayload = []byte(gofakeit.Word())

	var err error

	s.logger, err = zap.NewDevelopment()
	s.Require().NoError(err)

	identity, err := crypto.GenerateKey()
	s.Require().NoError(err)

	db, err := testutils.SetupTestMemorySQLDB(testutils.NewTestDBInitializer([]*bindata.AssetSource{
		{
			Names:     transportmigrations.AssetNames(),
			AssetFunc: transportmigrations.Asset,
		},
		{
			Names:     segmentationmigrations.AssetNames(),
			AssetFunc: segmentationmigrations.Asset,
		},
		{
			Names:     encryptionmigrations.AssetNames(),
			AssetFunc: encryptionmigrations.Asset,
		},
		{
			Names:     commonmigrations.AssetNames(),
			AssetFunc: commonmigrations.Asset,
		},
	}))
	s.Require().NoError(err)

	err = mvdsmigrations.Migrate(db)
	s.Require().NoError(err)

	stack := &common.MessagingStack{}

	wakuConfig := waku.DefaultConfig
	shh, err := waku.New(
		nil,
		&wakuConfig,
		s.logger,
		nil,
	)
	s.Require().NoError(err)
	s.Require().NoError(shh.Start())

	stack.Transport, err = transport.NewTransport(
		shh,
		identity,
		transport.NewSQLiteKeysPersistence(db),
		transport.NewSQLiteProcessedMessageIDsCachePersistence(db),
		&transport.EnvelopesMonitorConfig{},
		s.logger,
	)
	s.Require().NoError(err)

	stack.Segmentation = segmentation.NewSegmenter(
		segmentation.NewSQLitePersistence(db),
		s.logger,
	)

	stack.Encryption = encryption.New(
		encryption.NewSQLitePersistence(db),
		"installation-1",
		s.logger,
		trace.NewNoopTracer(),
	)

	stack.Reliability, err = reliability.NewReliability(
		mvdsnode.NewSQLitePersistence(db),
		identity,
		func(string, []string, string) error { return nil },
		s.logger,
	)
	s.Require().NoError(err)

	err = stack.Reliability.Start(func(*ecdsa.PublicKey, []byte, [][]byte) error { return nil })
	s.Require().NoError(err)

	s.T().Cleanup(stack.Reliability.Close)

	s.processor = NewProcessor(
		identity,
		stack,
		common.NewSQLiteMessageConfirmationPersistence(db),
		common.NewSQLiteHashRatchetPersistence(db),
		s.logger,
		trace.NewNoopTracer(),
	)
}

func (s *ProcessorSuite) TestProcessMessage() {
	senderKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	encodedPayload := s.testPayload

	message := &types.ReceivedMessage{}
	message.Sig = crypto.FromECDSAPub(&senderKey.PublicKey)
	message.Payload = s.testPayload

	response, err := s.processor.ProcessMessage(message)
	s.Require().NoError(err)
	decodedMessages := response.Messages

	s.Require().Equal(1, len(decodedMessages))
	s.Require().Equal(&senderKey.PublicKey, decodedMessages[0].SigPubKey())
	s.Require().Equal(encodedPayload, decodedMessages[0].EncryptionLayer.Payload)
}

func (s *ProcessorSuite) TestProcessMessageDatasync() {
	senderKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	dataSyncMessage := mvdsproto.Payload{
		Messages: []*mvdsproto.Message{
			{Body: s.testPayload},
		},
	}
	marshalledDataSyncMessage, err := proto.Marshal(&dataSyncMessage)
	s.Require().NoError(err)
	message := &types.ReceivedMessage{}
	message.Sig = crypto.FromECDSAPub(&senderKey.PublicKey)
	message.Payload = marshalledDataSyncMessage

	response, err := s.processor.ProcessMessage(message)
	s.Require().NoError(err)
	decodedMessages := response.Messages

	// We send two messages, the unwrapped one will be attributed to the relayer, while the wrapped one will be attributed to the author
	s.Require().Equal(1, len(decodedMessages))
	s.Require().Equal(&senderKey.PublicKey, decodedMessages[0].SigPubKey())
	s.Require().Equal(s.testPayload, decodedMessages[0].EncryptionLayer.Payload)
}

func (s *ProcessorSuite) TestProcessMessageDatasyncEncrypted() {
	senderKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	dataSyncMessage := mvdsproto.Payload{
		Messages: []*mvdsproto.Message{
			{Body: s.testPayload},
		},
	}
	marshalledDataSyncMessage, err := proto.Marshal(&dataSyncMessage)
	s.Require().NoError(err)

	// Create sender encryption protocol.
	senderDatabase, err := testutils.SetupTestMemorySQLDB(testutils.NewTestDBInitializer([]*bindata.AssetSource{
		{
			Names:     encryptionmigrations.AssetNames(),
			AssetFunc: encryptionmigrations.Asset,
		},
	}))
	s.Require().NoError(err)

	senderEncryptionProtocol := encryption.New(
		encryption.NewSQLitePersistence(senderDatabase),
		"installation-2",
		s.logger,
		trace.NewNoopTracer(),
	)

	messageSpec, err := senderEncryptionProtocol.BuildEncryptedMessage(
		senderKey,
		&s.processor.identity.PublicKey,
		marshalledDataSyncMessage,
	)
	s.Require().NoError(err)

	encryptedPayload, err := proto.Marshal(messageSpec.Message)
	s.Require().NoError(err)

	message := &types.ReceivedMessage{}
	message.Sig = crypto.FromECDSAPub(&senderKey.PublicKey)
	message.Payload = encryptedPayload

	response, err := s.processor.ProcessMessage(message)
	s.Require().NoError(err)
	decodedMessages := response.Messages

	// We send two messages, the unwrapped one will be attributed to the relayer,
	// while the wrapped one will be attributed to the author.
	s.Require().Equal(1, len(decodedMessages))
	s.Require().Equal(&senderKey.PublicKey, decodedMessages[0].SigPubKey())
	s.Require().Equal(s.testPayload, decodedMessages[0].EncryptionLayer.Payload)
}

func (s *ProcessorSuite) TestHandleOutOfOrderHashRatchet() {
	groupID := []byte("group-id")
	senderKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	// Create sender encryption protocol.
	senderDatabase, err := testutils.SetupTestMemorySQLDB(testutils.NewTestDBInitializer([]*bindata.AssetSource{
		{
			Names:     encryptionmigrations.AssetNames(),
			AssetFunc: encryptionmigrations.Asset,
		},
	}))
	s.Require().NoError(err)

	senderEncryptionProtocol := encryption.New(
		encryption.NewSQLitePersistence(senderDatabase),
		"installation-2",
		s.logger,
		trace.NewNoopTracer(),
	)

	ratchet, err := senderEncryptionProtocol.GenerateHashRatchetKey(groupID)
	s.Require().NoError(err)

	ratchets := []*encryption.HashRatchetKeyCompatibility{ratchet}

	hashRatchetKeyExchangeMessage, err := senderEncryptionProtocol.BuildHashRatchetKeyExchangeMessage(context.Background(), senderKey, &s.processor.identity.PublicKey, groupID, ratchets)
	s.Require().NoError(err)

	encryptedPayload1, err := proto.Marshal(hashRatchetKeyExchangeMessage.Message)
	s.Require().NoError(err)

	messageSpec2, err := senderEncryptionProtocol.BuildHashRatchetMessage(
		groupID,
		s.testPayload,
	)
	s.Require().NoError(err)

	encryptedPayload2, err := proto.Marshal(messageSpec2.Message)
	s.Require().NoError(err)

	message := &types.ReceivedMessage{}
	message.Sig = crypto.FromECDSAPub(&senderKey.PublicKey)
	message.Hash = []byte{0x1}
	message.Payload = encryptedPayload2

	_, err = s.processor.processMessage(message)
	s.Require().NoError(err)

	keyID, err := ratchet.GetKeyID()
	s.Require().NoError(err)

	msgs, err := s.processor.hashRatchetStorage.GetMessages(keyID)
	s.Require().NoError(err)

	s.Require().Len(msgs, 1)

	message = &types.ReceivedMessage{}
	message.Sig = crypto.FromECDSAPub(&senderKey.PublicKey)
	message.Hash = []byte{0x2}
	message.Payload = encryptedPayload1

	response, err := s.processor.ProcessMessage(message)
	s.Require().NoError(err)
	decodedMessages2 := response.Messages
	s.Require().NotNil(decodedMessages2)

	// It should have 2 messages, the key exchange and the one from the database
	s.Require().Len(decodedMessages2, 2)

	// it deletes the messages after being processed
	msgs, err = s.processor.hashRatchetStorage.GetMessages(keyID)
	s.Require().NoError(err)

	s.Require().Len(msgs, 0)
}

// A queued hash ratchet message can legitimately fail to complete on replay —
// here, it is encrypted with a ratchet key that still hasn't arrived, so
// processing it re-queues it and yields no response. That must not crash the
// replay and must not lose the message.
func (s *ProcessorSuite) TestQueuedHashRatchetMessageStillMissingItsKeySurvivesReplay() {
	groupID := []byte("group-id")
	otherGroupID := []byte("group-id-other")
	senderKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	senderDatabase, err := testutils.SetupTestMemorySQLDB(testutils.NewTestDBInitializer([]*bindata.AssetSource{
		{
			Names:     encryptionmigrations.AssetNames(),
			AssetFunc: encryptionmigrations.Asset,
		},
	}))
	s.Require().NoError(err)

	senderEncryptionProtocol := encryption.New(
		encryption.NewSQLitePersistence(senderDatabase),
		"installation-2",
		s.logger,
		trace.NewNoopTracer(),
	)

	ratchet, err := senderEncryptionProtocol.GenerateHashRatchetKey(groupID)
	s.Require().NoError(err)
	keyID, err := ratchet.GetKeyID()
	s.Require().NoError(err)

	// A second ratchet whose key exchange is never delivered to the receiver.
	otherRatchet, err := senderEncryptionProtocol.GenerateHashRatchetKey(otherGroupID)
	s.Require().NoError(err)
	otherKeyID, err := otherRatchet.GetKeyID()
	s.Require().NoError(err)

	hashRatchetKeyExchangeMessage, err := senderEncryptionProtocol.BuildHashRatchetKeyExchangeMessage(context.Background(), senderKey, &s.processor.identity.PublicKey, groupID, []*encryption.HashRatchetKeyCompatibility{ratchet})
	s.Require().NoError(err)
	keyExchangePayload, err := proto.Marshal(hashRatchetKeyExchangeMessage.Message)
	s.Require().NoError(err)

	dataMessageSpec, err := senderEncryptionProtocol.BuildHashRatchetMessage(groupID, s.testPayload)
	s.Require().NoError(err)
	dataPayload, err := proto.Marshal(dataMessageSpec.Message)
	s.Require().NoError(err)

	strayMessageSpec, err := senderEncryptionProtocol.BuildHashRatchetMessage(otherGroupID, s.testPayload)
	s.Require().NoError(err)
	strayPayload, err := proto.Marshal(strayMessageSpec.Message)
	s.Require().NoError(err)

	// The data message arrives before its key and gets queued.
	message := &types.ReceivedMessage{}
	message.Sig = crypto.FromECDSAPub(&senderKey.PublicKey)
	message.Hash = []byte{0x1}
	message.Payload = dataPayload

	_, err = s.processor.processMessage(message)
	s.Require().NoError(err)

	// A message for the other, still missing ratchet key sits in the same queue.
	strayMessage := &types.ReceivedMessage{}
	strayMessage.Sig = crypto.FromECDSAPub(&senderKey.PublicKey)
	strayMessage.Hash = []byte{0x2}
	strayMessage.Payload = strayPayload

	err = s.processor.hashRatchetStorage.SaveMessage(groupID, keyID, strayMessage)
	s.Require().NoError(err)

	// The key arrives: the replay must decode the data message, not crash on the
	// stray one, and re-queue the stray one under its own key.
	message = &types.ReceivedMessage{}
	message.Sig = crypto.FromECDSAPub(&senderKey.PublicKey)
	message.Hash = []byte{0x3}
	message.Payload = keyExchangePayload

	response, err := s.processor.ProcessMessage(message)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	// The key exchange itself and the queued data message.
	s.Require().Len(response.Messages, 2)

	// The stray message is not lost: it waits for its own key.
	msgs, err := s.processor.hashRatchetStorage.GetMessages(otherKeyID)
	s.Require().NoError(err)
	s.Require().Len(msgs, 1)
}

func (s *ProcessorSuite) TestHandleSegmentMessages() {
	s.T().Skip("flaky test, tracked in https://github.com/status-im/status-go/issues/7791")

	senderKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	segmentedMessages, err := s.processor.stack.Segmentation.Segment(s.testPayload, int(math.Ceil(float64(len(s.testPayload))/2)))
	s.Require().NoError(err)
	s.Require().Len(segmentedMessages, 2)

	message := &types.ReceivedMessage{}
	message.Sig = crypto.FromECDSAPub(&senderKey.PublicKey)
	message.Payload = segmentedMessages[0]

	// First segment is received, no messages are decoded
	response, err := s.processor.ProcessMessage(message)
	s.Require().NoError(err)
	s.Require().Nil(response)

	// Second (and final) segment is received, reassembled message is decoded
	message.Payload = segmentedMessages[1]
	response, err = s.processor.ProcessMessage(message)
	s.Require().NoError(err)

	decodedMessages := response.Messages
	s.Require().Len(decodedMessages, 1)
	s.Require().Equal(&senderKey.PublicKey, decodedMessages[0].SigPubKey())
	s.Require().Equal(s.testPayload, decodedMessages[0].EncryptionLayer.Payload)

	// Receiving another segment after reassembly should be ignored as a duplicate.
	response, err = s.processor.ProcessMessage(message)
	s.Require().NoError(err)
	s.Require().Nil(response)
}

func (s *ProcessorSuite) TestGetEphemeralKey() {
	keyMap := make(map[string]bool)
	for i := 0; i < maxNumOfEphemeralKeys; i++ {
		key, err := s.processor.GetEphemeralKey()
		s.Require().NoError(err)
		s.Require().NotNil(key)
		keyMap[crypto.PubkeyToHex(&key.PublicKey)] = true
	}
	s.Require().Len(keyMap, maxNumOfEphemeralKeys)
	// Add one more
	key, err := s.processor.GetEphemeralKey()
	s.Require().NoError(err)
	s.Require().NotNil(key)

	s.Require().True(keyMap[crypto.PubkeyToHex(&key.PublicKey)])
}

func (s *ProcessorSuite) TestSDSWrappedMessages() {
	payload := []byte("hello")
	sdsChannelID := "community123channel-1"

	wrappedPayload, _, err := s.processor.stack.Reliability.WrapPayloadForSDS(payload, sdsChannelID)
	s.Require().NoError(err)
	s.Require().True(len(wrappedPayload) > 0)

	receivedMsg := types.Message{
		EncryptionLayer: types.EncryptionLayer{
			Payload: wrappedPayload,
		},
	}

	err = s.processor.processSDSLayer(&receivedMsg)
	s.Require().NoError(err)
	s.Require().Equal(payload, receivedMsg.EncryptionLayer.Payload)

	anotherPayload := []byte("another-message")
	receivedMsg2 := types.Message{
		EncryptionLayer: types.EncryptionLayer{
			Payload: anotherPayload,
		},
	}
	err = s.processor.processSDSLayer(&receivedMsg2)
	s.Require().NoError(err)
	s.Require().Equal(anotherPayload, receivedMsg2.EncryptionLayer.Payload)
}

func (s *ProcessorSuite) TestSDSMissingDependencyTriggersFetchHintsAndRecoversPayloads() {
	senderReliability := s.newStandaloneReliability()
	receiverReliability := s.newStandaloneReliability()

	channelID := "community123general"
	oldPayload := []byte("old-community-message")
	newPayload := []byte("new-community-message")

	expectedOldMessageID := cryptotypes.EncodeHex(crypto.Keccak256(oldPayload))
	expectedNewMessageID := cryptotypes.EncodeHex(crypto.Keccak256(newPayload))
	expectedOldHint := "store-hash-" + expectedOldMessageID

	senderReliability.SetRetrievalHintProvider(func(messageID string) []byte {
		hint, err := proto.Marshal(&reliabilitypb.RetrievalHint{
			EnvelopeHashes: [][]byte{[]byte("store-hash-" + messageID)},
		})
		s.Require().NoError(err)
		return hint
	})

	type missingDepsCall struct {
		messageID string
		deps      []string
		channelID string
	}

	var (
		callsMu sync.Mutex
		calls   []missingDepsCall
	)

	receiverReliability.SetMissingDependenciesHandler(func(messageID string, missingDeps []string, missingDepsChannelID string) error {
		callsMu.Lock()
		calls = append(calls, missingDepsCall{
			messageID: messageID,
			deps:      append([]string(nil), missingDeps...),
			channelID: missingDepsChannelID,
		})
		callsMu.Unlock()
		return nil
	})

	wrappedOldPayload, _, err := senderReliability.WrapPayloadForSDS(oldPayload, channelID)
	s.Require().NoError(err)
	wrappedNewPayload, _, err := senderReliability.WrapPayloadForSDS(newPayload, channelID)
	s.Require().NoError(err)

	originalReliability := s.processor.stack.Reliability
	s.processor.stack.Reliability = receiverReliability
	defer func() {
		s.processor.stack.Reliability = originalReliability
	}()

	newMessageFirst := types.Message{
		EncryptionLayer: types.EncryptionLayer{Payload: wrappedNewPayload},
	}
	err = s.processor.processSDSLayer(&newMessageFirst)
	s.Require().NoError(err)

	err = testutils.RetryWithBackOff(func() error {
		callsMu.Lock()
		defer callsMu.Unlock()

		if len(calls) == 0 {
			return errors.New("missing dependencies callback not triggered")
		}

		return nil
	})
	s.Require().NoError(err)

	callsMu.Lock()
	firstCall := calls[0]
	callsMu.Unlock()

	s.Require().Equal(expectedNewMessageID, firstCall.messageID)
	s.Require().Equal(channelID, firstCall.channelID)
	s.Require().Contains(firstCall.deps, expectedOldHint)

	recovered := map[string]bool{}
	if bytes.Equal(newMessageFirst.EncryptionLayer.Payload, newPayload) {
		recovered["new"] = true
	}

	oldMessage := types.Message{
		EncryptionLayer: types.EncryptionLayer{Payload: wrappedOldPayload},
	}
	err = s.processor.processSDSLayer(&oldMessage)
	s.Require().NoError(err)
	if bytes.Equal(oldMessage.EncryptionLayer.Payload, oldPayload) {
		recovered["old"] = true
	}

	newMessageReplay := types.Message{
		EncryptionLayer: types.EncryptionLayer{Payload: wrappedNewPayload},
	}
	err = s.processor.processSDSLayer(&newMessageReplay)
	s.Require().NoError(err)
	if bytes.Equal(newMessageReplay.EncryptionLayer.Payload, newPayload) {
		recovered["new"] = true
	}

	s.Require().True(recovered["old"], "old payload should be recoverable after dependency handling")
	s.Require().True(recovered["new"], "new payload should be recoverable after dependency handling")
}
