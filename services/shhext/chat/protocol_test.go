package chat

import (
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang/protobuf/proto"
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

	s.alice = NewProtocolService(NewEncryptionService(alicePersistence, DefaultEncryptionServiceConfig("1")), addedBundlesHandler)
	s.bob = NewProtocolService(NewEncryptionService(bobPersistence, DefaultEncryptionServiceConfig("2")), addedBundlesHandler)
}

func (s *ProtocolServiceTestSuite) TestBuildDirectMessage() {
	bobKey, err := crypto.GenerateKey()
	s.NoError(err)
	aliceKey, err := crypto.GenerateKey()
	s.NoError(err)

	payload, err := proto.Marshal(&ChatMessagePayload{
		Content:     "Test content",
		ClockValue:  1,
		ContentType: "a",
		MessageType: "some type",
	})
	s.NoError(err)

	marshaledMsg, err := s.alice.BuildDirectMessage(aliceKey, payload, &bobKey.PublicKey, &aliceKey.PublicKey)

	s.NoError(err)
	s.NotNil(marshaledMsg, "It creates a message")
	s.NotNil(marshaledMsg[&aliceKey.PublicKey], "It creates a single message")

	unmarshaledMsg := &ProtocolMessage{}
	err = proto.Unmarshal(marshaledMsg[&bobKey.PublicKey], unmarshaledMsg)

	s.NoError(err)

	s.NotNilf(unmarshaledMsg.GetBundle(), "It adds a bundle to the message")

	directMessage := unmarshaledMsg.GetDirectMessage()

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

	payload := ChatMessagePayload{
		Content:     "Test content",
		ClockValue:  1,
		ContentType: "a",
		MessageType: "some type",
	}

	marshaledPayload, err := proto.Marshal(&payload)
	s.NoError(err)

	// Message is sent with DH
	marshaledMsg, err := s.alice.BuildDirectMessage(aliceKey, marshaledPayload, &bobKey.PublicKey)

	s.NoError(err)

	// Bob is able to decrypt the message
	unmarshaledMsg, err := s.bob.HandleMessage(bobKey, &aliceKey.PublicKey, marshaledMsg[&bobKey.PublicKey])
	s.NoError(err)

	s.NotNil(unmarshaledMsg)

	recoveredPayload := ChatMessagePayload{}
	err = proto.Unmarshal(unmarshaledMsg, &recoveredPayload)

	s.NoError(err)
	s.Equalf(proto.Equal(&payload, &recoveredPayload), true, "It successfully unmarshal the decrypted message")
}
