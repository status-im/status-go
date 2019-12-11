package protocol

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang/protobuf/proto"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/encryption"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/encryption/sharedsecret"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/sqlite"
	transport "github.com/status-im/status-go/protocol/transport/whisper"
	v1protocol "github.com/status-im/status-go/protocol/v1"
	"github.com/status-im/status-go/whisper/v6"
	datasyncproto "github.com/vacp2p/mvds/protobuf"
)

func TestMessageProcessorSuite(t *testing.T) {
	suite.Run(t, new(MessageProcessorSuite))
}

type MessageProcessorSuite struct {
	suite.Suite

	processor   *messageProcessor
	tmpDir      string
	testMessage Message
	logger      *zap.Logger
}

func (s *MessageProcessorSuite) SetupTest() {
	s.testMessage = Message{
		ChatMessage: protobuf.ChatMessage{
			Text:        "abc123",
			ChatId:      "testing-adamb",
			ContentType: protobuf.ChatMessage_TEXT_PLAIN,
			MessageType: protobuf.ChatMessage_PUBLIC_GROUP,
			Clock:       154593077368201,
			Timestamp:   1545930773682,
		},
	}

	var err error

	s.logger, err = zap.NewDevelopment()
	s.Require().NoError(err)

	s.tmpDir, err = ioutil.TempDir("", "")
	s.Require().NoError(err)

	identity, err := crypto.GenerateKey()
	s.Require().NoError(err)

	database, err := sqlite.Open(filepath.Join(s.tmpDir, "processor-test.sql"), "some-key")
	s.Require().NoError(err)

	onNewInstallations := func([]*multidevice.Installation) {}
	onNewSharedSecret := func([]*sharedsecret.Secret) {}
	onSendContactCode := func(*encryption.ProtocolMessageSpec) {}
	encryptionProtocol := encryption.New(
		database,
		"installation-1",
		onNewInstallations,
		onNewSharedSecret,
		onSendContactCode,
		s.logger,
	)

	whisperConfig := whisper.DefaultConfig
	whisperConfig.MinimumAcceptedPOW = 0
	shh := whisper.New(&whisperConfig)
	s.Require().NoError(shh.Start(nil))
	config := &config{}
	s.Require().NoError(WithDatasync()(config))

	whisperTransport, err := transport.NewWhisperServiceTransport(
		gethbridge.NewGethWhisperWrapper(shh),
		identity,
		database,
		nil,
		nil,
		s.logger,
	)
	s.Require().NoError(err)

	s.processor, err = newMessageProcessor(
		identity,
		database,
		encryptionProtocol,
		whisperTransport,
		nil,
		s.logger,
		featureFlags{},
	)
	s.Require().NoError(err)
}

func (s *MessageProcessorSuite) TearDownTest() {
	os.Remove(s.tmpDir)
	_ = s.logger.Sync()
}

func (s *MessageProcessorSuite) TestHandleDecodedMessagesWrapped() {
	relayerKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	authorKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	encodedPayload, err := proto.Marshal(&s.testMessage)
	s.Require().NoError(err)

	wrappedPayload, err := v1protocol.WrapMessageV1(encodedPayload, authorKey)
	s.Require().NoError(err)

	message := &types.Message{}
	message.Sig = crypto.FromECDSAPub(&relayerKey.PublicKey)
	message.Payload = wrappedPayload

	decodedMessages, err := s.processor.handleMessages(message, true)
	s.Require().NoError(err)

	s.Require().Equal(1, len(decodedMessages))
	s.Require().Equal(&authorKey.PublicKey, decodedMessages[0].SigPubKey())
	s.Require().Equal(v1protocol.MessageID(&authorKey.PublicKey, wrappedPayload), decodedMessages[0].ID)
	parsedMessage := decodedMessages[0].ParsedMessage.(protobuf.ChatMessage)
	s.Require().Equal(encodedPayload, decodedMessages[0].DecryptedPayload)
	s.Require().True(proto.Equal(&s.testMessage.ChatMessage, &parsedMessage))
	s.Require().Equal(v1protocol.MessageT, decodedMessages[0].MessageType)
}

func (s *MessageProcessorSuite) TestHandleDecodedMessagesDatasync() {
	relayerKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	authorKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	encodedPayload, err := proto.Marshal(&s.testMessage)
	s.Require().NoError(err)

	wrappedPayload, err := v1protocol.WrapMessageV1(encodedPayload, authorKey)
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

	decodedMessages, err := s.processor.handleMessages(message, true)
	s.Require().NoError(err)

	// We send two messages, the unwrapped one will be attributed to the relayer, while the wrapped one will be attributed to the author
	s.Require().Equal(1, len(decodedMessages))
	s.Require().Equal(&authorKey.PublicKey, decodedMessages[0].SigPubKey())
	s.Require().Equal(v1protocol.MessageID(&authorKey.PublicKey, wrappedPayload), decodedMessages[0].ID)
	s.Require().Equal(encodedPayload, decodedMessages[0].DecryptedPayload)
	parsedMessage := decodedMessages[0].ParsedMessage.(protobuf.ChatMessage)
	s.Require().True(proto.Equal(&s.testMessage.ChatMessage, &parsedMessage))
	s.Require().Equal(v1protocol.MessageT, decodedMessages[0].MessageType)
}

func (s *MessageProcessorSuite) TestHandleDecodedMessagesDatasyncEncrypted() {
	relayerKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	authorKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	encodedPayload, err := proto.Marshal(&s.testMessage)
	s.Require().NoError(err)

	wrappedPayload, err := v1protocol.WrapMessageV1(encodedPayload, authorKey)
	s.Require().NoError(err)

	dataSyncMessage := datasyncproto.Payload{
		Messages: []*datasyncproto.Message{
			&datasyncproto.Message{Body: wrappedPayload},
		},
	}
	marshalledDataSyncMessage, err := proto.Marshal(&dataSyncMessage)
	s.Require().NoError(err)

	// Create sender encryption protocol.
	senderDatabase, err := sqlite.Open(filepath.Join(s.tmpDir, "sender.db.sql"), "")
	s.Require().NoError(err)
	senderEncryptionProtocol := encryption.New(
		senderDatabase,
		"installation-2",
		func([]*multidevice.Installation) {},
		func([]*sharedsecret.Secret) {},
		func(*encryption.ProtocolMessageSpec) {},
		s.logger,
	)

	messageSpec, err := senderEncryptionProtocol.BuildDirectMessage(
		relayerKey,
		&s.processor.identity.PublicKey,
		marshalledDataSyncMessage,
	)
	s.Require().NoError(err)

	encryptedPayload, err := proto.Marshal(messageSpec.Message)
	s.Require().NoError(err)

	message := &types.Message{}
	message.Sig = crypto.FromECDSAPub(&relayerKey.PublicKey)
	message.Payload = encryptedPayload

	decodedMessages, err := s.processor.handleMessages(message, true)
	s.Require().NoError(err)

	// We send two messages, the unwrapped one will be attributed to the relayer,
	// while the wrapped one will be attributed to the author.
	s.Require().Equal(1, len(decodedMessages))
	s.Require().Equal(&authorKey.PublicKey, decodedMessages[0].SigPubKey())
	s.Require().Equal(v1protocol.MessageID(&authorKey.PublicKey, wrappedPayload), decodedMessages[0].ID)
	s.Require().Equal(encodedPayload, decodedMessages[0].DecryptedPayload)
	parsedMessage := decodedMessages[0].ParsedMessage.(protobuf.ChatMessage)
	s.Require().True(proto.Equal(&s.testMessage.ChatMessage, &parsedMessage))
	s.Require().Equal(v1protocol.MessageT, decodedMessages[0].MessageType)
}
