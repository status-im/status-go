package chat

import (
	"fmt"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/suite"
)

var cleartext = []byte("hello")

func TestEncryptionServiceTestSuite(t *testing.T) {
	suite.Run(t, new(EncryptionServiceTestSuite))
}

type EncryptionServiceTestSuite struct {
	suite.Suite
	alice *EncryptionService
	bob   *EncryptionService
}

func (s *EncryptionServiceTestSuite) SetupTest() {
	aliceDBPath := "/tmp/alice.db"
	aliceDBKey := "alice"
	bobDBPath := "/tmp/bob.db"
	bobDBKey := "bob"

	os.Remove(aliceDBPath)
	os.Remove(bobDBPath)

	alicePersistence, err := NewSqlLitePersistence(aliceDBPath, aliceDBKey)
	if err != nil {
		panic(err)
	}

	bobPersistence, err := NewSqlLitePersistence(bobDBPath, bobDBKey)
	if err != nil {
		panic(err)
	}

	s.alice = NewEncryptionService(alicePersistence)
	s.bob = NewEncryptionService(bobPersistence)
}

// Alice sends Bob an encrypted message with DH using an ephemeral key
// and Bob's identity key.
// Bob is able to decrypt it.
// Alice does not re-use the symmetric key
func (s *EncryptionServiceTestSuite) TestEncryptPayloadNoBundle() {
	bobKey, err := crypto.GenerateKey()
	s.NoError(err)
	aliceKey, err := crypto.GenerateKey()
	s.NoError(err)

	encryptionResponse1, err := s.alice.EncryptPayload(&bobKey.PublicKey, aliceKey, cleartext)
	s.NoError(err)
	fmt.Printf("%x\n", encryptionResponse1)

	cyphertext1 := encryptionResponse1.Payload
	ephemeralKey1 := encryptionResponse1.GetX3DHHeader().GetDhKey()

	s.NotNil(ephemeralKey1, "It generates an ephemeral key for DH exchange")
	s.NotNil(cyphertext1, "It generates an encrypted payload")
	s.NotEqualf(cyphertext1, cleartext, "It encrypts the payload correctly")

	// On the receiver side, we should be able to decrypt using our private key and the ephemeral just sent
	decryptedPayload1, err := s.bob.DecryptPayload(bobKey, &aliceKey.PublicKey, encryptionResponse1)
	s.NoError(err)
	s.Equalf(cleartext, decryptedPayload1, "It correctly decrypts the payload using DH")

	// The next message will not be re-using the same key
	encryptionResponse2, err := s.alice.EncryptPayload(&bobKey.PublicKey, aliceKey, cleartext)
	s.NoError(err)

	cyphertext2 := encryptionResponse2.GetPayload()
	ephemeralKey2 := encryptionResponse2.GetX3DHHeader().GetDhKey()

	s.NotEqual(cyphertext1, cyphertext2, "It does not re-use the symmetric key")
	s.NotEqual(ephemeralKey1, ephemeralKey2, "It does not re-use the ephemeral key")

	decryptedPayload2, err := s.bob.DecryptPayload(bobKey, &aliceKey.PublicKey, encryptionResponse2)
	s.NoError(err)

	s.Equalf(cleartext, decryptedPayload2, "It correctly decrypts the payload using DH")
}

// Alice has Bob's bundle
// Alice sends Bob an encrypted message with X3DH using an ephemeral key
// and Bob's bundle.
func (s *EncryptionServiceTestSuite) TestEncryptPayloadBundle() {
	bobKey, err := crypto.GenerateKey()
	s.NoError(err)
	aliceKey, err := crypto.GenerateKey()
	s.NoError(err)

	// Create a couple of bundles
	bobBundle1, err := s.bob.CreateBundle(bobKey)
	s.NoError(err)
	bobBundle2, err := s.bob.CreateBundle(bobKey)
	s.NoError(err)

	s.NotEqualf(bobBundle1, bobBundle2, "It creates different bundles")

	// We add bob bundle
	err = s.alice.ProcessPublicBundle(bobBundle2)
	s.NoError(err)

	// We send a message using the bundle
	encryptionResponse1, err := s.alice.EncryptPayload(&bobKey.PublicKey, aliceKey, cleartext)
	s.NoError(err)

	cyphertext1 := encryptionResponse1.GetPayload()
	ephemeralKey1 := encryptionResponse1.GetX3DHHeader().GetBundleKey()

	s.NoError(err)
	s.NotNil(cyphertext1, "It generates an encrypted payload")
	s.NotEqualf(cyphertext1, cleartext, "It encrypts the payload correctly")
	s.NotNil(ephemeralKey1, "It generates an ephemeral key")

	// Bob is able to decrypt it using the bundle
	bundleID := bobBundle2.GetSignedPreKey()

	s.Equalf(encryptionResponse1.GetX3DHHeader().GetBundleId(), bundleID, "It sets the bundle id")

	decryptedPayload1, err := s.bob.DecryptPayload(bobKey, &aliceKey.PublicKey, encryptionResponse1)
	s.NoError(err)
	s.Equalf(cleartext, decryptedPayload1, "It correctly decrypts the payload using X3DH")

	// Alice sends another message, this time she will use the same key as generated previously
	encryptionResponse2, err := s.alice.EncryptPayload(&bobKey.PublicKey, aliceKey, cleartext)
	s.NoError(err)

	cyphertext2 := encryptionResponse2.GetPayload()
	ephemeralKey2 := encryptionResponse2.GetX3DHHeader().GetSymKey()

	s.NoError(err)
	s.NotNil(cyphertext2, "It generates an encrypted payload")
	s.NotEqualf(cyphertext2, cleartext, "It encrypts the payload correctly")
	s.Equal(ephemeralKey1, ephemeralKey2, "It returns the same ephemeral key")

	// Bob this time should be able to decrypt it with a symmetric key
	decryptedPayload2, err := s.bob.DecryptPayload(bobKey, &aliceKey.PublicKey, encryptionResponse2)
	s.NoError(err)
	s.Equalf(cleartext, decryptedPayload2, "It correctly decrypts the payload using a symmetric key")
}

// Alice has Bob's bundle
// Alice sends Bob an encrypted message with X3DH using an ephemeral key
// and Bob's bundle.
// Alice sends another message. This message should be using a DR
// and should include the initial x3dh message

// Alice has Bob's bundle
// Alice sends Bob an encrypted message with X3DH using an ephemeral key
// and Bob's bundle.
// Bob's reply with a DR message
// Alice sends another message. This message should be using a DR
// and should not include the initial x3dh message
