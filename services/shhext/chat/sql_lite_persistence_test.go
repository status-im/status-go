package chat

import (
	"database/sql"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/suite"
)

const (
	dbPath = "/tmp/status-key-store.db"
	key    = "blahblahblah"
)

func TestSQLLitePersistenceTestSuite(t *testing.T) {
	suite.Run(t, new(SQLLitePersistenceTestSuite))
}

type SQLLitePersistenceTestSuite struct {
	suite.Suite
	// nolint: structcheck, megacheck
	db      *sql.DB
	service PersistenceServiceInterface
}

func (s *SQLLitePersistenceTestSuite) SetupTest() {
	os.Remove(dbPath)

	p, err := NewSQLLitePersistence(dbPath, key)
	s.Require().NoError(err)
	s.service = p
}

func (s *SQLLitePersistenceTestSuite) TestPrivateBundle() {
	key, err := crypto.GenerateKey()
	s.Require().NoError(err)

	actualBundle, err := s.service.GetPrivateBundle([]byte("non-existing"))
	s.Require().NoError(err, "It does not return an error if the bundle is not there")
	s.Nil(actualBundle)

	anyPrivateBundle, err := s.service.GetAnyPrivateBundle()
	s.Require().NoError(err)
	s.Nil(anyPrivateBundle)

	bundle, err := NewBundleContainer(key)
	s.Require().NoError(err)

	err = s.service.AddPrivateBundle(bundle)
	s.Require().NoError(err)

	bundleID := bundle.GetBundle().GetSignedPreKey()

	actualBundle, err = s.service.GetPrivateBundle(bundleID)
	s.Require().NoError(err)
	s.True(proto.Equal(bundle.GetBundle(), actualBundle.GetBundle()), "It returns the same bundle")

	anyPrivateBundle, err = s.service.GetAnyPrivateBundle()
	s.Require().NoError(err)
	s.NotNil(anyPrivateBundle)
	s.True(proto.Equal(bundle.GetBundle(), anyPrivateBundle.GetBundle()), "It returns the same bundle")
}

func (s *SQLLitePersistenceTestSuite) TestPublicBundle() {
	key, err := crypto.GenerateKey()
	s.Require().NoError(err)

	actualBundle, err := s.service.GetPublicBundle(&key.PublicKey)
	s.Require().NoError(err, "It does not return an error if the bundle is not there")
	s.Nil(actualBundle)

	bundleContainer, err := NewBundleContainer(key)
	s.Require().NoError(err)

	bundle := bundleContainer.GetBundle()
	err = s.service.AddPublicBundle(bundle)
	s.Require().NoError(err)

	actualBundle, err = s.service.GetPublicBundle(&key.PublicKey)
	s.Require().NoError(err)
	s.True(proto.Equal(bundle, actualBundle), "It returns the same bundle")
}

func (s *SQLLitePersistenceTestSuite) TestMultiplePublicBundle() {
	key, err := crypto.GenerateKey()
	s.Require().NoError(err)

	actualBundle, err := s.service.GetPublicBundle(&key.PublicKey)
	s.Require().NoError(err, "It does not return an error if the bundle is not there")
	s.Nil(actualBundle)

	bundleContainer, err := NewBundleContainer(key)
	s.Require().NoError(err)

	bundle := bundleContainer.GetBundle()
	err = s.service.AddPublicBundle(bundle)
	s.Require().NoError(err)

	// Adding it again does not throw an error
	err = s.service.AddPublicBundle(bundle)
	s.Require().NoError(err)

	// Adding a different bundle
	bundleContainer, err = NewBundleContainer(key)
	s.Require().NoError(err)

	bundle = bundleContainer.GetBundle()
	err = s.service.AddPublicBundle(bundle)
	s.Require().NoError(err)

	// Returns the most recent bundle
	actualBundle, err = s.service.GetPublicBundle(&key.PublicKey)
	s.Require().NoError(err)

	s.Equal(bundle, actualBundle)

}

func (s *SQLLitePersistenceTestSuite) TestRatchetInfoPrivateBundle() {
	key, err := crypto.GenerateKey()
	s.Require().NoError(err)

	// Add a private bundle
	bundle, err := NewBundleContainer(key)
	s.Require().NoError(err)

	err = s.service.AddPrivateBundle(bundle)
	s.Require().NoError(err)

	err = s.service.AddRatchetInfo(
		[]byte("symmetric-key"),
		[]byte("their-public-key"),
		bundle.GetBundle().GetSignedPreKey(),
		[]byte("ephemeral-public-key"),
	)
	s.Require().NoError(err)

	ratchetInfo, err := s.service.GetRatchetInfo(bundle.GetBundle().GetSignedPreKey(), []byte("their-public-key"))

	s.Require().NoError(err)
	s.NotNil(ratchetInfo.ID, "It adds an id")
	s.Equal(ratchetInfo.PrivateKey, bundle.GetPrivateSignedPreKey(), "It returns the private key")
	s.Equal(ratchetInfo.Sk, []byte("symmetric-key"), "It returns the symmetric key")
	s.Equal(ratchetInfo.Identity, []byte("their-public-key"), "It returns the identity of the contact")
	s.Equal(ratchetInfo.PublicKey, bundle.GetBundle().GetSignedPreKey(), "It  returns the public key of the bundle")
	s.Equal(bundle.GetBundle().GetSignedPreKey(), ratchetInfo.BundleID, "It returns the bundle id")
	s.Equal([]byte("ephemeral-public-key"), ratchetInfo.EphemeralKey, "It returns the ratchet ephemeral key")
}

func (s *SQLLitePersistenceTestSuite) TestRatchetInfoPublicBundle() {
	key, err := crypto.GenerateKey()
	s.Require().NoError(err)

	// Add a private bundle
	bundle, err := NewBundleContainer(key)
	s.Require().NoError(err)

	err = s.service.AddPublicBundle(bundle.GetBundle())
	s.Require().NoError(err)

	err = s.service.AddRatchetInfo(
		[]byte("symmetric-key"),
		[]byte("their-public-key"),
		bundle.GetBundle().GetSignedPreKey(),
		[]byte("public-ephemeral-key"),
	)
	s.Require().NoError(err)

	ratchetInfo, err := s.service.GetRatchetInfo(bundle.GetBundle().GetSignedPreKey(), []byte("their-public-key"))

	s.Require().NoError(err)
	s.Require().NotNil(ratchetInfo, "It returns the ratchet info")

	s.NotNil(ratchetInfo.ID, "It adds an id")
	s.Nil(ratchetInfo.PrivateKey, "It does not return the private key")
	s.Equal(ratchetInfo.Sk, []byte("symmetric-key"), "It returns the symmetric key")
	s.Equal(ratchetInfo.Identity, []byte("their-public-key"), "It returns the identity of the contact")
	s.Equal(ratchetInfo.PublicKey, bundle.GetBundle().GetSignedPreKey(), "It  returns the public key of the bundle")
	s.Nilf(ratchetInfo.PrivateKey, "It does not return the private key")

	ratchetInfo, err = s.service.GetAnyRatchetInfo([]byte("their-public-key"))
	s.Require().NoError(err)
	s.Require().NotNil(ratchetInfo, "It returns the ratchet info")
	s.NotNil(ratchetInfo.ID, "It adds an id")
	s.Nil(ratchetInfo.PrivateKey, "It does not return the private key")
	s.Equal(ratchetInfo.Sk, []byte("symmetric-key"), "It returns the symmetric key")
	s.Equal(ratchetInfo.Identity, []byte("their-public-key"), "It returns the identity of the contact")
	s.Equal(ratchetInfo.PublicKey, bundle.GetBundle().GetSignedPreKey(), "It  returns the public key of the bundle")
	s.Equal(bundle.GetBundle().GetSignedPreKey(), ratchetInfo.BundleID, "It returns the bundle id")
}

func (s *SQLLitePersistenceTestSuite) TestRatchetInfoNoBundle() {
	err := s.service.AddRatchetInfo(
		[]byte("symmetric-key"),
		[]byte("their-public-key"),
		[]byte("non-existing-bundle"),
		[]byte("non-existing-ephemeral-key"),
	)

	s.Error(err, "It returns an error")

	_, err = s.service.GetRatchetInfo([]byte("non-existing-bundle"), []byte("their-public-key"))
	s.Require().NoError(err)

	ratchetInfo, err := s.service.GetAnyRatchetInfo([]byte("their-public-key"))
	s.Require().NoError(err)
	s.Nil(ratchetInfo, "It returns nil when no bundle is there")
}

// TODO: Add test for MarkBundleExpired
// TODO: Add test for AddPublicBundle checking that it expires previous bundles
