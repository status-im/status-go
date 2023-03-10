package collectibles

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/appdatabase"
	protoSqlite "github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/sqlite"
)

func TestDatabaseSuite(t *testing.T) {
	suite.Run(t, new(DatabaseSuite))
}

type DatabaseSuite struct {
	suite.Suite

	db *Database
}

func (s *DatabaseSuite) SetupTest() {
	s.db = nil

	dbPath, err := ioutil.TempFile("", "")
	s.NoError(err, "creating temp file for db")

	db, err := appdatabase.InitializeDB(dbPath.Name(), "", sqlite.ReducedKDFIterationsNumber)
	s.NoError(err, "creating sqlite db instance")

	err = protoSqlite.Migrate(db)
	s.NoError(err, "protocol migrate")

	s.db = &Database{db: db}
}

func (s *DatabaseSuite) TestAddOwner() {
	owners, err := s.db.GetTokenOwners(5, "0x123")
	s.Require().NoError(err)
	s.Require().Len(owners, 0)

	err = s.db.AddTokenOwners(5, "0x123", []string{"A", "B"})
	s.Require().NoError(err)
	owners, err = s.db.GetTokenOwners(5, "0x123")
	s.Require().NoError(err)
	s.Require().Len(owners, 2)

	s.Equal(owners[0].Amount, 1)
	s.Equal(owners[1].Amount, 1)

	err = s.db.AddTokenOwners(5, "0x123", []string{"a"})
	s.Require().NoError(err)
	owners, err = s.db.GetTokenOwners(5, "0x123")
	s.Require().NoError(err)
	s.Require().Len(owners, 2)

	amount, err := s.db.GetAmount(5, "0x123", "A")
	s.Require().NoError(err)
	s.Equal(amount, 2)

	amount, err = s.db.GetAmount(5, "0x123", "B")
	s.Require().NoError(err)
	s.Equal(amount, 1)
}
