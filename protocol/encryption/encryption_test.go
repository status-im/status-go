package encryption

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/status-im/status-go/protocol/sqlite"

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
	logger      *zap.Logger
	alice       *Protocol
	bob         *Protocol
	aliceDBPath *os.File
	bobDBPath   *os.File
}

func (s *EncryptionServiceTestSuite) initDatabases(config encryptorConfig) {
	var err error

	s.aliceDBPath, err = ioutil.TempFile("", "alice.db.sql")
	s.Require().NoError(err)

	s.bobDBPath, err = ioutil.TempFile("", "bob.db.sql")
	s.Require().NoError(err)

	db, err := sqlite.Open(s.aliceDBPath.Name(), "alice-key")
	s.Require().NoError(err)
	config.InstallationID = aliceInstallationID
	s.alice = NewWithEncryptorConfig(
		db,
		aliceInstallationID,
		config,
		s.logger.With(zap.String("user", "alice")),
	)

	db, err = sqlite.Open(s.bobDBPath.Name(), "bob-key")
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
	s.logger = zap.NewNop()
	s.initDatabases(defaultEncryptorConfig("none", s.logger))
}

func (s *EncryptionServiceTestSuite) TearDownTest() {
	os.Remove(s.aliceDBPath.Name())
	os.Remove(s.bobDBPath.Name())
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

// Alice sends Bob an encrypted message with DH using an ephemeral key
// and Bob's identity key.
// Bob is able to decrypt it.
// Alice does not re-use the symmetric key
func (s *EncryptionServiceTestSuite) TestEncryptPayloadNoBundle() {
	bobKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	aliceKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	response1, err := s.alice.BuildDirectMessage(aliceKey, &bobKey.PublicKey, cleartext)
	s.Require().NoError(err)

	encryptionResponse1 := response1.Message.GetDirectMessage()

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
	response2, err := s.alice.BuildDirectMessage(aliceKey, &bobKey.PublicKey, cleartext)
	s.Require().NoError(err)

	encryptionResponse2 := response2.Message.GetDirectMessage()

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
	response1, err := s.alice.BuildDirectMessage(aliceKey, &bobKey.PublicKey, cleartext)
	s.Require().NoError(err)

	encryptionResponse1 := response1.Message.GetDirectMessage()

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
	_, err = s.alice.BuildDirectMessage(aliceKey, &bobKey.PublicKey, cleartext1)
	s.Require().NoError(err)

	// We send another message using the bundle
	response, err := s.alice.BuildDirectMessage(aliceKey, &bobKey.PublicKey, cleartext2)
	s.Require().NoError(err)
	encryptionResponse := response.Message.GetDirectMessage()

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
	response, err := s.alice.BuildDirectMessage(aliceKey, &bobKey.PublicKey, cleartext1)
	s.Require().NoError(err)

	// Bob receives the message
	_, err = s.bob.HandleMessage(bobKey, &aliceKey.PublicKey, response.Message, defaultMessageID)
	s.Require().NoError(err)

	// Bob replies to the message
	response, err = s.bob.BuildDirectMessage(bobKey, &aliceKey.PublicKey, cleartext1)
	s.Require().NoError(err)

	// Alice receives the message
	_, err = s.alice.HandleMessage(aliceKey, &bobKey.PublicKey, response.Message, defaultMessageID)
	s.Require().NoError(err)

	// We send another message using the bundle
	response, err = s.alice.BuildDirectMessage(aliceKey, &bobKey.PublicKey, cleartext2)
	s.Require().NoError(err)
	encryptionResponse := response.Message.GetDirectMessage()

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
		_, err = s.bob.BuildDirectMessage(bobKey, &aliceKey.PublicKey, bobText)
		s.Require().NoError(err)
	}

	// Bob sends a message
	bobMessage1, err := s.bob.BuildDirectMessage(bobKey, &aliceKey.PublicKey, bobText)
	s.Require().NoError(err)

	// Alice receives the message
	_, err = s.alice.HandleMessage(aliceKey, &bobKey.PublicKey, bobMessage1.Message, defaultMessageID)
	s.Require().NoError(err)

	// Bob sends a message
	_, err = s.bob.BuildDirectMessage(bobKey, &aliceKey.PublicKey, bobText)
	s.Require().NoError(err)

	// Bob sends a message
	bobMessage2, err := s.bob.BuildDirectMessage(bobKey, &aliceKey.PublicKey, bobText)
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
		_, err = s.bob.BuildDirectMessage(bobKey, &aliceKey.PublicKey, bobText)
		s.Require().NoError(err)
	}

	// Bob sends a message
	bobMessage1, err := s.bob.BuildDirectMessage(bobKey, &aliceKey.PublicKey, bobText)
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
		m, err := s.bob.BuildDirectMessage(bobKey, &aliceKey.PublicKey, bobText)
		s.Require().NoError(err)

		messages[i] = m.Message
	}

	// Another message to trigger the deletion
	m, err := s.bob.BuildDirectMessage(bobKey, &aliceKey.PublicKey, bobText)
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
		m, err := s.bob.BuildDirectMessage(bobKey, &aliceKey.PublicKey, bobText)
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
	aliceMessage1, err := s.alice.BuildDirectMessage(aliceKey, &bobKey.PublicKey, aliceText1)
	s.Require().NoError(err)

	// Bob sends a message
	bobMessage1, err := s.bob.BuildDirectMessage(bobKey, &aliceKey.PublicKey, bobText1)
	s.Require().NoError(err)

	// Bob receives the message
	_, err = s.bob.HandleMessage(bobKey, &aliceKey.PublicKey, aliceMessage1.Message, defaultMessageID)
	s.Require().NoError(err)

	// Alice receives the message
	_, err = s.alice.HandleMessage(aliceKey, &bobKey.PublicKey, bobMessage1.Message, defaultMessageID)
	s.Require().NoError(err)

	// Bob replies to the message
	bobMessage2, err := s.bob.BuildDirectMessage(bobKey, &aliceKey.PublicKey, bobText2)
	s.Require().NoError(err)

	// Alice sends a message
	aliceMessage2, err := s.alice.BuildDirectMessage(aliceKey, &bobKey.PublicKey, aliceText2)
	s.Require().NoError(err)

	// Alice receives the message
	_, err = s.alice.HandleMessage(aliceKey, &bobKey.PublicKey, bobMessage2.Message, defaultMessageID)
	s.Require().NoError(err)

	// Bob receives the message
	_, err = s.bob.HandleMessage(bobKey, &aliceKey.PublicKey, aliceMessage2.Message, defaultMessageID)
	s.Require().NoError(err)
}

func publish(
	e *Protocol,
	privateKey *ecdsa.PrivateKey,
	publicKey *ecdsa.PublicKey,
	errChan chan error,
	output chan *ProtocolMessage,
) {
	var wg sync.WaitGroup

	for i := 0; i < 200; i++ {

		// Simulate 5% of the messages dropped
		if rand.Intn(100) <= 95 {
			wg.Add(1)
			// Simulate out of order messages
			go func() {
				defer wg.Done()
				time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
				response, err := e.BuildDirectMessage(privateKey, publicKey, cleartext)
				if err != nil {
					errChan <- err
					return
				}

				output <- response.Message
			}()
		}
	}
	wg.Wait()
	close(output)
	close(errChan)
}

func receiver(
	s *Protocol,
	privateKey *ecdsa.PrivateKey,
	publicKey *ecdsa.PublicKey,
	errChan chan error,
	input chan *ProtocolMessage,
) {
	i := 0

	for payload := range input {
		actualCleartext, err := s.HandleMessage(privateKey, publicKey, payload, defaultMessageID)
		if err != nil {
			errChan <- err
			return
		}
		if !reflect.DeepEqual(actualCleartext.DecryptedMessage, cleartext) {
			errChan <- errors.New("Decrypted value does not match")
			return
		}
		i++
	}
	close(errChan)
}

func (s *EncryptionServiceTestSuite) TestRandomised() {

	seed := time.Now().UTC().UnixNano()
	rand.Seed(seed)

	// Print so that if it fails it can be replicated
	fmt.Printf("Starting test with seed: %x\n", seed)

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

	aliceChan := make(chan *ProtocolMessage, 100)
	bobChan := make(chan *ProtocolMessage, 100)

	alicePublisherErrChan := make(chan error, 1)
	bobPublisherErrChan := make(chan error, 1)

	aliceReceiverErrChan := make(chan error, 1)
	bobReceiverErrChan := make(chan error, 1)

	// Set up alice publishe
	go publish(s.alice, aliceKey, &bobKey.PublicKey, alicePublisherErrChan, bobChan)
	// Set up bob publisher
	go publish(s.bob, bobKey, &aliceKey.PublicKey, bobPublisherErrChan, aliceChan)

	// Set up bob receiver
	go receiver(s.bob, bobKey, &aliceKey.PublicKey, bobReceiverErrChan, bobChan)

	// Set up alice receiver
	go receiver(s.alice, aliceKey, &bobKey.PublicKey, aliceReceiverErrChan, aliceChan)

	aliceErr := <-alicePublisherErrChan
	s.Require().NoError(aliceErr)

	bobErr := <-bobPublisherErrChan
	s.Require().NoError(bobErr)

	aliceErr = <-aliceReceiverErrChan
	s.Require().NoError(aliceErr)

	bobErr = <-bobReceiverErrChan
	s.Require().NoError(bobErr)
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
	aliceMessage, err := s.alice.BuildDirectMessage(aliceKey, &bobKey.PublicKey, aliceText)
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
	aliceMessage, err := s.alice.BuildDirectMessage(aliceKey, &bobKey.PublicKey, []byte("does not matter"))
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
	response1, err := s.alice.BuildDirectMessage(aliceKey, &bobKey.PublicKey, []byte("anything"))
	s.Require().NoError(err)
	encryptionResponse1 := response1.Message.GetDirectMessage()

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
	response2, err := s.alice.BuildDirectMessage(aliceKey, &bobKey.PublicKey, []byte("anything"))
	s.Require().NoError(err)
	encryptionResponse2 := response2.Message.GetDirectMessage()

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
	bobMessage1, err := s.bob.BuildDirectMessage(bobKey, &aliceKey.PublicKey, bobText1)
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
	bobMessage2, err := s.bob.BuildDirectMessage(bobKey, &aliceKey.PublicKey, bobText1)
	s.Require().NoError(err)
	bobMessage2ID := []byte("bob-message-2-id")

	// Bob sends a message
	bobMessage3, err := s.bob.BuildDirectMessage(bobKey, &aliceKey.PublicKey, bobText1)
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
