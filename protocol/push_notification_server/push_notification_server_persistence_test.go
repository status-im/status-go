package push_notification_server

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/sqlite"
)

//tmpFile, err := ioutil.TempFile("", "")

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

	database, err := sqlite.Open(s.tmpFile.Name(), "")
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

	s.Require().NoError(s.persistence.SavePushNotificationRegistration(hashPublicKey(&key.PublicKey), registration))

	retrievedRegistration, err := s.persistence.GetPushNotificationRegistrationByPublicKeyAndInstallationID(hashPublicKey(&key.PublicKey), installationID)
	s.Require().NoError(err)

	s.Require().True(proto.Equal(registration, retrievedRegistration))
}
