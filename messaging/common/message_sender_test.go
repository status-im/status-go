package common

import (
	"math"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p/core/peer"
	bindata "github.com/status-im/migrate/v4/source/go_bindata"
	mvdsnode "github.com/status-im/mvds/node"
	mvdsmigrations "github.com/status-im/mvds/persistenceutil"
	datasyncproto "github.com/status-im/mvds/protobuf"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/status-im/status-go/crypto"
	messagesendermigrations "github.com/status-im/status-go/messaging/common/migrations"
	"github.com/status-im/status-go/messaging/layers/encryption"
	encryptionmigrations "github.com/status-im/status-go/messaging/layers/encryption/migrations"
	"github.com/status-im/status-go/messaging/layers/segmentation"
	segmentationmigrations "github.com/status-im/status-go/messaging/layers/segmentation/migrations"
	"github.com/status-im/status-go/messaging/layers/transport"
	transportmigrations "github.com/status-im/status-go/messaging/layers/transport/migrations"
	messagingtypes "github.com/status-im/status-go/messaging/types"
	wakuv2 "github.com/status-im/status-go/messaging/waku"
	wakutypes "github.com/status-im/status-go/messaging/waku/types"
	"github.com/status-im/status-go/protocol/protobuf"
	v1protocol "github.com/status-im/status-go/protocol/v1"
	"github.com/status-im/status-go/t/helpers"
)

func TestMessageSenderSuite(t *testing.T) {
	suite.Run(t, new(MessageSenderSuite))
}

type MessageSenderSuite struct {
	suite.Suite

	sender      *MessageSender
	testMessage protobuf.ChatMessage
	logger      *zap.Logger
}

func (s *MessageSenderSuite) SetupTest() {
	s.testMessage = protobuf.ChatMessage{
		Text:        "abc123",
		ChatId:      "testing-adamb",
		ContentType: protobuf.ChatMessage_TEXT_PLAIN,
		MessageType: protobuf.MessageType_PUBLIC_GROUP,
		Clock:       154593077368201,
		Timestamp:   1545930773682,
	}

	var err error

	s.logger, err = zap.NewDevelopment()
	s.Require().NoError(err)

	identity, err := crypto.GenerateKey()
	s.Require().NoError(err)

	db, err := helpers.SetupTestMemorySQLDB(helpers.NewTestDBInitializer([]*bindata.AssetSource{
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
			Names:     messagesendermigrations.AssetNames(),
			AssetFunc: messagesendermigrations.Asset,
		},
	}))
	s.Require().NoError(err)

	err = mvdsmigrations.Migrate(db)
	s.Require().NoError(err)

	encryptionProtocol := encryption.New(
		encryption.NewSQLitePersistence(db),
		"installation-1",
		s.logger,
	)

	wakuConfig := wakuv2.DefaultConfig
	shh, err := wakuv2.New(
		nil,
		&wakuConfig,
		s.logger,
		nil,
		nil,
		func([]byte, peer.AddrInfo, error) {},
		nil,
	)
	s.Require().NoError(err)
	s.Require().NoError(shh.Start())

	transport, err := transport.NewTransport(
		shh,
		identity,
		transport.NewSQLiteKeysPersistence(db),
		transport.NewSQLiteProcessedMessageIDsCachePersistence(db),
		&transport.EnvelopesMonitorConfig{},
		s.logger,
	)
	s.Require().NoError(err)

	s.sender, err = NewMessageSender(
		identity,
		NewSQLiteMessageSenderPersistence(db),
		mvdsnode.NewSQLitePersistence(db),
		segmentation.NewSQLitePersistence(db),
		transport,
		encryptionProtocol,
		s.logger,
	)
	s.Require().NoError(err)
}

func (s *MessageSenderSuite) TearDownTest() {
	_ = s.logger.Sync()
}

func (s *MessageSenderSuite) TestHandleDecodedMessagesWrapped() {
	relayerKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	authorKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	encodedPayload, err := proto.Marshal(&s.testMessage)
	s.Require().NoError(err)

	wrappedPayload, err := v1protocol.WrapMessageV1(encodedPayload, protobuf.ApplicationMetadataMessage_CHAT_MESSAGE, authorKey)
	s.Require().NoError(err)

	message := &messagingtypes.ReceivedMessage{}
	message.Sig = crypto.FromECDSAPub(&relayerKey.PublicKey)
	message.Payload = wrappedPayload

	response, err := s.sender.HandleMessages(message)
	s.Require().NoError(err)
	decodedMessages := response.StatusMessages

	s.Require().Equal(1, len(decodedMessages))
	s.Require().Equal(&authorKey.PublicKey, decodedMessages[0].SigPubKey())
	s.Require().Equal(v1protocol.MessageID(&authorKey.PublicKey, wrappedPayload), decodedMessages[0].ApplicationLayer.ID)
	s.Require().Equal(encodedPayload, decodedMessages[0].ApplicationLayer.Payload)
	s.Require().Equal(protobuf.ApplicationMetadataMessage_CHAT_MESSAGE, decodedMessages[0].ApplicationLayer.Type)
}

func (s *MessageSenderSuite) TestHandleDecodedMessagesDatasync() {
	relayerKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	authorKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	encodedPayload, err := proto.Marshal(&s.testMessage)
	s.Require().NoError(err)

	wrappedPayload, err := v1protocol.WrapMessageV1(encodedPayload, protobuf.ApplicationMetadataMessage_CHAT_MESSAGE, authorKey)
	s.Require().NoError(err)

	dataSyncMessage := datasyncproto.Payload{
		Messages: []*datasyncproto.Message{
			{Body: wrappedPayload},
		},
	}
	marshalledDataSyncMessage, err := proto.Marshal(&dataSyncMessage)
	s.Require().NoError(err)
	message := &messagingtypes.ReceivedMessage{}
	message.Sig = crypto.FromECDSAPub(&relayerKey.PublicKey)
	message.Payload = marshalledDataSyncMessage

	err = s.sender.StartReliability(nil, nil)
	s.Require().NoError(err)

	response, err := s.sender.HandleMessages(message)
	s.Require().NoError(err)
	decodedMessages := response.StatusMessages

	// We send two messages, the unwrapped one will be attributed to the relayer, while the wrapped one will be attributed to the author
	s.Require().Equal(1, len(decodedMessages))
	s.Require().Equal(&authorKey.PublicKey, decodedMessages[0].SigPubKey())
	s.Require().Equal(v1protocol.MessageID(&authorKey.PublicKey, wrappedPayload), decodedMessages[0].ApplicationLayer.ID)
	s.Require().Equal(encodedPayload, decodedMessages[0].ApplicationLayer.Payload)
	s.Require().Equal(protobuf.ApplicationMetadataMessage_CHAT_MESSAGE, decodedMessages[0].ApplicationLayer.Type)
}

func (s *MessageSenderSuite) TestHandleDecodedMessagesDatasyncEncrypted() {
	relayerKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	authorKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	encodedPayload, err := proto.Marshal(&s.testMessage)
	s.Require().NoError(err)

	wrappedPayload, err := v1protocol.WrapMessageV1(encodedPayload, protobuf.ApplicationMetadataMessage_CHAT_MESSAGE, authorKey)
	s.Require().NoError(err)

	dataSyncMessage := datasyncproto.Payload{
		Messages: []*datasyncproto.Message{
			{Body: wrappedPayload},
		},
	}
	marshalledDataSyncMessage, err := proto.Marshal(&dataSyncMessage)
	s.Require().NoError(err)

	// Create sender encryption protocol.
	senderDatabase, err := helpers.SetupTestMemorySQLDB(helpers.NewTestDBInitializer([]*bindata.AssetSource{
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
	)

	messageSpec, err := senderEncryptionProtocol.BuildEncryptedMessage(
		relayerKey,
		&s.sender.identity.PublicKey,
		marshalledDataSyncMessage,
	)
	s.Require().NoError(err)

	encryptedPayload, err := proto.Marshal(messageSpec.Message)
	s.Require().NoError(err)

	message := &messagingtypes.ReceivedMessage{}
	message.Sig = crypto.FromECDSAPub(&relayerKey.PublicKey)
	message.Payload = encryptedPayload

	err = s.sender.StartReliability(nil, nil)
	s.Require().NoError(err)

	response, err := s.sender.HandleMessages(message)
	s.Require().NoError(err)
	decodedMessages := response.StatusMessages

	// We send two messages, the unwrapped one will be attributed to the relayer,
	// while the wrapped one will be attributed to the author.
	s.Require().Equal(1, len(decodedMessages))
	s.Require().Equal(&authorKey.PublicKey, decodedMessages[0].SigPubKey())
	s.Require().Equal(v1protocol.MessageID(&authorKey.PublicKey, wrappedPayload), decodedMessages[0].ApplicationLayer.ID)
	s.Require().Equal(encodedPayload, decodedMessages[0].ApplicationLayer.Payload)
	s.Require().Equal(protobuf.ApplicationMetadataMessage_CHAT_MESSAGE, decodedMessages[0].ApplicationLayer.Type)
}

func (s *MessageSenderSuite) TestHandleOutOfOrderHashRatchet() {
	groupID := []byte("group-id")
	senderKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	encodedPayload, err := proto.Marshal(&s.testMessage)
	s.Require().NoError(err)

	// Create sender encryption protocol.
	senderDatabase, err := helpers.SetupTestMemorySQLDB(helpers.NewTestDBInitializer([]*bindata.AssetSource{
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
	)

	ratchet, err := senderEncryptionProtocol.GenerateHashRatchetKey(groupID)
	s.Require().NoError(err)

	ratchets := []*encryption.HashRatchetKeyCompatibility{ratchet}

	hashRatchetKeyExchangeMessage, err := senderEncryptionProtocol.BuildHashRatchetKeyExchangeMessage(senderKey, &s.sender.identity.PublicKey, groupID, ratchets)
	s.Require().NoError(err)

	encryptedPayload1, err := proto.Marshal(hashRatchetKeyExchangeMessage.Message)
	s.Require().NoError(err)

	wrappedPayload2, err := v1protocol.WrapMessageV1(encodedPayload, protobuf.ApplicationMetadataMessage_CHAT_MESSAGE, senderKey)
	s.Require().NoError(err)

	messageSpec2, err := senderEncryptionProtocol.BuildHashRatchetMessage(
		groupID,
		wrappedPayload2,
	)
	s.Require().NoError(err)

	encryptedPayload2, err := proto.Marshal(messageSpec2.Message)
	s.Require().NoError(err)

	message := &messagingtypes.ReceivedMessage{}
	message.Sig = crypto.FromECDSAPub(&senderKey.PublicKey)
	message.Hash = []byte{0x1}
	message.Payload = encryptedPayload2

	_, err = s.sender.HandleMessages(message)
	s.Require().NoError(err)

	keyID, err := ratchet.GetKeyID()
	s.Require().NoError(err)

	msgs, err := s.sender.persistence.GetHashRatchetMessages(keyID)
	s.Require().NoError(err)

	s.Require().Len(msgs, 1)

	message = &messagingtypes.ReceivedMessage{}
	message.Sig = crypto.FromECDSAPub(&senderKey.PublicKey)
	message.Hash = []byte{0x2}
	message.Payload = encryptedPayload1

	response, err := s.sender.HandleMessages(message)
	s.Require().NoError(err)
	decodedMessages2 := response.StatusMessages
	s.Require().NotNil(decodedMessages2)

	// It should have 2 messages, the key exchange and the one from the database
	s.Require().Len(decodedMessages2, 2)

	// it deletes the messages after being processed
	msgs, err = s.sender.persistence.GetHashRatchetMessages(keyID)
	s.Require().NoError(err)

	s.Require().Len(msgs, 0)
}

func (s *MessageSenderSuite) TestHandleSegmentMessages() {
	relayerKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	authorKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	encodedPayload, err := proto.Marshal(&s.testMessage)
	s.Require().NoError(err)

	wrappedPayload, err := v1protocol.WrapMessageV1(encodedPayload, protobuf.ApplicationMetadataMessage_CHAT_MESSAGE, authorKey)
	s.Require().NoError(err)

	segmentedMessages, err := s.sender.segmentMessageWithSize(&wakutypes.NewMessage{Payload: wrappedPayload}, int(math.Ceil(float64(len(wrappedPayload))/2)))
	s.Require().NoError(err)
	s.Require().Len(segmentedMessages, 2)

	message := &messagingtypes.ReceivedMessage{}
	message.Sig = crypto.FromECDSAPub(&relayerKey.PublicKey)
	message.Payload = segmentedMessages[0].Payload

	// First segment is received, no messages are decoded
	response, err := s.sender.HandleMessages(message)
	s.Require().NoError(err)
	s.Require().Nil(response)

	// Second (and final) segment is received, reassembled message is decoded
	message.Payload = segmentedMessages[1].Payload
	response, err = s.sender.HandleMessages(message)
	s.Require().NoError(err)

	decodedMessages := response.StatusMessages
	s.Require().Len(decodedMessages, 1)
	s.Require().Equal(&authorKey.PublicKey, decodedMessages[0].SigPubKey())
	s.Require().Equal(v1protocol.MessageID(&authorKey.PublicKey, wrappedPayload), decodedMessages[0].ApplicationLayer.ID)
	s.Require().Equal(encodedPayload, decodedMessages[0].ApplicationLayer.Payload)
	s.Require().Equal(protobuf.ApplicationMetadataMessage_CHAT_MESSAGE, decodedMessages[0].ApplicationLayer.Type)

	// Receiving another segment after the message has been reassembled is considered an error
	_, err = s.sender.HandleMessages(message)
	s.Require().ErrorIs(err, segmentation.ErrAlreadyCompleted)
}

func (s *MessageSenderSuite) TestGetEphemeralKey() {
	keyMap := make(map[string]bool)
	for i := 0; i < maxMessageSenderEphemeralKeys; i++ {
		key, err := s.sender.GetEphemeralKey()
		s.Require().NoError(err)
		s.Require().NotNil(key)
		keyMap[crypto.PubkeyToHex(&key.PublicKey)] = true
	}
	s.Require().Len(keyMap, maxMessageSenderEphemeralKeys)
	// Add one more
	key, err := s.sender.GetEphemeralKey()
	s.Require().NoError(err)
	s.Require().NotNil(key)

	s.Require().True(keyMap[crypto.PubkeyToHex(&key.PublicKey)])
}
