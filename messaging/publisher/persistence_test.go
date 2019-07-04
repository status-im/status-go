package publisher

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/status-im/status-go/messaging/chat"
	"github.com/stretchr/testify/suite"
)

func TestPersistenceTestSuite(t *testing.T) {
	suite.Run(t, new(PersistenceTestSuite))
}

type PersistenceTestSuite struct {
	suite.Suite
	persistence Persistence
}

func (s *PersistenceTestSuite) SetupTest() {
	dir, err := ioutil.TempDir("", "publisher-persistence-test")
	s.Require().NoError(err)

	p, err := chat.NewSQLLitePersistence(filepath.Join(dir, "db1.sql"), "pass")
	s.Require().NoError(err)

	s.persistence = NewSQLLitePersistence(p.DB)
}

func (s *PersistenceTestSuite) TestLastAcked() {
	identity := []byte("identity")
	// Nothing in the database
	lastAcked1, err := s.persistence.GetLastAcked(identity)
	s.Require().NoError(err)
	s.Require().Equal(int64(0), lastAcked1)

	err = s.persistence.SetLastAcked(identity, 3)
	s.Require().NoError(err)

	lastAcked2, err := s.persistence.GetLastAcked(identity)
	s.Require().NoError(err)
	s.Require().Equal(int64(3), lastAcked2)
}

func (s *PersistenceTestSuite) TestLastPublished() {
	lastPublished1, err := s.persistence.GetLastPublished()
	s.Require().NoError(err)
	s.Require().Equal(int64(0), lastPublished1)

	err = s.persistence.SetLastPublished(3)
	s.Require().NoError(err)

	lastPublished2, err := s.persistence.GetLastPublished()
	s.Require().NoError(err)
	s.Require().Equal(int64(3), lastPublished2)
}
