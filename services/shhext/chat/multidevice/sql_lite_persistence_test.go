package multidevice

import (
	"database/sql"
	"os"
	"testing"

	appDB "github.com/status-im/status-go/services/shhext/chat/db"
	"github.com/stretchr/testify/suite"
)

const (
	dbPath = "/tmp/status-key-store.db"
)

func TestSQLLitePersistenceTestSuite(t *testing.T) {
	suite.Run(t, new(SQLLitePersistenceTestSuite))
}

type SQLLitePersistenceTestSuite struct {
	suite.Suite
	// nolint: structcheck, megacheck
	db      *sql.DB
	service Persistence
}

func (s *SQLLitePersistenceTestSuite) SetupTest() {
	os.Remove(dbPath)

	db, err := appDB.Open(dbPath, "", 0)
	s.Require().NoError(err)

	s.service = NewSQLLitePersistence(db)
}

func (s *SQLLitePersistenceTestSuite) TestAddInstallations() {
	identity := []byte("alice")
	installations := []*Installation{
		{ID: "alice-1", Version: 1},
		{ID: "alice-2", Version: 2},
	}
	err := s.service.AddInstallations(
		identity,
		1,
		installations,
		true,
	)

	s.Require().NoError(err)

	enabledInstallations, err := s.service.GetActiveInstallations(5, identity)
	s.Require().NoError(err)

	s.Require().Equal(installations, enabledInstallations)
}

func (s *SQLLitePersistenceTestSuite) TestAddInstallationVersions() {
	identity := []byte("alice")
	installations := []*Installation{
		{ID: "alice-1", Version: 1},
	}
	err := s.service.AddInstallations(
		identity,
		1,
		installations,
		true,
	)

	s.Require().NoError(err)

	enabledInstallations, err := s.service.GetActiveInstallations(5, identity)
	s.Require().NoError(err)

	s.Require().Equal(installations, enabledInstallations)

	installationsWithDowngradedVersion := []*Installation{
		{ID: "alice-1", Version: 0},
	}

	err = s.service.AddInstallations(
		identity,
		3,
		installationsWithDowngradedVersion,
		true,
	)
	s.Require().NoError(err)

	enabledInstallations, err = s.service.GetActiveInstallations(5, identity)
	s.Require().NoError(err)
	s.Require().Equal(installations, enabledInstallations)
}

func (s *SQLLitePersistenceTestSuite) TestAddInstallationsLimit() {
	identity := []byte("alice")

	installations := []*Installation{
		{ID: "alice-1", Version: 1},
		{ID: "alice-2", Version: 2},
	}

	err := s.service.AddInstallations(
		identity,
		1,
		installations,
		true,
	)
	s.Require().NoError(err)

	installations = []*Installation{
		{ID: "alice-1", Version: 1},
		{ID: "alice-3", Version: 3},
	}

	err = s.service.AddInstallations(
		identity,
		2,
		installations,
		true,
	)
	s.Require().NoError(err)

	installations = []*Installation{
		{ID: "alice-2", Version: 2},
		{ID: "alice-3", Version: 3},
		{ID: "alice-4", Version: 4},
	}

	err = s.service.AddInstallations(
		identity,
		3,
		installations,
		true,
	)
	s.Require().NoError(err)

	enabledInstallations, err := s.service.GetActiveInstallations(3, identity)
	s.Require().NoError(err)

	s.Require().Equal(installations, enabledInstallations)
}

func (s *SQLLitePersistenceTestSuite) TestAddInstallationsDisabled() {
	identity := []byte("alice")

	installations := []*Installation{
		{ID: "alice-1", Version: 1},
		{ID: "alice-2", Version: 2},
	}

	err := s.service.AddInstallations(
		identity,
		1,
		installations,
		false,
	)
	s.Require().NoError(err)

	actualInstallations, err := s.service.GetActiveInstallations(3, identity)
	s.Require().NoError(err)

	s.Require().Nil(actualInstallations)
}

func (s *SQLLitePersistenceTestSuite) TestDisableInstallation() {
	identity := []byte("alice")

	installations := []*Installation{
		{ID: "alice-1", Version: 1},
		{ID: "alice-2", Version: 2},
	}

	err := s.service.AddInstallations(
		identity,
		1,
		installations,
		true,
	)
	s.Require().NoError(err)

	err = s.service.DisableInstallation(identity, "alice-1")
	s.Require().NoError(err)

	// We add the installations again
	installations = []*Installation{
		{ID: "alice-1", Version: 1},
		{ID: "alice-2", Version: 2},
	}

	err = s.service.AddInstallations(
		identity,
		1,
		installations,
		true,
	)
	s.Require().NoError(err)

	actualInstallations, err := s.service.GetActiveInstallations(3, identity)
	s.Require().NoError(err)

	expected := []*Installation{{ID: "alice-2", Version: 2}}
	s.Require().Equal(expected, actualInstallations)
}

func (s *SQLLitePersistenceTestSuite) TestEnableInstallation() {
	identity := []byte("alice")

	installations := []*Installation{
		{ID: "alice-1", Version: 1},
		{ID: "alice-2", Version: 2},
	}

	err := s.service.AddInstallations(
		identity,
		1,
		installations,
		true,
	)
	s.Require().NoError(err)

	err = s.service.DisableInstallation(identity, "alice-1")
	s.Require().NoError(err)

	actualInstallations, err := s.service.GetActiveInstallations(3, identity)
	s.Require().NoError(err)

	expected := []*Installation{{ID: "alice-2", Version: 2}}
	s.Require().Equal(expected, actualInstallations)

	err = s.service.EnableInstallation(identity, "alice-1")
	s.Require().NoError(err)

	actualInstallations, err = s.service.GetActiveInstallations(3, identity)
	s.Require().NoError(err)

	expected = []*Installation{
		{ID: "alice-1", Version: 1},
		{ID: "alice-2", Version: 2},
	}
	s.Require().Equal(expected, actualInstallations)

}
