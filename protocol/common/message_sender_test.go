package common

import (
	"path/filepath"
	"testing"

	transport2 "github.com/status-im/status-go/protocol/transport"

	"github.com/status-im/status-go/waku"

	"github.com/golang/protobuf/proto"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	datasyncproto "github.com/vacp2p/mvds/protobuf"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/encryption"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/sqlite"
	v1protocol "github.com/status-im/status-go/protocol/v1"
)

func TestMessageSenderSuite(t *testing.T) {
	suite.Run(t, new(MessageSenderSuite))
}

type MessageSenderSuite struct {
	suite.Suite

	sender      *MessageSender
	tmpDir      string
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

	s.tmpDir = s.T().TempDir()

	identity, err := crypto.GenerateKey()
	s.Require().NoError(err)

	database, err := sqlite.Open(filepath.Join(s.tmpDir, "sender-test.sql"), "some-key", sqlite.ReducedKDFIterationsNumber)
	s.Require().NoError(err)

	encryptionProtocol := encryption.New(
		database,
		"installation-1",
		s.logger,
	)

	wakuConfig := waku.DefaultConfig
	wakuConfig.MinimumAcceptedPoW = 0
	shh := waku.New(&wakuConfig, s.logger)
	s.Require().NoError(shh.Start())

	whisperTransport, err := transport2.NewTransport(
		gethbridge.NewGethWakuWrapper(shh),
		identity,
		database,
		"waku_keys",
		nil,
		nil,
		s.logger,
	)
	s.Require().NoError(err)

	s.sender, err = NewMessageSender(
		identity,
		database,
		encryptionProtocol,
		whisperTransport,
		s.logger,
		FeatureFlags{},
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

	message := &types.Message{}
	message.Sig = crypto.FromECDSAPub(&relayerKey.PublicKey)
	message.Payload = wrappedPayload

	decodedMessages, _, err := s.sender.HandleMessages(message, true)
	s.Require().NoError(err)

	s.Require().Equal(1, len(decodedMessages))
	s.Require().Equal(&authorKey.PublicKey, decodedMessages[0].SigPubKey())
	s.Require().Equal(v1protocol.MessageID(&authorKey.PublicKey, wrappedPayload), decodedMessages[0].ID)
	parsedMessage := decodedMessages[0].ParsedMessage.Interface().(protobuf.ChatMessage)
	s.Require().Equal(encodedPayload, decodedMessages[0].UnwrappedPayload)
	s.Require().True(proto.Equal(&s.testMessage, &parsedMessage))
	s.Require().Equal(protobuf.ApplicationMetadataMessage_CHAT_MESSAGE, decodedMessages[0].Type)
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
	message := &types.Message{}
	message.Sig = crypto.FromECDSAPub(&relayerKey.PublicKey)
	message.Payload = marshalledDataSyncMessage

	decodedMessages, _, err := s.sender.HandleMessages(message, true)
	s.Require().NoError(err)

	// We send two messages, the unwrapped one will be attributed to the relayer, while the wrapped one will be attributed to the author
	s.Require().Equal(1, len(decodedMessages))
	s.Require().Equal(&authorKey.PublicKey, decodedMessages[0].SigPubKey())
	s.Require().Equal(v1protocol.MessageID(&authorKey.PublicKey, wrappedPayload), decodedMessages[0].ID)
	s.Require().Equal(encodedPayload, decodedMessages[0].UnwrappedPayload)
	parsedMessage := decodedMessages[0].ParsedMessage.Interface().(protobuf.ChatMessage)
	s.Require().True(proto.Equal(&s.testMessage, &parsedMessage))
	s.Require().Equal(protobuf.ApplicationMetadataMessage_CHAT_MESSAGE, decodedMessages[0].Type)
}

func (s *MessageSenderSuite) CalculatePoWTest() {
	largeSizePayload := make([]byte, largeSizeInBytes)
	s.Require().Equal(whisperLargeSizePoW, calculatePoW(largeSizePayload))
	normalSizePayload := make([]byte, largeSizeInBytes-1)
	s.Require().Equal(whisperDefaultPoW, calculatePoW(normalSizePayload))

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
	senderDatabase, err := sqlite.Open(filepath.Join(s.tmpDir, "sender.db.sql"), "", sqlite.ReducedKDFIterationsNumber)
	s.Require().NoError(err)
	senderEncryptionProtocol := encryption.New(
		senderDatabase,
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

	message := &types.Message{}
	message.Sig = crypto.FromECDSAPub(&relayerKey.PublicKey)
	message.Payload = encryptedPayload

	decodedMessages, _, err := s.sender.HandleMessages(message, true)
	s.Require().NoError(err)

	// We send two messages, the unwrapped one will be attributed to the relayer,
	// while the wrapped one will be attributed to the author.
	s.Require().Equal(1, len(decodedMessages))
	s.Require().Equal(&authorKey.PublicKey, decodedMessages[0].SigPubKey())
	s.Require().Equal(v1protocol.MessageID(&authorKey.PublicKey, wrappedPayload), decodedMessages[0].ID)
	s.Require().Equal(encodedPayload, decodedMessages[0].UnwrappedPayload)
	parsedMessage := decodedMessages[0].ParsedMessage.Interface().(protobuf.ChatMessage)
	s.Require().True(proto.Equal(&s.testMessage, &parsedMessage))
	s.Require().Equal(protobuf.ApplicationMetadataMessage_CHAT_MESSAGE, decodedMessages[0].Type)
}
