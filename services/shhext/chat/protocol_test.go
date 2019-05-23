package chat

import (
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/status-go/services/shhext/chat/topic"
	"github.com/stretchr/testify/suite"
)

func TestProtocolServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ProtocolServiceTestSuite))
}

type ProtocolServiceTestSuite struct {
	suite.Suite
	alice *ProtocolService
	bob   *ProtocolService
}

func (s *ProtocolServiceTestSuite) SetupTest() {
	aliceDBPath := "/tmp/alice.db"
	aliceDBKey := "alice"
	bobDBPath := "/tmp/bob.db"
	bobDBKey := "bob"

	os.Remove(aliceDBPath)
	os.Remove(bobDBPath)

	alicePersistence, err := NewSQLLitePersistence(aliceDBPath, aliceDBKey)
	if err != nil {
		panic(err)
	}

	bobPersistence, err := NewSQLLitePersistence(bobDBPath, bobDBKey)
	if err != nil {
		panic(err)
	}

	addedBundlesHandler := func(addedBundles []IdentityAndIDPair) {}
	onNewTopicHandler := func(topic []*topic.Secret) {}

	s.alice = NewProtocolService(
		NewEncryptionService(alicePersistence, DefaultEncryptionServiceConfig("1")),
		topic.NewService(alicePersistence.GetTopicStorage()),
		addedBundlesHandler,
		onNewTopicHandler,
	)

	s.bob = NewProtocolService(
		NewEncryptionService(bobPersistence, DefaultEncryptionServiceConfig("2")),
		topic.NewService(bobPersistence.GetTopicStorage()),
		addedBundlesHandler,
		onNewTopicHandler,
	)

}

func (s *ProtocolServiceTestSuite) TestBuildPublicMessage() {
	aliceKey, err := crypto.GenerateKey()
	s.NoError(err)

	payload := []byte("test")
	s.NoError(err)

	msg, err := s.alice.BuildPublicMessage(aliceKey, payload)
	s.NoError(err)
	s.NotNil(msg, "It creates a message")

	s.NotNilf(msg.GetBundles(), "It adds a bundle to the message")
}

func (s *ProtocolServiceTestSuite) TestBuildDirectMessage() {
	bobKey, err := crypto.GenerateKey()
	s.NoError(err)
	aliceKey, err := crypto.GenerateKey()
	s.NoError(err)

	payload := []byte("test")

	msg, _, err := s.alice.BuildDirectMessage(aliceKey, &bobKey.PublicKey, payload)
	s.NoError(err)
	s.NotNil(msg, "It creates a message")

	s.NotNilf(msg.GetBundle(), "It adds a bundle to the message")

	directMessage := msg.GetDirectMessage()
	s.NotNilf(directMessage, "It sets the direct message")

	encryptedPayload := directMessage["none"].GetPayload()
	s.NotNilf(encryptedPayload, "It sets the payload of the message")

	s.NotEqualf(payload, encryptedPayload, "It encrypts the payload")
}

func (s *ProtocolServiceTestSuite) TestBuildAndReadDirectMessage() {
	bobKey, err := crypto.GenerateKey()
	s.NoError(err)
	aliceKey, err := crypto.GenerateKey()
	s.NoError(err)

	payload := []byte("test")

	// Message is sent with DH
	marshaledMsg, _, err := s.alice.BuildDirectMessage(aliceKey, &bobKey.PublicKey, payload)

	s.NoError(err)

	// Bob is able to decrypt the message
	unmarshaledMsg, err := s.bob.HandleMessage(bobKey, &aliceKey.PublicKey, marshaledMsg, []byte("message-id"))
	s.NoError(err)

	s.NotNil(unmarshaledMsg)

	recoveredPayload := []byte("test")
	s.Equalf(payload, recoveredPayload, "It successfully unmarshal the decrypted message")
}
