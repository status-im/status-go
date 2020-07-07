package push_notification_client

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/sqlite"
)

func TestSQLitePersistenceSuite(t *testing.T) {
	suite.Run(t, new(SQLitePersistenceSuite))
}

type SQLitePersistenceSuite struct {
	suite.Suite
	tmpFile     *os.File
	persistence *Persistence
}

func (s *SQLitePersistenceSuite) SetupTest() {
	tmpFile, err := ioutil.TempFile("", "")
	s.Require().NoError(err)
	s.tmpFile = tmpFile

	database, err := sqlite.Open(s.tmpFile.Name(), "")
	s.Require().NoError(err)
	s.persistence = NewPersistence(database)
}

func (s *SQLitePersistenceSuite) TearDownTest() {
	_ = os.Remove(s.tmpFile.Name())
}

func (s *SQLitePersistenceSuite) TestSaveAndRetrieveServer() {
	key, err := crypto.GenerateKey()
	s.Require().NoError(err)

	server := &PushNotificationServer{
		publicKey:    &key.PublicKey,
		registered:   true,
		registeredAt: 1,
	}

	s.Require().NoError(s.persistence.UpsertServer(server))

	retrievedServers, err := s.persistence.GetServers()
	s.Require().NoError(err)

	s.Require().Len(retrievedServers, 1)
	s.Require().True(retrievedServers[0].registered)
	s.Require().Equal(int64(1), retrievedServers[0].registeredAt)
	s.Require().True(common.IsPubKeyEqual(retrievedServers[0].publicKey, &key.PublicKey))

	server.registered = false
	server.registeredAt = 2

	s.Require().NoError(s.persistence.UpsertServer(server))

	retrievedServers, err = s.persistence.GetServers()
	s.Require().NoError(err)

	s.Require().Len(retrievedServers, 1)
	s.Require().False(retrievedServers[0].registered)
	s.Require().Equal(int64(2), retrievedServers[0].registeredAt)
	s.Require().True(common.IsPubKeyEqual(retrievedServers[0].publicKey, &key.PublicKey))
}
