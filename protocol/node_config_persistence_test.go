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

const (
	testWakuNodeEnrtree   = "enrtree://AL65EKLJAUXKKPG43HVTML5EFFWEZ7L4LOKTLZCLJASG4DSESQZEC@prod.status.nodes.status.im"
	testWakuNodeMultiaddr = "/ip4/127.0.0.1/tcp/34012"
)

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

func (s *NodeConfigPersistenceTestSuite) Test_SaveNewWakuNode() {
	// GIVEN
	wakuNodesBeforeChanges := s.config.ClusterConfig.WakuNodes

	// WHEN
	err := nodecfg.SaveNewWakuNode(s.db, testWakuNodeEnrtree)
	s.Require().NoError(err)
	err = nodecfg.SaveNewWakuNode(s.db, testWakuNodeMultiaddr)
	s.Require().NoError(err)

	// THEN
	dbNodeConfig, err := nodecfg.GetNodeConfigFromDB(s.db)
	s.Require().NoError(err)
	s.Require().Len(dbNodeConfig.ClusterConfig.WakuNodes, len(wakuNodesBeforeChanges)+2)
	s.Require().Contains(dbNodeConfig.ClusterConfig.WakuNodes, testWakuNodeEnrtree)
	s.Require().Contains(dbNodeConfig.ClusterConfig.WakuNodes, testWakuNodeMultiaddr)
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
