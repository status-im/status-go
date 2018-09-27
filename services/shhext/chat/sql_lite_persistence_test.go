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
	service PersistenceService
}

func (s *SQLLitePersistenceTestSuite) SetupTest() {
	os.Remove(dbPath)

	p, err := NewSQLLitePersistence(dbPath, key)
	s.Require().NoError(err)
	s.service = p
}

func (s *SQLLitePersistenceTestSuite) TestMultipleInit() {
	os.Remove(dbPath)

	_, err := NewSQLLitePersistence(dbPath, key)
	s.Require().NoError(err)

	_, err = NewSQLLitePersistence(dbPath, key)
	s.Require().NoError(err)
}

func (s *SQLLitePersistenceTestSuite) TestPrivateBundle() {
	installationID := "1"

	key, err := crypto.GenerateKey()
	s.Require().NoError(err)

	actualKey, err := s.service.GetPrivateKeyBundle([]byte("non-existing"))
	s.Require().NoError(err, "It does not return an error if the bundle is not there")
	s.Nil(actualKey)

	anyPrivateBundle, err := s.service.GetAnyPrivateBundle([]byte("non-existing-id"))
	s.Require().NoError(err)
	s.Nil(anyPrivateBundle)

	bundle, err := NewBundleContainer(key, installationID)
	s.Require().NoError(err)

	err = s.service.AddPrivateBundle(bundle)
	s.Require().NoError(err)

	bundleID := bundle.GetBundle().GetSignedPreKeys()[installationID].GetSignedPreKey()

	actualKey, err = s.service.GetPrivateKeyBundle(bundleID)
	s.Require().NoError(err)
	s.Equal(bundle.GetPrivateSignedPreKey(), actualKey, "It returns the same key")

	identity := crypto.CompressPubkey(&key.PublicKey)
	anyPrivateBundle, err = s.service.GetAnyPrivateBundle(identity)
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

	bundleContainer, err := NewBundleContainer(key, "1")
	s.Require().NoError(err)

	bundle := bundleContainer.GetBundle()
	err = s.service.AddPublicBundle(bundle)
	s.Require().NoError(err)

	actualBundle, err = s.service.GetPublicBundle(&key.PublicKey)
	s.Require().NoError(err)
	s.Equal(bundle.GetIdentity(), actualBundle.GetIdentity(), "It sets the right identity")
	s.Equal(bundle.GetSignedPreKeys(), actualBundle.GetSignedPreKeys(), "It sets the right prekeys")
}

func (s *SQLLitePersistenceTestSuite) TestMultiplePublicBundle() {
	key, err := crypto.GenerateKey()
	s.Require().NoError(err)

	actualBundle, err := s.service.GetPublicBundle(&key.PublicKey)
	s.Require().NoError(err, "It does not return an error if the bundle is not there")
	s.Nil(actualBundle)

	bundleContainer, err := NewBundleContainer(key, "1")
	s.Require().NoError(err)

	bundle := bundleContainer.GetBundle()
	err = s.service.AddPublicBundle(bundle)
	s.Require().NoError(err)

	// Adding it again does not throw an error
	err = s.service.AddPublicBundle(bundle)
	s.Require().NoError(err)

	// Adding a different bundle
	bundleContainer, err = NewBundleContainer(key, "1")
	s.Require().NoError(err)

	bundle = bundleContainer.GetBundle()
	err = s.service.AddPublicBundle(bundle)
	s.Require().NoError(err)

	// Returns the most recent bundle
	actualBundle, err = s.service.GetPublicBundle(&key.PublicKey)
	s.Require().NoError(err)

	s.Equal(bundle.GetIdentity(), actualBundle.GetIdentity(), "It sets the identity")
	s.Equal(bundle.GetSignedPreKeys(), actualBundle.GetSignedPreKeys(), "It sets the signed pre keys")

}

func (s *SQLLitePersistenceTestSuite) TestRatchetInfoPrivateBundle() {
	key, err := crypto.GenerateKey()
	s.Require().NoError(err)

	// Add a private bundle
	bundle, err := NewBundleContainer(key, "2")
	s.Require().NoError(err)

	err = s.service.AddPrivateBundle(bundle)
	s.Require().NoError(err)

	err = s.service.AddRatchetInfo(
		[]byte("symmetric-key"),
		[]byte("their-public-key"),
		bundle.GetBundle().GetSignedPreKeys()["2"].GetSignedPreKey(),
		[]byte("ephemeral-public-key"),
		"1",
	)
	s.Require().NoError(err)

	ratchetInfo, err := s.service.GetRatchetInfo(bundle.GetBundle().GetSignedPreKeys()["2"].GetSignedPreKey(), []byte("their-public-key"))

	s.Require().NoError(err)
	s.NotNil(ratchetInfo.ID, "It adds an id")
	s.Equal(ratchetInfo.PrivateKey, bundle.GetPrivateSignedPreKey(), "It returns the private key")
	s.Equal(ratchetInfo.Sk, []byte("symmetric-key"), "It returns the symmetric key")
	s.Equal(ratchetInfo.Identity, []byte("their-public-key"), "It returns the identity of the contact")
	s.Equal(ratchetInfo.PublicKey, bundle.GetBundle().GetSignedPreKeys()["2"].GetSignedPreKey(), "It  returns the public key of the bundle")
	s.Equal(bundle.GetBundle().GetSignedPreKeys()["2"].GetSignedPreKey(), ratchetInfo.BundleID, "It returns the bundle id")
	s.Equal([]byte("ephemeral-public-key"), ratchetInfo.EphemeralKey, "It returns the ratchet ephemeral key")
	s.Equal("1", ratchetInfo.InstallationID, "It returns the right installation id")
}

func (s *SQLLitePersistenceTestSuite) TestRatchetInfoPublicBundle() {
	installationID := "1"
	theirPublicKey := []byte("their-public-key")
	key, err := crypto.GenerateKey()
	s.Require().NoError(err)

	// Add a private bundle
	bundle, err := NewBundleContainer(key, installationID)
	s.Require().NoError(err)

	err = s.service.AddPublicBundle(bundle.GetBundle())
	s.Require().NoError(err)

	signedPreKey := bundle.GetBundle().GetSignedPreKeys()[installationID].GetSignedPreKey()

	err = s.service.AddRatchetInfo(
		[]byte("symmetric-key"),
		theirPublicKey,
		signedPreKey,
		[]byte("public-ephemeral-key"),
		installationID,
	)
	s.Require().NoError(err)

	ratchetInfo, err := s.service.GetRatchetInfo(signedPreKey, theirPublicKey)

	s.Require().NoError(err)
	s.Require().NotNil(ratchetInfo, "It returns the ratchet info")

	s.NotNil(ratchetInfo.ID, "It adds an id")
	s.Nil(ratchetInfo.PrivateKey, "It does not return the private key")
	s.Equal(ratchetInfo.Sk, []byte("symmetric-key"), "It returns the symmetric key")
	s.Equal(ratchetInfo.Identity, theirPublicKey, "It returns the identity of the contact")
	s.Equal(ratchetInfo.PublicKey, signedPreKey, "It  returns the public key of the bundle")
	s.Equal(installationID, ratchetInfo.InstallationID, "It returns the right installationID")
	s.Nilf(ratchetInfo.PrivateKey, "It does not return the private key")

	ratchetInfo, err = s.service.GetAnyRatchetInfo(theirPublicKey, installationID)
	s.Require().NoError(err)
	s.Require().NotNil(ratchetInfo, "It returns the ratchet info")
	s.NotNil(ratchetInfo.ID, "It adds an id")
	s.Nil(ratchetInfo.PrivateKey, "It does not return the private key")
	s.Equal(ratchetInfo.Sk, []byte("symmetric-key"), "It returns the symmetric key")
	s.Equal(ratchetInfo.Identity, theirPublicKey, "It returns the identity of the contact")
	s.Equal(ratchetInfo.PublicKey, signedPreKey, "It  returns the public key of the bundle")
	s.Equal(signedPreKey, ratchetInfo.BundleID, "It returns the bundle id")
	s.Equal(installationID, ratchetInfo.InstallationID, "It saves the right installation ID")
}

func (s *SQLLitePersistenceTestSuite) TestRatchetInfoNoBundle() {
	err := s.service.AddRatchetInfo(
		[]byte("symmetric-key"),
		[]byte("their-public-key"),
		[]byte("non-existing-bundle"),
		[]byte("non-existing-ephemeral-key"),
		"none",
	)

	s.Error(err, "It returns an error")

	_, err = s.service.GetRatchetInfo([]byte("non-existing-bundle"), []byte("their-public-key"))
	s.Require().NoError(err)

	ratchetInfo, err := s.service.GetAnyRatchetInfo([]byte("their-public-key"), "4")
	s.Require().NoError(err)
	s.Nil(ratchetInfo, "It returns nil when no bundle is there")
}

// TODO: Add test for MarkBundleExpired
// TODO: Add test for AddPublicBundle checking that it expires previous bundles
