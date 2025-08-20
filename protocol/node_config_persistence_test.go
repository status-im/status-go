package protocol

import (
	"database/sql"
	"testing"

	"github.com/status-im/status-go/params"

	"github.com/status-im/status-go/nodecfg"

	"github.com/stretchr/testify/suite"
)

func TestNodeConfigPersistence(t *testing.T) {
	suite.Run(t, new(NodeConfigPersistenceTestSuite))
}

type NodeConfigPersistenceTestSuite struct {
	suite.Suite
	db     *sql.DB
	config *params.NodeConfig
}

func (s *NodeConfigPersistenceTestSuite) SetupTest() {
	db, err := openTestDB()
	s.Require().NoError(err)
	s.db = db

	s.config, err = nodecfg.GetNodeConfigFromDB(s.db)
	s.Require().NoError(err)

	// write value to the db, otherwise log_config table won't be created
	err = nodecfg.SaveNodeConfig(s.db, s.config)
	s.Require().NoError(err)
}

func (s *NodeConfigPersistenceTestSuite) Test_SaveMaxLogBackups() {
	// GIVEN
	maxLogBackupsBeforeChanges := s.config.LogMaxBackups

	// WHEN
	err := nodecfg.SetMaxLogBackups(s.db, uint(maxLogBackupsBeforeChanges+10))
	s.Require().NoError(err)

	// THEN
	dbNodeConfig, err := nodecfg.GetNodeConfigFromDB(s.db)
	s.Require().NoError(err)
	s.Require().Equal(maxLogBackupsBeforeChanges+10, dbNodeConfig.LogMaxBackups)
}

func (s *NodeConfigPersistenceTestSuite) Test_SetLogLevelError() {
	// WHEN
	err := nodecfg.SetLogLevel(s.db, "ERROR")
	s.Require().NoError(err)

	// THEN
	dbNodeConfig, err := nodecfg.GetNodeConfigFromDB(s.db)
	s.Require().NoError(err)
	s.Require().Equal("ERROR", dbNodeConfig.LogLevel)
}

func (s *NodeConfigPersistenceTestSuite) Test_SetLogLevelDebug() {
	// WHEN
	err := nodecfg.SetLogLevel(s.db, "DEBUG")
	s.Require().NoError(err)

	// THEN
	dbNodeConfig, err := nodecfg.GetNodeConfigFromDB(s.db)
	s.Require().NoError(err)
	s.Require().Equal("DEBUG", dbNodeConfig.LogLevel)
}

func (s *NodeConfigPersistenceTestSuite) Test_SetLogEnabled() {
	// WHEN
	err := nodecfg.SetLogEnabled(s.db, false)
	s.Require().NoError(err)

	// THEN
	dbNodeConfig, err := nodecfg.GetNodeConfigFromDB(s.db)
	s.Require().NoError(err)
	s.Require().Equal(false, dbNodeConfig.LogEnabled)

	// WHEN
	err = nodecfg.SetLogEnabled(s.db, true)
	s.Require().NoError(err)

	// THEN
	dbNodeConfig, err = nodecfg.GetNodeConfigFromDB(s.db)
	s.Require().NoError(err)
	s.Require().Equal(true, dbNodeConfig.LogEnabled)
}
