package encryption

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/t/helpers"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/crypto"
)

var cleartext = []byte("hello")
var aliceInstallationID = "1"
var bobInstallationID = "2"
var defaultMessageID = []byte("default")

func TestEncryptionServiceTestSuite(t *testing.T) {
	suite.Run(t, new(EncryptionServiceTestSuite))
}

type EncryptionServiceTestSuite struct {
	suite.Suite
	logger *zap.Logger
	alice  *Protocol
	bob    *Protocol
}

func (s *EncryptionServiceTestSuite) initDatabases(config encryptorConfig) {
	var err error

	db, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	s.Require().NoError(err)
	err = sqlite.Migrate(db)
	s.Require().NoError(err)
	config.InstallationID = aliceInstallationID
	s.alice = NewWithEncryptorConfig(
		db,
		aliceInstallationID,
		config,
		s.logger.With(zap.String("user", "alice")),
	)

	db, err = helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	s.Require().NoError(err)
	err = sqlite.Migrate(db)
	s.Require().NoError(err)
	config.InstallationID = bobInstallationID
	s.bob = NewWithEncryptorConfig(
		db,
		bobInstallationID,
		config,
		s.logger.With(zap.String("user", "bob")),
	)
}

func (s *EncryptionServiceTestSuite) SetupTest() {
	s.logger, _ = zap.NewProduction()
	s.initDatabases(defaultEncryptorConfig("none", s.logger))
}

func (s *EncryptionServiceTestSuite) TearDownTest() {
	_ = s.logger.Sync()
}

func (s *EncryptionServiceTestSuite) TestGetBundle() {
	aliceKey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	aliceBundle1, err := s.alice.GetBundle(aliceKey)
	s.Require().NoError(err)
	s.NotNil(aliceBundle1, "It creates a bundle")

	aliceBundle2, err := s.alice.GetBundle(aliceKey)
	s.Require().NoError(err)
	s.Equal(aliceBundle1.GetSignedPreKeys(), aliceBundle2.GetSignedPreKeys(), "It returns the same signed pre keys")
	s.NotEqual(aliceBundle1.Timestamp, aliceBundle2.Timestamp, "It refreshes the timestamp")
}

func (s *EncryptionServiceTestSuite) TestHashRatchetSend() {
	aliceKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	bobKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	groupID := []byte("test_community_id")
	s.Require().NotNil(aliceKey)
	s.Require().NotNil(bobKey)

	s.logger.Info("Hash ratchet key exchange 1")
	keyID1, err := s.alice.encryptor.GenerateHashRatchetKey(groupID)
	s.Require().NoError(err)

	hashRatchetKeyExMsg1, err := s.alice.BuildHashRatchetKeyExchangeMessage(aliceKey, &bobKey.PublicKey, groupID, []*HashRatchetKeyCompatibility{keyID1})
	s.Require().NoError(err)

	s.logger.Info("Hash ratchet key exchange 1", zap.Any("msg", hashRatchetKeyExMsg1.Message))
	s.Require().NotNil(hashRatchetKeyExMsg1)

	s.logger.Info("Handle hash ratchet key msg 1")
	decryptedResponse1, err := s.bob.HandleMessage(bobKey, &aliceKey.PublicKey, hashRatchetKeyExMsg1.Message, defaultMessageID)
	s.Require().NoError(err)
	s.Require().NotNil(decryptedResponse1)

	decryptedHashRatchetKeyBytes1 := decryptedResponse1.DecryptedMessage
	decryptedHashRatchetKeyID1, err := s.bob.encryptor.persistence.GetCurrentKeyForGroup(groupID)
	s.logger.Info("Current hash ratchet key in DB 1", zap.Any("keyId", decryptedHashRatchetKeyID1))
	s.Require().NoError(err)
	s.Require().NotNil(decryptedHashRatchetKeyID1)
	s.Require().NotNil(decryptedHashRatchetKeyID1.Key)

	keyID, err := decryptedHashRatchetKeyID1.GetKeyID()
	s.Require().NoError(err)
	s.Require().NotNil(keyID)

	s.Require().NotNil(decryptedHashRatchetKeyID1.GroupID)
	s.Require().NotEmpty(decryptedHashRatchetKeyID1.Timestamp)
	s.Require().NotNil(decryptedHashRatchetKeyBytes1)
	//s.Equal(decryptedHashRatchetKey1, decryptedHashRatchetKeyBytes1)

	payload1 := []byte("community msg 1")
	hashRatchetMsg1, err := s.bob.BuildHashRatchetMessage(groupID, payload1)

	s.Require().NoError(err)
	s.Require().NotNil(hashRatchetMsg1)
	s.Require().NotNil(hashRatchetMsg1.Message)

	decryptedResponse2, err := s.alice.HandleMessage(aliceKey, nil, hashRatchetMsg1.Message, defaultMessageID)

	s.Require().NoError(err)
	s.Require().NotNil(decryptedResponse2)
	s.Equal(payload1, decryptedResponse2.DecryptedMessage)

	payload2 := []byte("community msg 2")
	hashRatchetMsg2, err := s.alice.BuildHashRatchetMessage(groupID, payload2)

	s.Require().NoError(err)
	s.Require().NotNil(hashRatchetMsg2)
	s.Require().NotNil(hashRatchetMsg2.Message)

	decryptedResponse3, err := s.bob.HandleMessage(bobKey, nil, hashRatchetMsg2.Message, defaultMessageID)

	s.Require().NoError(err)
	s.Require().NotNil(decryptedResponse3)
	s.Equal(payload2, decryptedResponse3.DecryptedMessage)

	// Re-generate hash ratchet key. Bob generates a new key and sends it to Alice

	keyID2, err := s.bob.encryptor.GenerateHashRatchetKey(groupID)
	s.Require().NoError(err)

	hashRatchetKeyExMsg2, err := s.bob.BuildHashRatchetKeyExchangeMessage(bobKey, &aliceKey.PublicKey, groupID, []*HashRatchetKeyCompatibility{keyID2})
	s.Require().NoError(err)

	s.logger.Info("Hash ratchet key exchange 2", zap.Any("msg", hashRatchetKeyExMsg2.Message))
	s.Require().NotNil(hashRatchetKeyExMsg2)

	s.logger.Info("Handle hash ratchet key msg 2")
	decryptedResponse4, err := s.alice.HandleMessage(aliceKey, &bobKey.PublicKey, hashRatchetKeyExMsg2.Message, defaultMessageID)
	s.Require().NoError(err)
	decryptedHashRatchetKeyBytes2 := decryptedResponse4.DecryptedMessage
	decryptedHashRatchetKeyID2, err := s.alice.encryptor.persistence.GetCurrentKeyForGroup(groupID)
	s.Require().NoError(err)
	s.logger.Info("Current hash ratchet key in DB 2", zap.Any("keyId", decryptedHashRatchetKeyID2))
	s.Require().NotNil(decryptedHashRatchetKeyID2)
	s.Require().NotNil(decryptedHashRatchetKeyBytes2)

	payload3 := []byte("community msg 3")
	hashRatchetMsg3, err := s.alice.BuildHashRatchetMessage(groupID, payload3)

	s.logger.Info("BuildHashRatchetMessage err", zap.Any("err", err))
	s.Require().NotNil(hashRatchetMsg3)
	s.Require().NotNil(hashRatchetMsg3.Message)

	//directMsg1 := hashRatchetMsg.Message.GetEncryptedMessage()

	decryptedResponse5, err := s.bob.HandleMessage(bobKey, nil, hashRatchetMsg3.Message, defaultMessageID)

	s.logger.Info("HandleHashRatchetMessage err", zap.Any("err", err))
	s.Require().NotNil(decryptedResponse5)
	s.Equal(payload3, decryptedResponse5.DecryptedMessage)

	payload4 := []byte("community msg 4")
	payload5 := []byte("community msg 5")
	payload6 := []byte("community msg 6")
	hashRatchetMsg4, err := s.alice.BuildHashRatchetMessage(groupID, payload4) // seqNo=2
	s.Require().NoError(err)
	hashRatchetMsg5, err := s.alice.BuildHashRatchetMessage(groupID, payload5) // seqNo=3
	s.Require().NoError(err)
	hashRatchetMsg6, err := s.alice.BuildHashRatchetMessage(groupID, payload6) // seqNo=3
	s.Require().NoError(err)

	// Handle them out of order plus an older one we've received earlier with seqNo=1

	decryptedResponse6, err := s.bob.HandleMessage(bobKey, nil, hashRatchetMsg6.Message, defaultMessageID)
	s.Require().NoError(err)
	decryptedResponse7, err := s.bob.HandleMessage(bobKey, nil, hashRatchetMsg5.Message, defaultMessageID)
	s.Require().NoError(err)
	decryptedResponse8, err := s.bob.HandleMessage(bobKey, nil, hashRatchetMsg4.Message, defaultMessageID)
	s.Require().NoError(err)
	decryptedResponse9, err := s.bob.HandleMessage(bobKey, nil, hashRatchetMsg3.Message, defaultMessageID)
	s.Require().NoError(err)

	s.logger.Info("HandleHashRatchetMessage err", zap.Any("err", err))
	s.Require().NotNil(decryptedResponse6)
	s.Equal(payload6, decryptedResponse6.DecryptedMessage)
	s.Require().NotNil(decryptedResponse7)
	s.Equal(payload5, decryptedResponse7.DecryptedMessage)
	s.Require().NotNil(decryptedResponse8)
	s.Equal(payload4, decryptedResponse8.DecryptedMessage)
	s.Require().NotNil(decryptedResponse9)
	s.Equal(payload3, decryptedResponse9.DecryptedMessage)

	// Handle message with previous key
	decryptedResponse10, err := s.bob.HandleMessage(bobKey, nil, hashRatchetMsg2.Message, defaultMessageID)
	s.Require().NoError(err)
	s.Require().NotNil(decryptedResponse10)
	s.Equal(payload2, decryptedResponse10.DecryptedMessage)
}

// Alice sends Bob an encrypted message with DH using an ephemeral key
// and Bob's identity key.
// Bob is able to decrypt it.
// Alice does not re-use the symmetric key
func (s *EncryptionServiceTestSuite) TestEncryptPayloadNoBundle() {
	bobKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	aliceKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	response1, err := s.alice.BuildEncryptedMessage(aliceKey, &bobKey.PublicKey, cleartext)
	s.Require().NoError(err)

	encryptionResponse1 := response1.Message.GetEncryptedMessage()

	installationResponse1 := encryptionResponse1["none"]
	// That's for any device
	s.Require().NotNil(installationResponse1)

	cyphertext1 := installationResponse1.Payload
	ephemeralKey1 := installationResponse1.GetDHHeader().GetKey()
	s.NotNil(ephemeralKey1, "It generates an ephemeral key for DH exchange")
	s.NotNil(cyphertext1, "It generates an encrypted payload")
	s.NotEqual(cyphertext1, cleartext, "It encrypts the payload correctly")

	// On the receiver side, we should be able to decrypt using our private key and the ephemeral just sent
	decryptedPayload1, err := s.bob.HandleMessage(bobKey, &aliceKey.PublicKey, response1.Message, defaultMessageID)
	s.Require().NoError(err)
	s.Equal(cleartext, decryptedPayload1.DecryptedMessage, "It correctly decrypts the payload using DH")

	// The next message will not be re-using the same key
	response2, err := s.alice.BuildEncryptedMessage(aliceKey, &bobKey.PublicKey, cleartext)
	s.Require().NoError(err)

	encryptionResponse2 := response2.Message.GetEncryptedMessage()

	installationResponse2 := encryptionResponse2[aliceInstallationID]

	cyphertext2 := installationResponse2.GetPayload()
	ephemeralKey2 := installationResponse2.GetDHHeader().GetKey()
	s.NotEqual(cyphertext1, cyphertext2, "It does not re-use the symmetric key")
	s.NotEqual(ephemeralKey1, ephemeralKey2, "It does not re-use the ephemeral key")

	decryptedPayload2, err := s.bob.HandleMessage(bobKey, &aliceKey.PublicKey, response2.Message, defaultMessageID)
	s.Require().NoError(err)
	s.Equal(cleartext, decryptedPayload2.DecryptedMessage, "It correctly decrypts the payload using DH")
}

// Alice has Bob's bundle
// Alice sends Bob an encrypted message with X3DH and DR using an ephemeral key
// and Bob's bundle.
func (s *EncryptionServiceTestSuite) TestEncryptPayloadBundle() {
	bobKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	aliceKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	// Create a bundle
	bobBundle, err := s.bob.GetBundle(bobKey)
	s.Require().NoError(err)

	// We add bob bundle
	_, err = s.alice.ProcessPublicBundle(aliceKey, bobBundle)
	s.Require().NoError(err)

	// We send a message using the bundle
	response1, err := s.alice.BuildEncryptedMessage(aliceKey, &bobKey.PublicKey, cleartext)
	s.Require().NoError(err)

	encryptionResponse1 := response1.Message.GetEncryptedMessage()

	installationResponse1 := encryptionResponse1[bobInstallationID]
	s.Require().NotNil(installationResponse1)

	cyphertext1 := installationResponse1.GetPayload()
	x3dhHeader := installationResponse1.GetX3DHHeader()
	drHeader := installationResponse1.GetDRHeader()

	s.NotNil(cyphertext1, "It generates an encrypted payload")
	s.NotEqual(cyphertext1, cleartext, "It encrypts the payload correctly")

	// Check X3DH Header
	bundleID := bobBundle.GetSignedPreKeys()[bobInstallationID].GetSignedPreKey()

	s.NotNil(x3dhHeader, "It adds an x3dh header")
	s.NotNil(x3dhHeader.GetKey(), "It adds an ephemeral key")
	s.Equal(x3dhHeader.GetId(), bundleID, "It sets the bundle id")

	// Check DR Header
	s.NotNil(drHeader, "It adds a DR header")
	s.NotNil(drHeader.GetKey(), "It adds a key to the DR header")
	s.Equal(bundleID, drHeader.GetId(), "It adds the bundle id")
	s.Equal(uint32(0), drHeader.GetN(), "It adds the correct message number")
	s.Equal(uint32(0), drHeader.GetPn(), "It adds the correct length of the message chain")

	// Bob is able to decrypt it using the bundle
	decryptedPayload1, err := s.bob.HandleMessage(bobKey, &aliceKey.PublicKey, response1.Message, defaultMessageID)
	s.Require().NoError(err)
	s.Equal(cleartext, decryptedPayload1.DecryptedMessage, "It correctly decrypts the payload using X3DH")
}

// Alice has Bob's bundle
// Alice sends Bob 2 encrypted messages with X3DH and DR using an ephemeral key
// and Bob's bundle.
// Alice sends another message. This message should be using a DR
// and should include the initial x3dh message
// Bob receives only the last one, he should be able to decrypt it
// nolint: megacheck
func (s *EncryptionServiceTestSuite) TestConsequentMessagesBundle() {
	cleartext1 := []byte("message 1")
	cleartext2 := []byte("message 2")

	bobKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	aliceKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	// Create a bundle
	bobBundle, err := s.bob.GetBundle(bobKey)
	s.Require().NoError(err)

	// We add bob bundle
	_, err = s.alice.ProcessPublicBundle(aliceKey, bobBundle)
	s.Require().NoError(err)

	// We send a message using the bundle
	_, err = s.alice.BuildEncryptedMessage(aliceKey, &bobKey.PublicKey, cleartext1)
	s.Require().NoError(err)

	// We send another message using the bundle
	response, err := s.alice.BuildEncryptedMessage(aliceKey, &bobKey.PublicKey, cleartext2)
	s.Require().NoError(err)
	encryptionResponse := response.Message.GetEncryptedMessage()

	installationResponse := encryptionResponse[bobInstallationID]
	s.Require().NotNil(installationResponse)

	cyphertext1 := installationResponse.GetPayload()
	x3dhHeader := installationResponse.GetX3DHHeader()
	drHeader := installationResponse.GetDRHeader()

	s.NotNil(cyphertext1, "It generates an encrypted payload")
	s.NotEqual(cyphertext1, cleartext2, "It encrypts the payload correctly")

	// Check X3DH Header
	bundleID := bobBundle.GetSignedPreKeys()[bobInstallationID].GetSignedPreKey()

	s.NotNil(x3dhHeader, "It adds an x3dh header")
	s.NotNil(x3dhHeader.GetKey(), "It adds an ephemeral key")
	s.Equal(x3dhHeader.GetId(), bundleID, "It sets the bundle id")

	// Check DR Header
	s.NotNil(drHeader, "It adds a DR header")
	s.NotNil(drHeader.GetKey(), "It adds a key to the DR header")
	s.Equal(bundleID, drHeader.GetId(), "It adds the bundle id")

	s.Equal(uint32(1), drHeader.GetN(), "It adds the correct message number")
	s.Equal(uint32(0), drHeader.GetPn(), "It adds the correct length of the message chain")

	// Bob is able to decrypt it using the bundle
	decryptedPayload1, err := s.bob.HandleMessage(bobKey, &aliceKey.PublicKey, response.Message, defaultMessageID)
	s.Require().NoError(err)

	s.Equal(cleartext2, decryptedPayload1.DecryptedMessage, "It correctly decrypts the payload using X3DH")
}

// Alice has Bob's bundle
// Alice sends Bob an encrypted message with X3DH using an ephemeral key
// and Bob's bundle.
// Bob's receives the message
// Bob replies to the message
// Alice replies to the message

func (s *EncryptionServiceTestSuite) TestConversation() {
	cleartext1 := []byte("message 1")
	cleartext2 := []byte("message 2")

	bobKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	aliceKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	// Create a bundle
	bobBundle, err := s.bob.GetBundle(bobKey)
	s.Require().NoError(err)

	// Create a bundle
	aliceBundle, err := s.alice.GetBundle(aliceKey)
	s.Require().NoError(err)

	// We add bob bundle
	_, err = s.alice.ProcessPublicBundle(aliceKey, bobBundle)
	s.Require().NoError(err)

	// We add alice bundle
	_, err = s.bob.ProcessPublicBundle(bobKey, aliceBundle)
	s.Require().NoError(err)

	// Alice sends a message
	response, err := s.alice.BuildEncryptedMessage(aliceKey, &bobKey.PublicKey, cleartext1)
	s.Require().NoError(err)

	// Bob receives the message
	_, err = s.bob.HandleMessage(bobKey, &aliceKey.PublicKey, response.Message, defaultMessageID)
	s.Require().NoError(err)

	// Bob replies to the message
	response, err = s.bob.BuildEncryptedMessage(bobKey, &aliceKey.PublicKey, cleartext1)
	s.Require().NoError(err)

	// Alice receives the message
	_, err = s.alice.HandleMessage(aliceKey, &bobKey.PublicKey, response.Message, defaultMessageID)
	s.Require().NoError(err)

	// We send another message using the bundle
	response, err = s.alice.BuildEncryptedMessage(aliceKey, &bobKey.PublicKey, cleartext2)
	s.Require().NoError(err)
	encryptionResponse := response.Message.GetEncryptedMessage()

	installationResponse := encryptionResponse[bobInstallationID]
	s.Require().NotNil(installationResponse)

	cyphertext1 := installationResponse.GetPayload()
	x3dhHeader := installationResponse.GetX3DHHeader()
	drHeader := installationResponse.GetDRHeader()

	s.NotNil(cyphertext1, "It generates an encrypted payload")
	s.NotEqual(cyphertext1, cleartext2, "It encrypts the payload correctly")

	// It does not send the x3dh bundle
	s.Nil(x3dhHeader, "It does not add an x3dh header")

	// Check DR Header
	bundleID := bobBundle.GetSignedPreKeys()[bobInstallationID].GetSignedPreKey()

	s.NotNil(drHeader, "It adds a DR header")
	s.NotNil(drHeader.GetKey(), "It adds a key to the DR header")
	s.Equal(bundleID, drHeader.GetId(), "It adds the bundle id")

	s.Equal(uint32(0), drHeader.GetN(), "It adds the correct message number")
	s.Equal(uint32(1), drHeader.GetPn(), "It adds the correct length of the message chain")

	// Bob is able to decrypt it using the bundle
	decryptedPayload1, err := s.bob.HandleMessage(bobKey, &aliceKey.PublicKey, response.Message, defaultMessageID)
	s.Require().NoError(err)

	s.Equal(cleartext2, decryptedPayload1.DecryptedMessage, "It correctly decrypts the payload using X3DH")
}

// Previous implementation allowed max maxSkip keys in the same receiving chain
// leading to a problem whereby dropped messages would accumulate and eventually
// we would not be able to decrypt any new message anymore.
// Here we are testing that maxSkip only applies to *consecutive* messages, not
// overall.
func (s *EncryptionServiceTestSuite) TestMaxSkipKeys() {
	bobText := []byte("text")

	bobKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	aliceKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	// Create a bundle
	bobBundle, err := s.bob.GetBundle(bobKey)
	s.Require().NoError(err)

	// We add bob bundle
	_, err = s.alice.ProcessPublicBundle(aliceKey, bobBundle)
	s.Require().NoError(err)

	// Create a bundle
	aliceBundle, err := s.alice.GetBundle(aliceKey)
	s.Require().NoError(err)

	// We add alice bundle
	_, err = s.bob.ProcessPublicBundle(bobKey, aliceBundle)
	s.Require().NoError(err)

	// Bob sends a message

	for i := 0; i < s.alice.encryptor.config.MaxSkip; i++ {
		_, err = s.bob.BuildEncryptedMessage(bobKey, &aliceKey.PublicKey, bobText)
		s.Require().NoError(err)
	}

	// Bob sends a message
	bobMessage1, err := s.bob.BuildEncryptedMessage(bobKey, &aliceKey.PublicKey, bobText)
	s.Require().NoError(err)

	// Alice receives the message
	_, err = s.alice.HandleMessage(aliceKey, &bobKey.PublicKey, bobMessage1.Message, defaultMessageID)
	s.Require().NoError(err)

	// Bob sends a message
	_, err = s.bob.BuildEncryptedMessage(bobKey, &aliceKey.PublicKey, bobText)
	s.Require().NoError(err)

	// Bob sends a message
	bobMessage2, err := s.bob.BuildEncryptedMessage(bobKey, &aliceKey.PublicKey, bobText)
	s.Require().NoError(err)

	// Alice receives the message, we should have maxSkip + 1 keys in the db, but
	// we should not throw an error
	_, err = s.alice.HandleMessage(aliceKey, &bobKey.PublicKey, bobMessage2.Message, defaultMessageID)
	s.Require().NoError(err)
}

// Test that an error is thrown if max skip is reached
func (s *EncryptionServiceTestSuite) TestMaxSkipKeysError() {
	bobText := []byte("text")

	bobKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	aliceKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	// Create a bundle
	bobBundle, err := s.bob.GetBundle(bobKey)
	s.Require().NoError(err)

	// We add bob bundle
	_, err = s.alice.ProcessPublicBundle(aliceKey, bobBundle)
	s.Require().NoError(err)

	// Create a bundle
	aliceBundle, err := s.alice.GetBundle(aliceKey)
	s.Require().NoError(err)

	// We add alice bundle
	_, err = s.bob.ProcessPublicBundle(bobKey, aliceBundle)
	s.Require().NoError(err)

	// Bob sends a message

	for i := 0; i < s.alice.encryptor.config.MaxSkip+1; i++ {
		_, err = s.bob.BuildEncryptedMessage(bobKey, &aliceKey.PublicKey, bobText)
		s.Require().NoError(err)
	}

	// Bob sends a message
	bobMessage1, err := s.bob.BuildEncryptedMessage(bobKey, &aliceKey.PublicKey, bobText)
	s.Require().NoError(err)

	// Alice receives the message
	_, err = s.alice.HandleMessage(aliceKey, &bobKey.PublicKey, bobMessage1.Message, defaultMessageID)
	s.Require().Equal(errors.New("can't skip current chain message keys: too many messages"), err)
}

func (s *EncryptionServiceTestSuite) TestMaxMessageKeysPerSession() {
	config := defaultEncryptorConfig("none", zap.NewNop())
	// Set MaxKeep and MaxSkip to an high value so it does not interfere
	config.MaxKeep = 100000
	config.MaxSkip = 100000

	s.initDatabases(config)

	bobText := []byte("text")

	bobKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	aliceKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	// Create a bundle
	bobBundle, err := s.bob.GetBundle(bobKey)
	s.Require().NoError(err)

	// We add bob bundle
	_, err = s.alice.ProcessPublicBundle(aliceKey, bobBundle)
	s.Require().NoError(err)

	// Create a bundle
	aliceBundle, err := s.alice.GetBundle(aliceKey)
	s.Require().NoError(err)

	// We add alice bundle
	_, err = s.bob.ProcessPublicBundle(bobKey, aliceBundle)
	s.Require().NoError(err)

	// We create just enough messages so that the first key should be deleted

	nMessages := s.alice.encryptor.config.MaxMessageKeysPerSession
	messages := make([]*ProtocolMessage, nMessages)
	for i := 0; i < nMessages; i++ {
		m, err := s.bob.BuildEncryptedMessage(bobKey, &aliceKey.PublicKey, bobText)
		s.Require().NoError(err)

		messages[i] = m.Message
	}

	// Another message to trigger the deletion
	m, err := s.bob.BuildEncryptedMessage(bobKey, &aliceKey.PublicKey, bobText)
	s.Require().NoError(err)
	_, err = s.alice.HandleMessage(aliceKey, &bobKey.PublicKey, m.Message, defaultMessageID)
	s.Require().NoError(err)

	// We decrypt the first message, and it should fail
	_, err = s.alice.HandleMessage(aliceKey, &bobKey.PublicKey, messages[0], defaultMessageID)
	s.Require().Equal(errors.New("can't skip current chain message keys: bad until: probably an out-of-order message that was deleted"), err)

	// We decrypt the second message, and it should be decrypted
	_, err = s.alice.HandleMessage(aliceKey, &bobKey.PublicKey, messages[1], defaultMessageID)
	s.Require().NoError(err)
}

func (s *EncryptionServiceTestSuite) TestMaxKeep() {
	config := defaultEncryptorConfig("none", s.logger)
	// Set MaxMessageKeysPerSession to an high value so it does not interfere
	config.MaxMessageKeysPerSession = 100000
	// Set MaxKeep to a small number for testing
	config.MaxKeep = 10

	s.initDatabases(config)

	bobText := []byte("text")

	bobKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	aliceKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	// Create a bundle
	bobBundle, err := s.bob.GetBundle(bobKey)
	s.Require().NoError(err)

	// We add bob bundle
	_, err = s.alice.ProcessPublicBundle(aliceKey, bobBundle)
	s.Require().NoError(err)

	// Create a bundle
	aliceBundle, err := s.alice.GetBundle(aliceKey)
	s.Require().NoError(err)

	// We add alice bundle
	_, err = s.bob.ProcessPublicBundle(bobKey, aliceBundle)
	s.Require().NoError(err)

	// We decrypt all messages but 1 & 2
	messages := make([]*ProtocolMessage, s.alice.encryptor.config.MaxKeep)
	for i := 0; i < s.alice.encryptor.config.MaxKeep; i++ {
		m, err := s.bob.BuildEncryptedMessage(bobKey, &aliceKey.PublicKey, bobText)
		messages[i] = m.Message
		s.Require().NoError(err)

		if i != 0 && i != 1 {
			messageID := []byte(fmt.Sprintf("%d", i))
			_, err = s.alice.HandleMessage(aliceKey, &bobKey.PublicKey, m.Message, messageID)
			s.Require().NoError(err)
			err = s.alice.ConfirmMessageProcessed(messageID)
			s.Require().NoError(err)
		}

	}

	// We decrypt the first message, and it should fail, as it should have been removed
	_, err = s.alice.HandleMessage(aliceKey, &bobKey.PublicKey, messages[0], defaultMessageID)
	s.Require().Equal(errors.New("can't skip current chain message keys: bad until: probably an out-of-order message that was deleted"), err)

	// We decrypt the second message, and it should be decrypted
	_, err = s.alice.HandleMessage(aliceKey, &bobKey.PublicKey, messages[1], defaultMessageID)
	s.Require().NoError(err)
}

// Alice has Bob's bundle
// Bob has Alice's bundle
// Bob sends a message to alice
// Alice sends a message to Bob
// Bob receives alice message
// Alice receives Bob message
// Bob sends another message to alice and vice-versa.
func (s *EncryptionServiceTestSuite) TestConcurrentBundles() {
	bobText1 := []byte("bob text 1")
	bobText2 := []byte("bob text 2")
	aliceText1 := []byte("alice text 1")
	aliceText2 := []byte("alice text 2")

	bobKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	aliceKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	// Create a bundle
	bobBundle, err := s.bob.GetBundle(bobKey)
	s.Require().NoError(err)

	// We add bob bundle
	_, err = s.alice.ProcessPublicBundle(aliceKey, bobBundle)
	s.Require().NoError(err)

	// Create a bundle
	aliceBundle, err := s.alice.GetBundle(aliceKey)
	s.Require().NoError(err)

	// We add alice bundle
	_, err = s.bob.ProcessPublicBundle(bobKey, aliceBundle)
	s.Require().NoError(err)

	// Alice sends a message
	aliceMessage1, err := s.alice.BuildEncryptedMessage(aliceKey, &bobKey.PublicKey, aliceText1)
	s.Require().NoError(err)

	// Bob sends a message
	bobMessage1, err := s.bob.BuildEncryptedMessage(bobKey, &aliceKey.PublicKey, bobText1)
	s.Require().NoError(err)

	// Bob receives the message
	_, err = s.bob.HandleMessage(bobKey, &aliceKey.PublicKey, aliceMessage1.Message, defaultMessageID)
	s.Require().NoError(err)

	// Alice receives the message
	_, err = s.alice.HandleMessage(aliceKey, &bobKey.PublicKey, bobMessage1.Message, defaultMessageID)
	s.Require().NoError(err)

	// Bob replies to the message
	bobMessage2, err := s.bob.BuildEncryptedMessage(bobKey, &aliceKey.PublicKey, bobText2)
	s.Require().NoError(err)

	// Alice sends a message
	aliceMessage2, err := s.alice.BuildEncryptedMessage(aliceKey, &bobKey.PublicKey, aliceText2)
	s.Require().NoError(err)

	// Alice receives the message
	_, err = s.alice.HandleMessage(aliceKey, &bobKey.PublicKey, bobMessage2.Message, defaultMessageID)
	s.Require().NoError(err)

	// Bob receives the message
	_, err = s.bob.HandleMessage(bobKey, &aliceKey.PublicKey, aliceMessage2.Message, defaultMessageID)
	s.Require().NoError(err)
}

// Edge cases

// The bundle is lost
func (s *EncryptionServiceTestSuite) TestBundleNotExisting() {
	aliceText := []byte("alice text")

	bobKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	aliceKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	// Create a bundle without saving it
	bobBundleContainer, err := NewBundleContainer(bobKey, bobInstallationID)
	s.Require().NoError(err)

	err = SignBundle(bobKey, bobBundleContainer)
	s.Require().NoError(err)

	bobBundle := bobBundleContainer.GetBundle()

	// We add bob bundle
	_, err = s.alice.ProcessPublicBundle(aliceKey, bobBundle)
	s.Require().NoError(err)

	// Alice sends a message
	aliceMessage, err := s.alice.BuildEncryptedMessage(aliceKey, &bobKey.PublicKey, aliceText)
	s.Require().NoError(err)

	// Bob receives the message, and returns a bundlenotfound error
	_, err = s.bob.HandleMessage(bobKey, &aliceKey.PublicKey, aliceMessage.Message, defaultMessageID)
	s.Require().Error(err)
	s.Equal(errSessionNotFound, err)
}

// Device is not included in the bundle
func (s *EncryptionServiceTestSuite) TestDeviceNotIncluded() {
	bobDevice2InstallationID := "bob2"

	bobKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	aliceKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	// Create a bundle without saving it
	bobBundleContainer, err := NewBundleContainer(bobKey, bobDevice2InstallationID)
	s.Require().NoError(err)

	err = SignBundle(bobKey, bobBundleContainer)
	s.Require().NoError(err)

	bobBundle := bobBundleContainer.GetBundle()

	// We add bob bundle
	_, err = s.alice.ProcessPublicBundle(aliceKey, bobBundle)
	s.Require().NoError(err)

	// Alice sends a message
	aliceMessage, err := s.alice.BuildEncryptedMessage(aliceKey, &bobKey.PublicKey, []byte("does not matter"))
	s.Require().NoError(err)

	// Bob receives the message, and returns a bundlenotfound error
	_, err = s.bob.HandleMessage(bobKey, &aliceKey.PublicKey, aliceMessage.Message, defaultMessageID)
	s.Require().Error(err)
	s.Equal(ErrDeviceNotFound, err)
}

// A new bundle has been received
func (s *EncryptionServiceTestSuite) TestRefreshedBundle() {
	config := defaultEncryptorConfig("none", s.logger)
	// Set up refresh interval to "always"
	config.BundleRefreshInterval = 1000

	s.initDatabases(config)

	bobKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	aliceKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	// Create bundles
	bobBundle1, err := s.bob.GetBundle(bobKey)
	s.Require().NoError(err)
	s.Require().Equal(uint32(1), bobBundle1.GetSignedPreKeys()[bobInstallationID].GetVersion())

	// Sleep the required time so that bundle is refreshed
	time.Sleep(time.Duration(config.BundleRefreshInterval) * time.Millisecond)

	// Create bundles
	bobBundle2, err := s.bob.GetBundle(bobKey)
	s.Require().NoError(err)
	s.Require().Equal(uint32(2), bobBundle2.GetSignedPreKeys()[bobInstallationID].GetVersion())

	// We add the first bob bundle
	_, err = s.alice.ProcessPublicBundle(aliceKey, bobBundle1)
	s.Require().NoError(err)

	// Alice sends a message
	response1, err := s.alice.BuildEncryptedMessage(aliceKey, &bobKey.PublicKey, []byte("anything"))
	s.Require().NoError(err)
	encryptionResponse1 := response1.Message.GetEncryptedMessage()

	installationResponse1 := encryptionResponse1[bobInstallationID]
	s.Require().NotNil(installationResponse1)

	// This message is using bobBundle1

	x3dhHeader1 := installationResponse1.GetX3DHHeader()
	s.NotNil(x3dhHeader1)
	s.Equal(bobBundle1.GetSignedPreKeys()[bobInstallationID].GetSignedPreKey(), x3dhHeader1.GetId())

	// Bob decrypts the message
	_, err = s.bob.HandleMessage(bobKey, &aliceKey.PublicKey, response1.Message, defaultMessageID)
	s.Require().NoError(err)

	// We add the second bob bundle
	_, err = s.alice.ProcessPublicBundle(aliceKey, bobBundle2)
	s.Require().NoError(err)

	// Alice sends a message
	response2, err := s.alice.BuildEncryptedMessage(aliceKey, &bobKey.PublicKey, []byte("anything"))
	s.Require().NoError(err)
	encryptionResponse2 := response2.Message.GetEncryptedMessage()

	installationResponse2 := encryptionResponse2[bobInstallationID]
	s.Require().NotNil(installationResponse2)

	// This message is using bobBundle2

	x3dhHeader2 := installationResponse2.GetX3DHHeader()
	s.NotNil(x3dhHeader2)
	s.Equal(bobBundle2.GetSignedPreKeys()[bobInstallationID].GetSignedPreKey(), x3dhHeader2.GetId())

	// Bob decrypts the message
	_, err = s.bob.HandleMessage(bobKey, &aliceKey.PublicKey, response2.Message, defaultMessageID)
	s.Require().NoError(err)
}

func (s *EncryptionServiceTestSuite) TestMessageConfirmation() {
	bobText1 := []byte("bob text 1")

	bobKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	aliceKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	// Create a bundle
	bobBundle, err := s.bob.GetBundle(bobKey)
	s.Require().NoError(err)

	// We add bob bundle
	_, err = s.alice.ProcessPublicBundle(aliceKey, bobBundle)
	s.Require().NoError(err)

	// Create a bundle
	aliceBundle, err := s.alice.GetBundle(aliceKey)
	s.Require().NoError(err)

	// We add alice bundle
	_, err = s.bob.ProcessPublicBundle(bobKey, aliceBundle)
	s.Require().NoError(err)

	// Bob sends a message
	bobMessage1, err := s.bob.BuildEncryptedMessage(bobKey, &aliceKey.PublicKey, bobText1)
	s.Require().NoError(err)
	bobMessage1ID := []byte("bob-message-1-id")

	// Alice receives the message once
	_, err = s.alice.HandleMessage(aliceKey, &bobKey.PublicKey, bobMessage1.Message, bobMessage1ID)
	s.Require().NoError(err)

	// Alice receives the message twice
	_, err = s.alice.HandleMessage(aliceKey, &bobKey.PublicKey, bobMessage1.Message, bobMessage1ID)
	s.Require().NoError(err)

	// Alice confirms the message
	err = s.alice.ConfirmMessageProcessed(bobMessage1ID)
	s.Require().NoError(err)

	// Alice decrypts it again, it should fail
	_, err = s.alice.HandleMessage(aliceKey, &bobKey.PublicKey, bobMessage1.Message, bobMessage1ID)
	s.Require().Equal(errors.New("can't skip current chain message keys: bad until: probably an out-of-order message that was deleted"), err)

	// Bob sends a message
	bobMessage2, err := s.bob.BuildEncryptedMessage(bobKey, &aliceKey.PublicKey, bobText1)
	s.Require().NoError(err)
	bobMessage2ID := []byte("bob-message-2-id")

	// Bob sends a message
	bobMessage3, err := s.bob.BuildEncryptedMessage(bobKey, &aliceKey.PublicKey, bobText1)
	s.Require().NoError(err)
	bobMessage3ID := []byte("bob-message-3-id")

	// Alice receives message 3 once
	_, err = s.alice.HandleMessage(aliceKey, &bobKey.PublicKey, bobMessage3.Message, bobMessage3ID)
	s.Require().NoError(err)

	// Alice receives message 3 twice
	_, err = s.alice.HandleMessage(aliceKey, &bobKey.PublicKey, bobMessage3.Message, bobMessage3ID)
	s.Require().NoError(err)

	// Alice receives message 2 once
	_, err = s.alice.HandleMessage(aliceKey, &bobKey.PublicKey, bobMessage2.Message, bobMessage2ID)
	s.Require().NoError(err)

	// Alice receives message 2 twice
	_, err = s.alice.HandleMessage(aliceKey, &bobKey.PublicKey, bobMessage2.Message, bobMessage2ID)
	s.Require().NoError(err)

	// Alice confirms the messages
	err = s.alice.ConfirmMessageProcessed(bobMessage2ID)
	s.Require().NoError(err)
	err = s.alice.ConfirmMessageProcessed(bobMessage3ID)
	s.Require().NoError(err)

	// Alice decrypts it again, it should fail
	_, err = s.alice.HandleMessage(aliceKey, &bobKey.PublicKey, bobMessage3.Message, bobMessage3ID)
	s.Require().Equal(errors.New("can't skip current chain message keys: bad until: probably an out-of-order message that was deleted"), err)

	// Alice decrypts it again, it should fail
	_, err = s.alice.HandleMessage(aliceKey, &bobKey.PublicKey, bobMessage2.Message, bobMessage2ID)
	s.Require().Equal(errors.New("can't skip current chain message keys: bad until: probably an out-of-order message that was deleted"), err)
}

// Tests:

// 1) Client has an old key, it upgrades, receives a message from an old client
// 2) Client has an old key, it upgrades, receives a message from a new client

func (s *EncryptionServiceTestSuite) TestHashRatchetCompatibility() {
	aliceKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	bobKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	groupID := []byte("test_community_id")
	s.Require().NotNil(aliceKey)
	s.Require().NotNil(bobKey)

	// We create a hash ratchet on bob
	s.logger.Info("Hash ratchet key exchange 1")
	keyID1, err := s.bob.encryptor.GenerateHashRatchetKey(groupID)
	s.Require().NoError(err)

	// We replicate the same error condition
	timestamp32 := keyID1.DeprecatedKeyID()
	_, err = s.alice.encryptor.persistence.DB.Exec("INSERT INTO hash_ratchet_encryption(group_id, deprecated_key_id, key, key_id) VALUES(?,?,?,?)", groupID, timestamp32, keyID1.Key, append(groupID, []byte("some-bytes")...))
	s.Require().NoError(err)

	// We migrate
	_, err = s.alice.encryptor.persistence.DB.Exec("UPDATE hash_ratchet_encryption SET key_timestamp = deprecated_key_id")
	s.Require().NoError(err)

	payload1 := []byte("community msg 1")
	hashRatchetMsg1, err := s.bob.BuildHashRatchetMessage(groupID, payload1)

	s.Require().NoError(err)
	s.Require().NotNil(hashRatchetMsg1)
	s.Require().NotNil(hashRatchetMsg1.Message)

	// We remove groupID, as that's whats coming from older clients
	hashRatchetMsg1.Message.EncryptedMessage["none"].HRHeader.KeyId = nil
	s.Require().NotEmpty(hashRatchetMsg1.Message.EncryptedMessage["none"].HRHeader.DeprecatedKeyId)
	s.Require().Equal(timestamp32, hashRatchetMsg1.Message.EncryptedMessage["none"].HRHeader.DeprecatedKeyId)

	decryptedResponse, err := s.alice.HandleMessage(aliceKey, nil, hashRatchetMsg1.Message, defaultMessageID)
	s.Require().NoError(err)
	s.Require().NotEmpty(decryptedResponse)

	// New message structure, on old key
	hashRatchetMsg2, err := s.bob.BuildHashRatchetMessage(groupID, payload1)

	s.Require().NoError(err)
	s.Require().NotNil(hashRatchetMsg2)
	s.Require().NotNil(hashRatchetMsg2.Message)

	s.Require().NotEmpty(hashRatchetMsg2.Message.EncryptedMessage["none"].HRHeader.DeprecatedKeyId)
	s.Require().NotEmpty(hashRatchetMsg2.Message.EncryptedMessage["none"].HRHeader.KeyId)
	s.Require().Equal(timestamp32, hashRatchetMsg2.Message.EncryptedMessage["none"].HRHeader.DeprecatedKeyId)

	decryptedResponse, err = s.alice.HandleMessage(aliceKey, nil, hashRatchetMsg2.Message, defaultMessageID)
	s.Require().NoError(err)
	s.Require().NotEmpty(decryptedResponse)
}

func (s *EncryptionServiceTestSuite) TestHashRatchetRekey() {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	groupID := []byte{0x4}
	var timestamp uint64 = 10
	keyID1, err := s.alice.encryptor.GenerateHashRatchetKey(groupID)
	s.Require().NoError(err)

	keyMaterial := keyID1.Key

	key1String := "e8395a5d2289d14d47f5f5c506001a2b4f039d96ebf576a6a39e5f23c7a9c618"

	key1, err := crypto.HexToECDSA(key1String)
	s.Require().NoError(err)

	key1KeyBytes := publicKeyMostRelevantBytes(&key1.PublicKey)

	s.Require().Equal(uint32(0x6da634b9), key1KeyBytes)

	key2String := "e8395a5d5b3c4081c0e1f63c5d588c6f2c4ba7c6ec590f5f8e1a96b48f5e6e7e"
	key2, err := crypto.HexToECDSA(key2String)
	s.Require().NoError(err)

	key2KeyBytes := publicKeyMostRelevantBytes(&key2.PublicKey)
	s.Require().Equal(uint32(0x72d6c574), key2KeyBytes)

	messages, err := buildGroupRekeyMessage(privateKey, groupID, timestamp, keyMaterial, []*ecdsa.PublicKey{&key1.PublicKey, &key1.PublicKey, &key1.PublicKey, &key2.PublicKey}, 10)
	s.Require().NoError(err)

	message := messages[0]

	_, err = proto.Marshal(message)
	s.Require().NoError(err)

	s.Require().Equal(timestamp, message.Timestamp)

	s.Require().NotNil(message.Keys)

	s.Require().NotEmpty(message.Keys[key1KeyBytes])
	s.Require().NotEmpty(message.Keys[key2KeyBytes])

	s.Require().Len(message.Keys[key1KeyBytes], 180)
	s.Require().Len(message.Keys[key2KeyBytes], 60)

	// We break the first one, so that we simulate two keys with different encryption but
	// starting from the same

	message.Keys[key1KeyBytes][keySize-2] = byte(3)
	message.Keys[key1KeyBytes][keySize+2] = byte(3)

	// Easier case, key does not need to be split
	decryptedKey2, err := decryptGroupRekeyMessage(key2, &privateKey.PublicKey, message)
	s.Require().NoError(err)
	s.Require().Equal(decryptedKey2, keyMaterial)

	decryptedKey1, err := decryptGroupRekeyMessage(key1, &privateKey.PublicKey, message)
	s.Require().NoError(err)
	s.Require().Equal(decryptedKey1, keyMaterial)

	messages, err = buildGroupRekeyMessage(privateKey, groupID, timestamp+1, keyMaterial, []*ecdsa.PublicKey{&key1.PublicKey, &key1.PublicKey, &key1.PublicKey, &key2.PublicKey}, 2)
	s.Require().NoError(err)
	s.Require().Len(messages, 2)
}

// We test that adding a new field and leaving the old blank won't crash the app
func (s *EncryptionServiceTestSuite) TestHashRatchetRekeyCompatibility() {
	_, err := s.alice.HandleHashRatchetKeys([]byte{0x1}, nil, nil, nil)
	s.Require().NoError(err)
}

// We test that adding a new field and leaving the old blank won't crash the app
func (s *EncryptionServiceTestSuite) TestHashRatchetRekeyHandleRatchet() {
	aliceKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	bobKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	groupID := []byte{0x1}
	specs, err := s.alice.BuildHashRatchetReKeyGroupMessage(aliceKey, []*ecdsa.PublicKey{&bobKey.PublicKey}, groupID, nil)
	s.Require().NoError(err)
	s.Require().Len(specs, 1)

	response, err := s.bob.HandleMessage(bobKey, &aliceKey.PublicKey, specs[0].Message, []byte{0x2})
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.HashRatchetInfo, 1)
}
