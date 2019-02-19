package chat

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

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/suite"
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
	alice       *EncryptionService
	bob         *EncryptionService
	aliceDBPath string
	bobDBPath   string
}

func (s *EncryptionServiceTestSuite) initDatabases(baseConfig *EncryptionServiceConfig) {

	aliceDBFile, err := ioutil.TempFile(os.TempDir(), "alice")
	s.Require().NoError(err)
	aliceDBPath := aliceDBFile.Name()

	bobDBFile, err := ioutil.TempFile(os.TempDir(), "bob")
	s.Require().NoError(err)
	bobDBPath := bobDBFile.Name()

	s.aliceDBPath = aliceDBPath
	s.bobDBPath = bobDBPath

	if baseConfig == nil {
		config := DefaultEncryptionServiceConfig(aliceInstallationID)
		baseConfig = &config
	}

	const (
		aliceDBKey = "alice"
		bobDBKey   = "bob"
	)

	alicePersistence, err := NewSQLLitePersistence(aliceDBPath, aliceDBKey)
	if err != nil {
		panic(err)
	}

	bobPersistence, err := NewSQLLitePersistence(bobDBPath, bobDBKey)
	if err != nil {
		panic(err)
	}

	baseConfig.InstallationID = aliceInstallationID
	s.alice = NewEncryptionService(alicePersistence, *baseConfig)

	baseConfig.InstallationID = bobInstallationID
	s.bob = NewEncryptionService(bobPersistence, *baseConfig)
}

func (s *EncryptionServiceTestSuite) SetupTest() {
	s.initDatabases(nil)
}

func (s *EncryptionServiceTestSuite) TearDownTest() {
	os.Remove(s.aliceDBPath)
	os.Remove(s.bobDBPath)
}

func (s *EncryptionServiceTestSuite) TestCreateBundle() {
	aliceKey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	aliceBundle1, err := s.alice.CreateBundle(aliceKey)
	s.Require().NoError(err)
	s.NotNil(aliceBundle1, "It creates a bundle")

	aliceBundle2, err := s.alice.CreateBundle(aliceKey)
	s.Require().NoError(err)
	s.Equal(aliceBundle1, aliceBundle2, "It returns the same bundle")
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

	encryptionResponse1, err := s.alice.EncryptPayload(&bobKey.PublicKey, aliceKey, cleartext)
	s.Require().NoError(err)

	installationResponse1 := encryptionResponse1["none"]
	// That's for any device
	s.Require().NotNil(installationResponse1)

	cyphertext1 := installationResponse1.Payload
	ephemeralKey1 := installationResponse1.GetDHHeader().GetKey()
	s.NotNil(ephemeralKey1, "It generates an ephemeral key for DH exchange")
	s.NotNil(cyphertext1, "It generates an encrypted payload")
	s.NotEqual(cyphertext1, cleartext, "It encrypts the payload correctly")

	// On the receiver side, we should be able to decrypt using our private key and the ephemeral just sent
	decryptedPayload1, err := s.bob.DecryptPayload(bobKey, &aliceKey.PublicKey, aliceInstallationID, encryptionResponse1, defaultMessageID)
	s.Require().NoError(err)
	s.Equal(cleartext, decryptedPayload1, "It correctly decrypts the payload using DH")

	// The next message will not be re-using the same key
	encryptionResponse2, err := s.alice.EncryptPayload(&bobKey.PublicKey, aliceKey, cleartext)
	s.Require().NoError(err)

	installationResponse2 := encryptionResponse2[aliceInstallationID]

	cyphertext2 := installationResponse2.GetPayload()
	ephemeralKey2 := installationResponse2.GetDHHeader().GetKey()
	s.NotEqual(cyphertext1, cyphertext2, "It does not re-use the symmetric key")
	s.NotEqual(ephemeralKey1, ephemeralKey2, "It does not re-use the ephemeral key")

	decryptedPayload2, err := s.bob.DecryptPayload(bobKey, &aliceKey.PublicKey, aliceInstallationID, encryptionResponse2, defaultMessageID)
	s.Require().NoError(err)
	s.Equal(cleartext, decryptedPayload2, "It correctly decrypts the payload using DH")
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
	bobBundle, err := s.bob.CreateBundle(bobKey)
	s.Require().NoError(err)

	// We add bob bundle
	_, err = s.alice.ProcessPublicBundle(aliceKey, bobBundle)
	s.Require().NoError(err)

	// We send a message using the bundle
	encryptionResponse1, err := s.alice.EncryptPayload(&bobKey.PublicKey, aliceKey, cleartext)
	s.Require().NoError(err)

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
	decryptedPayload1, err := s.bob.DecryptPayload(bobKey, &aliceKey.PublicKey, aliceInstallationID, encryptionResponse1, defaultMessageID)
	s.Require().NoError(err)
	s.Equal(cleartext, decryptedPayload1, "It correctly decrypts the payload using X3DH")
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
	bobBundle, err := s.bob.CreateBundle(bobKey)
	s.Require().NoError(err)

	// We add bob bundle
	_, err = s.alice.ProcessPublicBundle(aliceKey, bobBundle)
	s.Require().NoError(err)

	// We send a message using the bundle
	_, err = s.alice.EncryptPayload(&bobKey.PublicKey, aliceKey, cleartext1)
	s.Require().NoError(err)

	// We send another message using the bundle
	encryptionResponse, err := s.alice.EncryptPayload(&bobKey.PublicKey, aliceKey, cleartext2)
	s.Require().NoError(err)

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
	decryptedPayload1, err := s.bob.DecryptPayload(bobKey, &aliceKey.PublicKey, aliceInstallationID, encryptionResponse, defaultMessageID)
	s.Require().NoError(err)

	s.Equal(cleartext2, decryptedPayload1, "It correctly decrypts the payload using X3DH")
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
	bobBundle, err := s.bob.CreateBundle(bobKey)
	s.Require().NoError(err)

	// Create a bundle
	aliceBundle, err := s.alice.CreateBundle(aliceKey)
	s.Require().NoError(err)

	// We add bob bundle
	_, err = s.alice.ProcessPublicBundle(aliceKey, bobBundle)
	s.Require().NoError(err)

	// We add alice bundle
	_, err = s.bob.ProcessPublicBundle(bobKey, aliceBundle)
	s.Require().NoError(err)

	// Alice sends a message
	encryptionResponse, err := s.alice.EncryptPayload(&bobKey.PublicKey, aliceKey, cleartext1)
	s.Require().NoError(err)

	// Bob receives the message
	_, err = s.bob.DecryptPayload(bobKey, &aliceKey.PublicKey, aliceInstallationID, encryptionResponse, defaultMessageID)
	s.Require().NoError(err)

	// Bob replies to the message
	encryptionResponse, err = s.bob.EncryptPayload(&aliceKey.PublicKey, bobKey, cleartext1)
	s.Require().NoError(err)

	// Alice receives the message
	_, err = s.alice.DecryptPayload(aliceKey, &bobKey.PublicKey, bobInstallationID, encryptionResponse, defaultMessageID)
	s.Require().NoError(err)

	// We send another message using the bundle
	encryptionResponse, err = s.alice.EncryptPayload(&bobKey.PublicKey, aliceKey, cleartext2)
	s.Require().NoError(err)

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
	decryptedPayload1, err := s.bob.DecryptPayload(bobKey, &aliceKey.PublicKey, aliceInstallationID, encryptionResponse, defaultMessageID)
	s.Require().NoError(err)

	s.Equal(cleartext2, decryptedPayload1, "It correctly decrypts the payload using X3DH")
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
	bobBundle, err := s.bob.CreateBundle(bobKey)
	s.Require().NoError(err)

	// We add bob bundle
	_, err = s.alice.ProcessPublicBundle(aliceKey, bobBundle)
	s.Require().NoError(err)

	// Create a bundle
	aliceBundle, err := s.alice.CreateBundle(aliceKey)
	s.Require().NoError(err)

	// We add alice bundle
	_, err = s.bob.ProcessPublicBundle(bobKey, aliceBundle)
	s.Require().NoError(err)

	// Bob sends a message

	for i := 0; i < s.alice.config.MaxSkip; i++ {
		_, err = s.bob.EncryptPayload(&aliceKey.PublicKey, bobKey, bobText)
		s.Require().NoError(err)
	}

	// Bob sends a message
	bobMessage1, err := s.bob.EncryptPayload(&aliceKey.PublicKey, bobKey, bobText)
	s.Require().NoError(err)

	// Alice receives the message
	_, err = s.alice.DecryptPayload(aliceKey, &bobKey.PublicKey, bobInstallationID, bobMessage1, defaultMessageID)
	s.Require().NoError(err)

	// Bob sends a message
	_, err = s.bob.EncryptPayload(&aliceKey.PublicKey, bobKey, bobText)
	s.Require().NoError(err)

	// Bob sends a message
	bobMessage2, err := s.bob.EncryptPayload(&aliceKey.PublicKey, bobKey, bobText)
	s.Require().NoError(err)

	// Alice receives the message, we should have maxSkip + 1 keys in the db, but
	// we should not throw an error
	_, err = s.alice.DecryptPayload(aliceKey, &bobKey.PublicKey, bobInstallationID, bobMessage2, defaultMessageID)
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
	bobBundle, err := s.bob.CreateBundle(bobKey)
	s.Require().NoError(err)

	// We add bob bundle
	_, err = s.alice.ProcessPublicBundle(aliceKey, bobBundle)
	s.Require().NoError(err)

	// Create a bundle
	aliceBundle, err := s.alice.CreateBundle(aliceKey)
	s.Require().NoError(err)

	// We add alice bundle
	_, err = s.bob.ProcessPublicBundle(bobKey, aliceBundle)
	s.Require().NoError(err)

	// Bob sends a message

	for i := 0; i < s.alice.config.MaxSkip+1; i++ {
		_, err = s.bob.EncryptPayload(&aliceKey.PublicKey, bobKey, bobText)
		s.Require().NoError(err)
	}

	// Bob sends a message
	bobMessage1, err := s.bob.EncryptPayload(&aliceKey.PublicKey, bobKey, bobText)
	s.Require().NoError(err)

	// Alice receives the message
	_, err = s.alice.DecryptPayload(aliceKey, &bobKey.PublicKey, bobInstallationID, bobMessage1, defaultMessageID)
	s.Require().Equal(errors.New("can't skip current chain message keys: too many messages"), err)
}

func (s *EncryptionServiceTestSuite) TestMaxMessageKeysPerSession() {
	config := DefaultEncryptionServiceConfig("none")
	// Set MaxKeep and MaxSkip to an high value so it does not interfere
	config.MaxKeep = 100000
	config.MaxSkip = 100000

	s.initDatabases(&config)

	bobText := []byte("text")

	bobKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	aliceKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	// Create a bundle
	bobBundle, err := s.bob.CreateBundle(bobKey)
	s.Require().NoError(err)

	// We add bob bundle
	_, err = s.alice.ProcessPublicBundle(aliceKey, bobBundle)
	s.Require().NoError(err)

	// Create a bundle
	aliceBundle, err := s.alice.CreateBundle(aliceKey)
	s.Require().NoError(err)

	// We add alice bundle
	_, err = s.bob.ProcessPublicBundle(bobKey, aliceBundle)
	s.Require().NoError(err)

	// We create just enough messages so that the first key should be deleted

	nMessages := s.alice.config.MaxMessageKeysPerSession
	messages := make([]map[string]*DirectMessageProtocol, nMessages)
	for i := 0; i < nMessages; i++ {
		m, err := s.bob.EncryptPayload(&aliceKey.PublicKey, bobKey, bobText)
		s.Require().NoError(err)

		messages[i] = m
	}

	// Another message to trigger the deletion
	m, err := s.bob.EncryptPayload(&aliceKey.PublicKey, bobKey, bobText)
	s.Require().NoError(err)
	_, err = s.alice.DecryptPayload(aliceKey, &bobKey.PublicKey, bobInstallationID, m, defaultMessageID)
	s.Require().NoError(err)

	// We decrypt the first message, and it should fail
	_, err = s.alice.DecryptPayload(aliceKey, &bobKey.PublicKey, bobInstallationID, messages[0], defaultMessageID)
	s.Require().Equal(errors.New("can't skip current chain message keys: bad until: probably an out-of-order message that was deleted"), err)

	// We decrypt the second message, and it should be decrypted
	_, err = s.alice.DecryptPayload(aliceKey, &bobKey.PublicKey, bobInstallationID, messages[1], defaultMessageID)
	s.Require().NoError(err)
}

func (s *EncryptionServiceTestSuite) TestMaxKeep() {
	config := DefaultEncryptionServiceConfig("none")
	// Set MaxMessageKeysPerSession to an high value so it does not interfere
	config.MaxMessageKeysPerSession = 100000

	s.initDatabases(&config)

	bobText := []byte("text")

	bobKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	aliceKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	// Create a bundle
	bobBundle, err := s.bob.CreateBundle(bobKey)
	s.Require().NoError(err)

	// We add bob bundle
	_, err = s.alice.ProcessPublicBundle(aliceKey, bobBundle)
	s.Require().NoError(err)

	// Create a bundle
	aliceBundle, err := s.alice.CreateBundle(aliceKey)
	s.Require().NoError(err)

	// We add alice bundle
	_, err = s.bob.ProcessPublicBundle(bobKey, aliceBundle)
	s.Require().NoError(err)

	// We decrypt all messages but 1 & 2
	messages := make([]map[string]*DirectMessageProtocol, s.alice.config.MaxKeep)
	for i := 0; i < s.alice.config.MaxKeep; i++ {
		m, err := s.bob.EncryptPayload(&aliceKey.PublicKey, bobKey, bobText)
		messages[i] = m
		s.Require().NoError(err)

		if i != 0 && i != 1 {
			messageID := []byte(fmt.Sprintf("%d", i))
			_, err = s.alice.DecryptPayload(aliceKey, &bobKey.PublicKey, bobInstallationID, m, messageID)
			s.Require().NoError(err)
			err = s.alice.ConfirmMessagesProcessed([][]byte{messageID})
			s.Require().NoError(err)
		}

	}

	// We decrypt the first message, and it should fail, as it should have been removed
	_, err = s.alice.DecryptPayload(aliceKey, &bobKey.PublicKey, bobInstallationID, messages[0], defaultMessageID)
	s.Require().Equal(errors.New("can't skip current chain message keys: bad until: probably an out-of-order message that was deleted"), err)

	// We decrypt the second message, and it should be decrypted
	_, err = s.alice.DecryptPayload(aliceKey, &bobKey.PublicKey, bobInstallationID, messages[1], defaultMessageID)
	s.Require().NoError(err)
}

// Alice has Bob's bundle
// Bob has Alice's bundle
// Bob sends a message to alice
// Alice sends a message to Bob
// Bob receives alice message
// Alice receives Bob message
// Bob sends another message to alice and viceversa

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
	bobBundle, err := s.bob.CreateBundle(bobKey)
	s.Require().NoError(err)

	// We add bob bundle
	_, err = s.alice.ProcessPublicBundle(aliceKey, bobBundle)
	s.Require().NoError(err)

	// Create a bundle
	aliceBundle, err := s.alice.CreateBundle(aliceKey)
	s.Require().NoError(err)

	// We add alice bundle
	_, err = s.bob.ProcessPublicBundle(bobKey, aliceBundle)
	s.Require().NoError(err)

	// Alice sends a message
	aliceMessage1, err := s.alice.EncryptPayload(&bobKey.PublicKey, aliceKey, aliceText1)
	s.Require().NoError(err)

	// Bob sends a message
	bobMessage1, err := s.bob.EncryptPayload(&aliceKey.PublicKey, bobKey, bobText1)
	s.Require().NoError(err)

	// Bob receives the message
	_, err = s.bob.DecryptPayload(bobKey, &aliceKey.PublicKey, aliceInstallationID, aliceMessage1, defaultMessageID)
	s.Require().NoError(err)

	// Alice receives the message
	_, err = s.alice.DecryptPayload(aliceKey, &bobKey.PublicKey, bobInstallationID, bobMessage1, defaultMessageID)
	s.Require().NoError(err)

	// Bob replies to the message
	bobMessage2, err := s.bob.EncryptPayload(&aliceKey.PublicKey, bobKey, bobText2)
	s.Require().NoError(err)

	// Alice sends a message
	aliceMessage2, err := s.alice.EncryptPayload(&bobKey.PublicKey, aliceKey, aliceText2)
	s.Require().NoError(err)

	// Alice receives the message
	_, err = s.alice.DecryptPayload(aliceKey, &bobKey.PublicKey, bobInstallationID, bobMessage2, defaultMessageID)
	s.Require().NoError(err)

	// Bob receives the message
	_, err = s.bob.DecryptPayload(bobKey, &aliceKey.PublicKey, aliceInstallationID, aliceMessage2, defaultMessageID)
	s.Require().NoError(err)
}

func publisher(
	e *EncryptionService,
	privateKey *ecdsa.PrivateKey,
	publicKey *ecdsa.PublicKey,
	errChan chan error,
	output chan map[string]*DirectMessageProtocol,
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
				response, err := e.EncryptPayload(publicKey, privateKey, cleartext)
				if err != nil {
					errChan <- err
					return
				}

				output <- response
			}()
		}
	}
	wg.Wait()
	close(output)
	close(errChan)
}

func receiver(
	s *EncryptionService,
	privateKey *ecdsa.PrivateKey,
	publicKey *ecdsa.PublicKey,
	installationID string,
	errChan chan error,
	input chan map[string]*DirectMessageProtocol,
) {
	i := 0

	for payload := range input {
		actualCleartext, err := s.DecryptPayload(privateKey, publicKey, installationID, payload, defaultMessageID)
		if err != nil {
			errChan <- err
			return
		}
		if !reflect.DeepEqual(actualCleartext, cleartext) {
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
	bobBundle, err := s.bob.CreateBundle(bobKey)
	s.Require().NoError(err)

	// We add bob bundle
	_, err = s.alice.ProcessPublicBundle(aliceKey, bobBundle)
	s.Require().NoError(err)

	// Create a bundle
	aliceBundle, err := s.alice.CreateBundle(aliceKey)
	s.Require().NoError(err)

	// We add alice bundle
	_, err = s.bob.ProcessPublicBundle(bobKey, aliceBundle)
	s.Require().NoError(err)

	aliceChan := make(chan map[string]*DirectMessageProtocol, 100)
	bobChan := make(chan map[string]*DirectMessageProtocol, 100)

	alicePublisherErrChan := make(chan error, 1)
	bobPublisherErrChan := make(chan error, 1)

	aliceReceiverErrChan := make(chan error, 1)
	bobReceiverErrChan := make(chan error, 1)

	// Set up alice publishe
	go publisher(s.alice, aliceKey, &bobKey.PublicKey, alicePublisherErrChan, bobChan)
	// Set up bob publisher
	go publisher(s.bob, bobKey, &aliceKey.PublicKey, bobPublisherErrChan, aliceChan)

	// Set up bob receiver
	go receiver(s.bob, bobKey, &aliceKey.PublicKey, aliceInstallationID, bobReceiverErrChan, bobChan)

	// Set up alice receiver
	go receiver(s.alice, aliceKey, &bobKey.PublicKey, bobInstallationID, aliceReceiverErrChan, aliceChan)

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
	aliceMessage, err := s.alice.EncryptPayload(&bobKey.PublicKey, aliceKey, aliceText)
	s.Require().NoError(err)

	// Bob receives the message, and returns a bundlenotfound error
	_, err = s.bob.DecryptPayload(bobKey, &aliceKey.PublicKey, aliceInstallationID, aliceMessage, defaultMessageID)
	s.Require().Error(err)
	s.Equal(ErrSessionNotFound, err)
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
	aliceMessage, err := s.alice.EncryptPayload(&bobKey.PublicKey, aliceKey, []byte("does not matter"))
	s.Require().NoError(err)

	// Bob receives the message, and returns a bundlenotfound error
	_, err = s.bob.DecryptPayload(bobKey, &aliceKey.PublicKey, aliceInstallationID, aliceMessage, defaultMessageID)
	s.Require().Error(err)
	s.Equal(ErrDeviceNotFound, err)
}

// A new bundle has been received
func (s *EncryptionServiceTestSuite) TestRefreshedBundle() {

	config := DefaultEncryptionServiceConfig("none")
	// Set up refresh interval to "always"
	config.BundleRefreshInterval = 1000

	s.initDatabases(&config)

	bobKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	aliceKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	// Create bundles
	bobBundle1, err := s.bob.CreateBundle(bobKey)
	s.Require().NoError(err)
	s.Require().Equal(uint32(1), bobBundle1.GetSignedPreKeys()[bobInstallationID].GetVersion())

	// Sleep the required time so that bundle is refreshed
	time.Sleep(time.Duration(config.BundleRefreshInterval) * time.Millisecond)

	// Create bundles
	bobBundle2, err := s.bob.CreateBundle(bobKey)
	s.Require().NoError(err)
	s.Require().Equal(uint32(2), bobBundle2.GetSignedPreKeys()[bobInstallationID].GetVersion())

	// We add the first bob bundle
	_, err = s.alice.ProcessPublicBundle(aliceKey, bobBundle1)
	s.Require().NoError(err)

	// Alice sends a message
	encryptionResponse1, err := s.alice.EncryptPayload(&bobKey.PublicKey, aliceKey, []byte("anything"))
	s.Require().NoError(err)

	installationResponse1 := encryptionResponse1[bobInstallationID]
	s.Require().NotNil(installationResponse1)

	// This message is using bobBundle1

	x3dhHeader1 := installationResponse1.GetX3DHHeader()
	s.NotNil(x3dhHeader1)
	s.Equal(bobBundle1.GetSignedPreKeys()[bobInstallationID].GetSignedPreKey(), x3dhHeader1.GetId())

	// We add the second bob bundle
	_, err = s.alice.ProcessPublicBundle(aliceKey, bobBundle2)
	s.Require().NoError(err)

	// Alice sends a message
	encryptionResponse2, err := s.alice.EncryptPayload(&bobKey.PublicKey, aliceKey, []byte("anything"))
	s.Require().NoError(err)

	installationResponse2 := encryptionResponse2[bobInstallationID]
	s.Require().NotNil(installationResponse2)

	// This message is using bobBundle2

	x3dhHeader2 := installationResponse2.GetX3DHHeader()
	s.NotNil(x3dhHeader2)
	s.Equal(bobBundle2.GetSignedPreKeys()[bobInstallationID].GetSignedPreKey(), x3dhHeader2.GetId())

}

func (s *EncryptionServiceTestSuite) TestMessageConfirmation() {
	bobText1 := []byte("bob text 1")

	bobKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	aliceKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	// Create a bundle
	bobBundle, err := s.bob.CreateBundle(bobKey)
	s.Require().NoError(err)

	// We add bob bundle
	_, err = s.alice.ProcessPublicBundle(aliceKey, bobBundle)
	s.Require().NoError(err)

	// Create a bundle
	aliceBundle, err := s.alice.CreateBundle(aliceKey)
	s.Require().NoError(err)

	// We add alice bundle
	_, err = s.bob.ProcessPublicBundle(bobKey, aliceBundle)
	s.Require().NoError(err)

	// Bob sends a message
	bobMessage1, err := s.bob.EncryptPayload(&aliceKey.PublicKey, bobKey, bobText1)
	s.Require().NoError(err)
	bobMessage1ID := []byte("bob-message-1-id")

	// Alice receives the message once
	_, err = s.alice.DecryptPayload(aliceKey, &bobKey.PublicKey, bobInstallationID, bobMessage1, bobMessage1ID)
	s.Require().NoError(err)

	// Alice receives the message twice
	_, err = s.alice.DecryptPayload(aliceKey, &bobKey.PublicKey, bobInstallationID, bobMessage1, bobMessage1ID)
	s.Require().NoError(err)

	// Alice confirms the message
	err = s.alice.ConfirmMessagesProcessed([][]byte{bobMessage1ID})
	s.Require().NoError(err)

	// Alice decrypts it again, it should fail
	_, err = s.alice.DecryptPayload(aliceKey, &bobKey.PublicKey, bobInstallationID, bobMessage1, bobMessage1ID)
	s.Require().Equal(errors.New("can't skip current chain message keys: bad until: probably an out-of-order message that was deleted"), err)

	// Bob sends a message
	bobMessage2, err := s.bob.EncryptPayload(&aliceKey.PublicKey, bobKey, bobText1)
	s.Require().NoError(err)
	bobMessage2ID := []byte("bob-message-2-id")

	// Bob sends a message
	bobMessage3, err := s.bob.EncryptPayload(&aliceKey.PublicKey, bobKey, bobText1)
	s.Require().NoError(err)
	bobMessage3ID := []byte("bob-message-3-id")

	// Alice receives message 3 once
	_, err = s.alice.DecryptPayload(aliceKey, &bobKey.PublicKey, bobInstallationID, bobMessage3, bobMessage3ID)
	s.Require().NoError(err)

	// Alice receives message 3 twice
	_, err = s.alice.DecryptPayload(aliceKey, &bobKey.PublicKey, bobInstallationID, bobMessage3, bobMessage3ID)
	s.Require().NoError(err)

	// Alice receives message 2 once
	_, err = s.alice.DecryptPayload(aliceKey, &bobKey.PublicKey, bobInstallationID, bobMessage2, bobMessage2ID)
	s.Require().NoError(err)

	// Alice receives message 2 twice
	_, err = s.alice.DecryptPayload(aliceKey, &bobKey.PublicKey, bobInstallationID, bobMessage2, bobMessage2ID)
	s.Require().NoError(err)

	// Alice confirms the messages
	err = s.alice.ConfirmMessagesProcessed([][]byte{bobMessage2ID, bobMessage3ID})
	s.Require().NoError(err)

	// Alice decrypts it again, it should fail
	_, err = s.alice.DecryptPayload(aliceKey, &bobKey.PublicKey, bobInstallationID, bobMessage3, bobMessage3ID)
	s.Require().Equal(errors.New("can't skip current chain message keys: bad until: probably an out-of-order message that was deleted"), err)

	// Alice decrypts it again, it should fail
	_, err = s.alice.DecryptPayload(aliceKey, &bobKey.PublicKey, bobInstallationID, bobMessage2, bobMessage2ID)
	s.Require().Equal(errors.New("can't skip current chain message keys: bad until: probably an out-of-order message that was deleted"), err)
}
