package topic

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	appDB "github.com/status-im/status-go/services/shhext/chat/db"
	"github.com/stretchr/testify/suite"
)

func TestServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ServiceTestSuite))
}

type ServiceTestSuite struct {
	suite.Suite
	service *Service
	path    string
}

func (s *ServiceTestSuite) SetupTest() {
	dbFile, err := ioutil.TempFile(os.TempDir(), "topic")
	s.Require().NoError(err)
	s.path = dbFile.Name()

	db, err := appDB.Open(s.path, "", 0)

	s.Require().NoError(err)

	s.service = NewService(NewSQLLitePersistence(db))
}

func (s *ServiceTestSuite) TearDownTest() {
	os.Remove(s.path)
}

func (s *ServiceTestSuite) TestSingleInstallationID() {
	installationID1 := "1"
	installationID2 := "2"

	myKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	theirKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	// We receive a message from installationID1
	sharedKey1, err := s.service.Receive(myKey, &theirKey.PublicKey, installationID1)
	s.Require().NoError(err)
	s.Require().NotNil(sharedKey1, "it generates a shared key")

	// We want to send a message to installationID1
	sharedKey2, err := s.service.Send(&theirKey.PublicKey, []string{installationID1})
	s.Require().NoError(err)
	s.Require().NotNil(sharedKey2, "We can retrieve a shared secret")
	s.Require().Equal(sharedKey1, sharedKey2, "The shared secret is the same as the one stored")

	// We want to send a message to multiple installationIDs, one of which we haven't never communicated with
	sharedKey3, err := s.service.Send(&theirKey.PublicKey, []string{installationID1, installationID2})
	s.Require().NoError(err)
	s.Require().Nil(sharedKey3, "No shared key is returned")

	// We receive a message from installationID2
	sharedKey4, err := s.service.Receive(myKey, &theirKey.PublicKey, installationID2)
	s.Require().NoError(err)
	s.Require().NotNil(sharedKey4, "it generates a shared key")
	s.Require().Equal(sharedKey1, sharedKey4, "It generates the same key")

	// We want to send a message to installationID 1 & 2, both have been
	sharedKey5, err := s.service.Send(&theirKey.PublicKey, []string{installationID1, installationID2})
	s.Require().NoError(err)
	s.Require().NotNil(sharedKey5, "We can retrieve a shared secret")
	s.Require().Equal(sharedKey1, sharedKey5, "The shared secret is the same as the one stored")

}

func (s *ServiceTestSuite) TestAll() {
	installationID1 := "1"
	installationID2 := "2"

	myKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	theirKey1, err := crypto.GenerateKey()
	s.Require().NoError(err)

	theirKey2, err := crypto.GenerateKey()
	s.Require().NoError(err)

	// We receive a message from user 1
	sharedKey1, err := s.service.Receive(myKey, &theirKey1.PublicKey, installationID1)
	s.Require().NoError(err)
	s.Require().NotNil(sharedKey1, "it generates a shared key")

	// We receive a message from user 2
	sharedKey2, err := s.service.Receive(myKey, &theirKey2.PublicKey, installationID2)
	s.Require().NoError(err)
	s.Require().NotNil(sharedKey2, "it generates a shared key")

	// All the topics are there
	topics, err := s.service.All()
	s.Require().NoError(err)
	s.Require().Equal([][]byte{sharedKey1, sharedKey2}, topics)
}
