package chat

import (
	"database/sql"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/suite"
	"os"
	"testing"
)

const (
	dbPath = "/tmp/status-key-store.db"
	key    = "blahblahblah"
)

func TestSqlLitePersistenceTestSuite(t *testing.T) {
	suite.Run(t, new(SqlLitePersistenceTestSuite))
}

type SqlLitePersistenceTestSuite struct {
	suite.Suite
	db      *sql.DB
	service PersistenceServiceInterface
}

func (s *SqlLitePersistenceTestSuite) SetupTest() {
	os.Remove(dbPath)

	p, err := NewSqlLitePersistence(dbPath, key)
	if err != nil {
		panic(err)
	}
	s.service = p
}

func (s *SqlLitePersistenceTestSuite) TestPrivateBundle() {
	key, err := crypto.GenerateKey()
	s.NoError(err)

	actualBundle, err := s.service.GetPrivateBundle([]byte("non-existing"))
	s.NoErrorf(err, "It does not return an error if the bundle is not there")
	s.Nil(actualBundle)

	anyPrivateBundle, err := s.service.GetAnyPrivateBundle()
	s.NoError(err)
	s.Nil(anyPrivateBundle)

	bundle, err := NewBundleContainer(key)
	s.NoError(err)

	err = s.service.AddPrivateBundle(bundle)
	s.NoError(err)

	bundleID := bundle.GetBundle().GetSignedPreKey()

	actualBundle, err = s.service.GetPrivateBundle(bundleID)
	s.NoError(err)

	s.Equalf(true, proto.Equal(bundle, actualBundle), "It returns the same bundle")

	anyPrivateBundle, err = s.service.GetAnyPrivateBundle()
	s.NoError(err)

	s.Equalf(true, proto.Equal(bundle.GetBundle(), anyPrivateBundle), "It returns the same bundle")

}

func (s *SqlLitePersistenceTestSuite) TestPublicBundle() {
	key, err := crypto.GenerateKey()
	s.NoError(err)

	actualBundle, err := s.service.GetPublicBundle(&key.PublicKey)
	s.NoErrorf(err, "It does not return an error if the bundle is not there")
	s.Nil(actualBundle)

	bundleContainer, err := NewBundleContainer(key)
	bundle := bundleContainer.GetBundle()
	s.NoError(err)

	err = s.service.AddPublicBundle(bundle)
	s.NoError(err)

	actualBundle, err = s.service.GetPublicBundle(&key.PublicKey)
	s.NoError(err)

	s.Equalf(true, proto.Equal(bundle, actualBundle), "It returns the same bundle")
}

func (s *SqlLitePersistenceTestSuite) TestSymmetricKey() {
	identityKey, err := crypto.GenerateKey()
	s.NoError(err)

	ephemeralKey, err := crypto.GenerateKey()
	s.NoError(err)
	symKey := []byte("hello")

	actualKey, err := s.service.GetSymmetricKey(&identityKey.PublicKey, &ephemeralKey.PublicKey)
	s.NoErrorf(err, "It does not return an error if the key is not there")
	s.Nil(actualKey)

	actualKey, actualEphemeralKey, err := s.service.GetAnySymmetricKey(&identityKey.PublicKey)
	s.NoError(err)
	s.Nil(actualKey)
	s.Nil(actualEphemeralKey)

	err = s.service.AddSymmetricKey(&identityKey.PublicKey, &ephemeralKey.PublicKey, symKey)
	s.NoError(err)

	actualKey, err = s.service.GetSymmetricKey(&identityKey.PublicKey, &ephemeralKey.PublicKey)
	s.NoError(err)

	s.Equalf(symKey, actualKey, "It returns the same key")

	actualKey, actualEphemeralKey, err = s.service.GetAnySymmetricKey(&identityKey.PublicKey)
	s.NoError(err)

	s.Equalf(symKey, actualKey, "It returns the same key")
	s.Equalf(ephemeralKey.PublicKey, *actualEphemeralKey, "It returns the same ephemeral key")
}
