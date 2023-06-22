package pushnotificationserver

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/sqlite"
)

func TestSQLitePersistenceSuite(t *testing.T) {
	suite.Run(t, new(SQLitePersistenceSuite))
}

type SQLitePersistenceSuite struct {
	suite.Suite
	tmpFile     *os.File
	persistence Persistence
}

func (s *SQLitePersistenceSuite) SetupTest() {
	tmpFile, err := ioutil.TempFile("", "")
	s.Require().NoError(err)
	s.tmpFile = tmpFile

	database, err := sqlite.Open(s.tmpFile.Name(), "", sqlite.ReducedKDFIterationsNumber)
	s.Require().NoError(err)
	s.persistence = NewSQLitePersistence(database)
}

func (s *SQLitePersistenceSuite) TearDownTest() {
	_ = os.Remove(s.tmpFile.Name())
}

func (s *SQLitePersistenceSuite) TestSaveAndRetrieve() {
	key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	installationID := "54242d02-bb92-11ea-b3de-0242ac130004"

	registration := &protobuf.PushNotificationRegistration{
		InstallationId: installationID,
		Version:        5,
	}

	s.Require().NoError(s.persistence.SavePushNotificationRegistration(common.HashPublicKey(&key.PublicKey), registration))

	retrievedRegistration, err := s.persistence.GetPushNotificationRegistrationByPublicKeyAndInstallationID(common.HashPublicKey(&key.PublicKey), installationID)
	s.Require().NoError(err)

	s.Require().True(proto.Equal(registration, retrievedRegistration))
}

func (s *SQLitePersistenceSuite) TestSaveAndRetrieveIdentity() {
	retrievedKey, err := s.persistence.GetIdentity()
	s.Require().NoError(err)
	s.Require().Nil(retrievedKey)

	key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	s.Require().NoError(s.persistence.SaveIdentity(key))

	retrievedKey, err = s.persistence.GetIdentity()
	s.Require().NoError(err)

	s.Require().Equal(key, retrievedKey)
}

func (s *SQLitePersistenceSuite) TestSaveDifferentIdenities() {
	key1, err := crypto.GenerateKey()
	s.Require().NoError(err)
	key2, err := crypto.GenerateKey()
	s.Require().NoError(err)

	// First one should be successul, second should fail
	s.Require().NoError(s.persistence.SaveIdentity(key1))
	s.Require().Error(s.persistence.SaveIdentity(key2))
}

func (s *SQLitePersistenceSuite) TestExists() {
	messageID1 := []byte("1")
	messageID2 := []byte("2")

	result, err := s.persistence.PushNotificationExists(messageID1)
	s.Require().NoError(err)
	s.Require().False(result)

	result, err = s.persistence.PushNotificationExists(messageID1)
	s.Require().NoError(err)

	s.Require()
	s.Require().True(result)

	result, err = s.persistence.PushNotificationExists(messageID2)
	s.Require().NoError(err)
	s.Require().False(result)
}
