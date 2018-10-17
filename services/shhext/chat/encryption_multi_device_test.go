package chat

import (
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/suite"
	"os"
	"sort"
	"testing"
)

func TestEncryptionServiceMultiDeviceSuite(t *testing.T) {
	suite.Run(t, new(EncryptionServiceMultiDeviceSuite))
}

type EncryptionServiceMultiDeviceSuite struct {
	suite.Suite
	alice1 *EncryptionService
	bob1   *EncryptionService
	alice2 *EncryptionService
	bob2   *EncryptionService
}

func (s *EncryptionServiceMultiDeviceSuite) SetupTest() {
	const (
		aliceDBPath1 = "/tmp/alice1.db"
		aliceDBKey1  = "alice1"
		aliceDBPath2 = "/tmp/alice2.db"
		aliceDBKey2  = "alice2"
		bobDBPath1   = "/tmp/bob1.db"
		bobDBKey1    = "bob1"
		bobDBPath2   = "/tmp/bob2.db"
		bobDBKey2    = "bob2"
	)

	os.Remove(aliceDBPath1)
	os.Remove(bobDBPath1)
	os.Remove(aliceDBPath2)
	os.Remove(bobDBPath2)

	alicePersistence1, err := NewSQLLitePersistence(aliceDBPath1, aliceDBKey1)
	if err != nil {
		panic(err)
	}

	alicePersistence2, err := NewSQLLitePersistence(aliceDBPath2, aliceDBKey2)
	if err != nil {
		panic(err)
	}

	bobPersistence1, err := NewSQLLitePersistence(bobDBPath1, bobDBKey1)
	if err != nil {
		panic(err)
	}

	bobPersistence2, err := NewSQLLitePersistence(bobDBPath2, bobDBKey2)
	if err != nil {
		panic(err)
	}

	s.alice1 = NewEncryptionService(alicePersistence1, "alice1")
	s.bob1 = NewEncryptionService(bobPersistence1, "bob1")

	s.alice2 = NewEncryptionService(alicePersistence2, "alice2")
	s.bob2 = NewEncryptionService(bobPersistence2, "bob2")

}

func (s *EncryptionServiceMultiDeviceSuite) TestProcessPublicBundle() {
	aliceKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	alice1Bundle, err := s.alice1.CreateBundle(aliceKey)
	s.Require().NoError(err)

	alice1Identity, err := ExtractIdentity(alice1Bundle)
	s.Require().NoError(err)

	alice2Bundle, err := s.alice2.CreateBundle(aliceKey)
	s.Require().NoError(err)

	alice2Identity, err := ExtractIdentity(alice2Bundle)
	s.Require().NoError(err)

	response, err := s.alice1.ProcessPublicBundle(aliceKey, alice2Bundle)
	s.Require().NoError(err)
	s.Require().Equal(IdentityAndIDPair{alice2Identity, "alice2"}, response[0])

	alice1MergedBundle1, err := s.alice1.CreateBundle(aliceKey)
	s.Require().NoError(err)

	s.Require().NotNil(alice1MergedBundle1.GetSignedPreKeys()["alice1"])
	s.Require().NotNil(alice1MergedBundle1.GetSignedPreKeys()["alice2"])

	response, err = s.alice1.ProcessPublicBundle(aliceKey, alice1MergedBundle1)
	s.Require().NoError(err)
	sort.Slice(response, func(i, j int) bool {
		return response[i][1] < response[j][1]
	})
	s.Require().Equal(IdentityAndIDPair{alice1Identity, "alice1"}, response[0])
	s.Require().Equal(IdentityAndIDPair{alice2Identity, "alice2"}, response[1])
}

func (s *EncryptionServiceMultiDeviceSuite) TestProcessPublicBundleOutOfOrder() {
	aliceKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	// Alice1 creates a bundle
	alice1Bundle, err := s.alice1.CreateBundle(aliceKey)
	s.Require().NoError(err)

	// Alice2 Receives the bundle
	_, err = s.alice2.ProcessPublicBundle(aliceKey, alice1Bundle)
	s.Require().NoError(err)

	// Alice2 Creates a Bundle
	_, err = s.alice2.CreateBundle(aliceKey)
	s.Require().NoError(err)

	// It should contain both bundles
	alice1MergedBundle1, err := s.alice2.CreateBundle(aliceKey)
	s.Require().NoError(err)

	s.Require().NotNil(alice1MergedBundle1.GetSignedPreKeys()["alice1"])
	s.Require().NotNil(alice1MergedBundle1.GetSignedPreKeys()["alice2"])
}
