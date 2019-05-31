package chat

import (
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/status-go/services/shhext/chat/multidevice"
	"github.com/status-im/status-go/services/shhext/chat/sharedsecret"
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

	addedBundlesHandler := func(addedBundles []multidevice.IdentityAndIDPair) {}
	onNewSharedSecretHandler := func(secret []*sharedsecret.Secret) {}

	aliceMultideviceConfig := &multidevice.Config{
		MaxInstallations: 3,
		InstallationID:   "1",
	}

	s.alice = NewProtocolService(
		NewEncryptionService(alicePersistence, DefaultEncryptionServiceConfig("1")),
		sharedsecret.NewService(alicePersistence.GetSharedSecretStorage()),
		multidevice.New(aliceMultideviceConfig, alicePersistence.GetMultideviceStorage()),
		addedBundlesHandler,
		onNewSharedSecretHandler,
	)

	bobMultideviceConfig := &multidevice.Config{
		MaxInstallations: 3,
		InstallationID:   "2",
	}

	s.bob = NewProtocolService(
		NewEncryptionService(bobPersistence, DefaultEncryptionServiceConfig("2")),
		sharedsecret.NewService(bobPersistence.GetSharedSecretStorage()),
		multidevice.New(bobMultideviceConfig, bobPersistence.GetMultideviceStorage()),
		addedBundlesHandler,
		onNewSharedSecretHandler,
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

	msgSpec, err := s.alice.BuildDirectMessage(aliceKey, &bobKey.PublicKey, payload)
	s.NoError(err)
	s.NotNil(msgSpec, "It creates a message spec")

	msg := msgSpec.Message
	s.NotNil(msg, "It creates a messages")

	s.NotNilf(msg.GetBundle(), "It adds a bundle to the message")

	directMessage := msg.GetDirectMessage()
	s.NotNilf(directMessage, "It sets the direct message")

	encryptedPayload := directMessage["none"].GetPayload()
	s.NotNilf(encryptedPayload, "It sets the payload of the message")

	s.NotEqualf(payload, encryptedPayload, "It encrypts the payload")
}

func (s *ProtocolServiceTestSuite) TestBuildAndReadDirectMessage() {
	bobKey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	aliceKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	payload := []byte("test")

	// Message is sent with DH
	msgSpec, err := s.alice.BuildDirectMessage(aliceKey, &bobKey.PublicKey, payload)
	s.Require().NoError(err)
	s.Require().NotNil(msgSpec)

	msg := msgSpec.Message
	s.Require().NotNil(msg)

	// Bob is able to decrypt the message
	unmarshaledMsg, err := s.bob.HandleMessage(bobKey, &aliceKey.PublicKey, msg, []byte("message-id"))
	s.NoError(err)
	s.NotNil(unmarshaledMsg)

	recoveredPayload := []byte("test")
	s.Equalf(payload, recoveredPayload, "It successfully unmarshal the decrypted message")
}
