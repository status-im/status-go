package newsfeed

import (
	"database/sql"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/t/helpers"
)

type SQLitePersistenceTestSuite struct {
	suite.Suite
	db          *sql.DB
	persistence *SQLitePersistence
}

func TestSQLitePersistenceTestSuite(t *testing.T) {
	suite.Run(t, new(SQLitePersistenceTestSuite))
}

func (s *SQLitePersistenceTestSuite) SetupTest() {
	db, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	s.Require().NoError(err)
	s.db = db

	err = SQLiteMigrate(s.db)
	s.Require().NoError(err)

	s.persistence = NewSQLitePersistence(s.db)
}

func (s *SQLitePersistenceTestSuite) TearDownTest() {
	if s.db != nil {
		s.Require().NoError(s.db.Close())
	}
}

func (s *SQLitePersistenceTestSuite) TestNewsFeedEnabled_ToggleMultipleTimes() {
	for i := 0; i < 3; i++ {
		err := s.persistence.SaveNewsFeedEnabled(true)
		s.Require().NoError(err)

		enabled, err := s.persistence.NewsFeedEnabled()
		s.Require().NoError(err)
		s.Require().True(enabled)

		err = s.persistence.SaveNewsFeedEnabled(false)
		s.Require().NoError(err)

		enabled, err = s.persistence.NewsFeedEnabled()
		s.Require().NoError(err)
		s.Require().False(enabled)
	}
}

func (s *SQLitePersistenceTestSuite) TestDefaultValues() {
	newsFeedEnabled, err := s.persistence.NewsFeedEnabled()
	s.Require().NoError(err)
	s.Require().True(newsFeedEnabled)

	newsRSSEnabled, err := s.persistence.NewsRSSEnabled()
	s.Require().NoError(err)
	s.Require().True(newsRSSEnabled)

	timestamp, err := s.persistence.NewsFeedLastFetchedTimestamp()
	s.Require().NoError(err)
	s.Require().False(timestamp.IsZero())
}

func (s *SQLitePersistenceTestSuite) TestNewsRSSEnabled_ToggleMultipleTimes() {
	for i := 0; i < 3; i++ {
		err := s.persistence.SaveNewsRSSEnabled(true)
		s.Require().NoError(err)

		enabled, err := s.persistence.NewsRSSEnabled()
		s.Require().NoError(err)
		s.Require().True(enabled)

		err = s.persistence.SaveNewsRSSEnabled(false)
		s.Require().NoError(err)

		enabled, err = s.persistence.NewsRSSEnabled()
		s.Require().NoError(err)
		s.Require().False(enabled)
	}
}

func (s *SQLitePersistenceTestSuite) TestNewsFeedLastFetchedTimestamp_UpdateMultipleTimes() {
	timestamps := []time.Time{
		gofakeit.Date(),
		gofakeit.Date(),
		gofakeit.Date(),
	}

	for _, expected := range timestamps {
		err := s.persistence.SaveNewsFeedLastFetchedTimestamp(expected)
		s.Require().NoError(err)

		timestamp, err := s.persistence.NewsFeedLastFetchedTimestamp()
		s.Require().NoError(err)
		s.Require().Equal(expected.Unix(), timestamp.Unix())
	}
}

func (s *SQLitePersistenceTestSuite) TestNewSQLitePersistence() {
	persistence := NewSQLitePersistence(s.db)
	s.Require().NotNil(persistence)
	s.Require().Equal(s.db, persistence.db)
}
