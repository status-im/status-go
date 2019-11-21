package protocol

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang/protobuf/proto"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethbridge "github.com/status-im/status-go/protocol/bridge/geth"
	"github.com/status-im/status-go/protocol/encryption"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/encryption/sharedsecret"
	"github.com/status-im/status-go/protocol/sqlite"
	transport "github.com/status-im/status-go/protocol/transport/whisper"
	whispertypes "github.com/status-im/status-go/protocol/transport/whisper/types"
	v1protocol "github.com/status-im/status-go/protocol/v1"
	whisper "github.com/status-im/whisper/whisperv6"
	datasyncproto "github.com/vacp2p/mvds/protobuf"
)

func TestMessageProcessorSuite(t *testing.T) {
	suite.Run(t, new(MessageProcessorSuite))
}

type MessageProcessorSuite struct {
	suite.Suite

	processor   *messageProcessor
	tmpDir      string
	testMessage v1protocol.Message
	logger      *zap.Logger
}

func (s *MessageProcessorSuite) SetupTest() {
	s.testMessage = v1protocol.Message{
		Text:      "abc123",
		ContentT:  "text/plain",
		MessageT:  "public-group-user-message",
		Clock:     154593077368201,
		Timestamp: 1545930773682,
		Content: v1protocol.Content{
			ChatID: "testing-adamb",
			Text:   "abc123",
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

func (s *MessageProcessorSuite) TestHandleDecodedMessagesSingle() {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	encodedPayload, err := v1protocol.EncodeMessage(s.testMessage)
	s.Require().NoError(err)

	message := &whispertypes.Message{}
	message.Sig = crypto.FromECDSAPub(&privateKey.PublicKey)
	message.Payload = encodedPayload

	decodedMessages, err := s.processor.handleMessages(message, true)
	s.Require().NoError(err)
	s.Require().Equal(1, len(decodedMessages))
	s.Require().Equal(encodedPayload, decodedMessages[0].DecryptedPayload)
	s.Require().Equal(&privateKey.PublicKey, decodedMessages[0].SigPubKey())
	s.Require().Equal(v1protocol.MessageID(&privateKey.PublicKey, encodedPayload), decodedMessages[0].ID)
	s.Require().Equal(s.testMessage, decodedMessages[0].ParsedMessage)
	s.Require().Equal(v1protocol.MessageT, decodedMessages[0].MessageType)
}

func (s *MessageProcessorSuite) TestHandleDecodedMessagesRaw() {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	encodedPayload, err := v1protocol.EncodeMessage(s.testMessage)
	s.Require().NoError(err)

	message := &whispertypes.Message{}
	message.Sig = crypto.FromECDSAPub(&privateKey.PublicKey)
	message.Payload = encodedPayload

	decodedMessages, err := s.processor.handleMessages(message, false)
	s.Require().NoError(err)
	s.Require().Equal(1, len(decodedMessages))
	s.Require().Equal(message, decodedMessages[0].TransportMessage)
	s.Require().Equal(encodedPayload, decodedMessages[0].DecryptedPayload)
	s.Require().Equal(&privateKey.PublicKey, decodedMessages[0].SigPubKey())
	s.Require().Equal(v1protocol.MessageID(&privateKey.PublicKey, encodedPayload), decodedMessages[0].ID)
	s.Require().Equal(nil, decodedMessages[0].ParsedMessage)
}

func (s *MessageProcessorSuite) TestHandleDecodedMessagesWrapped() {
	relayerKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	authorKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	encodedPayload, err := v1protocol.EncodeMessage(s.testMessage)
	s.Require().NoError(err)

	wrappedPayload, err := v1protocol.WrapMessageV1(encodedPayload, authorKey)
	s.Require().NoError(err)

	message := &whispertypes.Message{}
	message.Sig = crypto.FromECDSAPub(&relayerKey.PublicKey)
	message.Payload = wrappedPayload

	decodedMessages, err := s.processor.handleMessages(message, true)
	s.Require().NoError(err)

	s.Require().Equal(1, len(decodedMessages))
	s.Require().Equal(&authorKey.PublicKey, decodedMessages[0].SigPubKey())
	s.Require().Equal(v1protocol.MessageID(&authorKey.PublicKey, wrappedPayload), decodedMessages[0].ID)
	s.Require().Equal(encodedPayload, decodedMessages[0].DecryptedPayload)
	s.Require().Equal(s.testMessage, decodedMessages[0].ParsedMessage)
	s.Require().Equal(v1protocol.MessageT, decodedMessages[0].MessageType)
}

func (s *MessageProcessorSuite) TestHandleDecodedMessagesDatasync() {
	relayerKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	authorKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	encodedPayload, err := v1protocol.EncodeMessage(s.testMessage)
	s.Require().NoError(err)

	wrappedPayload, err := v1protocol.WrapMessageV1(encodedPayload, authorKey)
	s.Require().NoError(err)

	dataSyncMessage := datasyncproto.Payload{
		Messages: []*datasyncproto.Message{
			{Body: encodedPayload},
			{Body: wrappedPayload},
		},
	}
	marshalledDataSyncMessage, err := proto.Marshal(&dataSyncMessage)
	s.Require().NoError(err)
	message := &whispertypes.Message{}
	message.Sig = crypto.FromECDSAPub(&relayerKey.PublicKey)
	message.Payload = marshalledDataSyncMessage

	decodedMessages, err := s.processor.handleMessages(message, true)
	s.Require().NoError(err)

	// We send two messages, the unwrapped one will be attributed to the relayer, while the wrapped one will be attributed to the author
	s.Require().Equal(2, len(decodedMessages))
	s.Require().Equal(&relayerKey.PublicKey, decodedMessages[0].SigPubKey())
	s.Require().Equal(v1protocol.MessageID(&relayerKey.PublicKey, encodedPayload), decodedMessages[0].ID)
	s.Require().Equal(encodedPayload, decodedMessages[0].DecryptedPayload)
	s.Require().Equal(s.testMessage, decodedMessages[0].ParsedMessage)
	s.Require().Equal(v1protocol.MessageT, decodedMessages[0].MessageType)

	s.Require().Equal(&authorKey.PublicKey, decodedMessages[1].SigPubKey())
	s.Require().Equal(v1protocol.MessageID(&authorKey.PublicKey, wrappedPayload), decodedMessages[1].ID)
	s.Require().Equal(encodedPayload, decodedMessages[1].DecryptedPayload)
	s.Require().Equal(s.testMessage, decodedMessages[1].ParsedMessage)
	s.Require().Equal(v1protocol.MessageT, decodedMessages[1].MessageType)
}

func (s *MessageProcessorSuite) TestHandleDecodedMessagesDatasyncEncrypted() {
	relayerKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	authorKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	encodedPayload, err := v1protocol.EncodeMessage(s.testMessage)
	s.Require().NoError(err)

	wrappedPayload, err := v1protocol.WrapMessageV1(encodedPayload, authorKey)
	s.Require().NoError(err)

	dataSyncMessage := datasyncproto.Payload{
		Messages: []*datasyncproto.Message{
			{Body: encodedPayload},
			{Body: wrappedPayload},
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

	message := &whispertypes.Message{}
	message.Sig = crypto.FromECDSAPub(&relayerKey.PublicKey)
	message.Payload = encryptedPayload

	decodedMessages, err := s.processor.handleMessages(message, true)
	s.Require().NoError(err)

	// We send two messages, the unwrapped one will be attributed to the relayer,
	// while the wrapped one will be attributed to the author.
	s.Require().Equal(2, len(decodedMessages))
	s.Require().Equal(&relayerKey.PublicKey, decodedMessages[0].SigPubKey())
	s.Require().Equal(v1protocol.MessageID(&relayerKey.PublicKey, encodedPayload), decodedMessages[0].ID)
	s.Require().Equal(encodedPayload, decodedMessages[0].DecryptedPayload)
	s.Require().Equal(s.testMessage, decodedMessages[0].ParsedMessage)
	s.Require().Equal(v1protocol.MessageT, decodedMessages[0].MessageType)

	s.Require().Equal(&authorKey.PublicKey, decodedMessages[1].SigPubKey())
	s.Require().Equal(v1protocol.MessageID(&authorKey.PublicKey, wrappedPayload), decodedMessages[1].ID)
	s.Require().Equal(encodedPayload, decodedMessages[1].DecryptedPayload)
	s.Require().Equal(s.testMessage, decodedMessages[1].ParsedMessage)
	s.Require().Equal(v1protocol.MessageT, decodedMessages[1].MessageType)
}
