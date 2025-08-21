package multidevice

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/t/helpers"
)

func TestSQLLitePersistenceTestSuite(t *testing.T) {
	suite.Run(t, new(SQLLitePersistenceTestSuite))
}

type SQLLitePersistenceTestSuite struct {
	suite.Suite
	service *sqlitePersistence
}

func (s *SQLLitePersistenceTestSuite) SetupTest() {
	db, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	s.Require().NoError(err)
	err = sqlite.Migrate(db)
	s.Require().NoError(err)

	s.service = newSQLitePersistence(db)
}

func (s *SQLLitePersistenceTestSuite) TestAddInstallations() {
	identity := []byte("alice")
	installations := []*Installation{
		{ID: "alice-1", Version: 1, Enabled: true},
		{ID: "alice-2", Version: 2, Enabled: true},
	}
	addedInstallations, err := s.service.AddInstallations(
		identity,
		1,
		installations,
		true,
	)
	s.Require().NoError(err)

	enabledInstallations, err := s.service.GetActiveInstallations(5, identity)
	s.Require().NoError(err)

	s.Require().Equal(installations, enabledInstallations)
	s.Require().Equal(installations, addedInstallations)
}

func (s *SQLLitePersistenceTestSuite) TestAddInstallationVersions() {
	identity := []byte("alice")
	installations := []*Installation{
		{ID: "alice-1", Version: 1, Enabled: true},
	}
	_, err := s.service.AddInstallations(
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

	_, err = s.service.AddInstallations(
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

	_, err := s.service.AddInstallations(
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

	_, err = s.service.AddInstallations(
		identity,
		2,
		installations,
		true,
	)
	s.Require().NoError(err)

	installations = []*Installation{
		{ID: "alice-2", Version: 2, Enabled: true},
		{ID: "alice-3", Version: 3, Enabled: true},
		{ID: "alice-4", Version: 4, Enabled: true},
	}

	_, err = s.service.AddInstallations(
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

	_, err := s.service.AddInstallations(
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

	_, err := s.service.AddInstallations(
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

	addedInstallations, err := s.service.AddInstallations(
		identity,
		1,
		installations,
		true,
	)
	s.Require().NoError(err)
	s.Require().Equal(0, len(addedInstallations))

	actualInstallations, err := s.service.GetActiveInstallations(3, identity)
	s.Require().NoError(err)

	expected := []*Installation{{ID: "alice-2", Version: 2, Enabled: true}}
	s.Require().Equal(expected, actualInstallations)
}

func (s *SQLLitePersistenceTestSuite) TestEnableInstallation() {
	identity := []byte("alice")

	installations := []*Installation{
		{ID: "alice-1", Version: 1},
		{ID: "alice-2", Version: 2},
	}

	_, err := s.service.AddInstallations(
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

	expected := []*Installation{{ID: "alice-2", Version: 2, Enabled: true}}
	s.Require().Equal(expected, actualInstallations)

	err = s.service.EnableInstallation(identity, "alice-1")
	s.Require().NoError(err)

	actualInstallations, err = s.service.GetActiveInstallations(3, identity)
	s.Require().NoError(err)

	expected = []*Installation{
		{ID: "alice-1", Version: 1, Enabled: true},
		{ID: "alice-2", Version: 2, Enabled: true},
	}
	s.Require().Equal(expected, actualInstallations)
}

func (s *SQLLitePersistenceTestSuite) TestGetInstallations() {
	identity := []byte("alice")

	installations := []*Installation{
		{ID: "alice-1", Version: 1},
		{ID: "alice-2", Version: 2},
	}

	_, err := s.service.AddInstallations(
		identity,
		1,
		installations,
		true,
	)
	s.Require().NoError(err)

	err = s.service.DisableInstallation(identity, "alice-1")
	s.Require().NoError(err)

	actualInstallations, err := s.service.GetInstallations(identity)
	s.Require().NoError(err)

	emptyMetadata := &InstallationMetadata{}

	expected := []*Installation{
		{ID: "alice-1", Version: 1, Timestamp: 1, Enabled: false, InstallationMetadata: emptyMetadata},
		{ID: "alice-2", Version: 2, Timestamp: 1, Enabled: true, InstallationMetadata: emptyMetadata},
	}
	s.Require().Equal(2, len(actualInstallations))
	s.Require().ElementsMatch(expected, actualInstallations)
}

func (s *SQLLitePersistenceTestSuite) TestSetMetadata() {
	identity := []byte("alice")

	installations := []*Installation{
		{ID: "alice-1", Version: 1},
		{ID: "alice-2", Version: 2},
	}

	_, err := s.service.AddInstallations(
		identity,
		1,
		installations,
		true,
	)
	s.Require().NoError(err)

	err = s.service.DisableInstallation(identity, "alice-1")
	s.Require().NoError(err)

	emptyMetadata := &InstallationMetadata{}
	setMetadata := &InstallationMetadata{
		Name:       "a",
		FCMToken:   "b",
		DeviceType: "c",
	}

	err = s.service.SetInstallationMetadata(identity, "alice-2", setMetadata)
	s.Require().NoError(err)

	actualInstallations, err := s.service.GetInstallations(identity)
	s.Require().NoError(err)

	expected := []*Installation{
		{ID: "alice-1", Version: 1, Timestamp: 1, Enabled: false, InstallationMetadata: emptyMetadata},
		{ID: "alice-2", Version: 2, Timestamp: 1, Enabled: true, InstallationMetadata: setMetadata},
	}
	s.Require().ElementsMatch(expected, actualInstallations)
}
