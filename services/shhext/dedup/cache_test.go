package dedup

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/storage"
)

func TestDedupCacheTestSuite(t *testing.T) {
	suite.Run(t, new(DedupCacheTestSuite))
}

type DedupCacheTestSuite struct {
	suite.Suite
	c  *cache
	db *leveldb.DB
}

func (s *DedupCacheTestSuite) SetupTest() {
	db, err := leveldb.Open(storage.NewMemStorage(), nil)

	if err != nil {
		panic(err)
	}
	s.db = db

	s.c = newCache(db)
}

func (s *DedupCacheTestSuite) TearDownTest() {
	s.NoError(s.db.Close())
}

func (s *DedupCacheTestSuite) TestMultipleFilterIDs() {
	const (
		filterID1 = "filter-id1"
		filterID2 = "filter-id2"
		filterID3 = "filter-id"
	)
	messagesFilter1 := generateMessages(10)
	s.NoError(s.c.Put(filterID1, messagesFilter1))

	for _, msg := range messagesFilter1 {
		has, err := s.c.Has(filterID1, msg)
		s.NoError(err)
		s.True(has)

		has, err = s.c.Has(filterID2, msg)
		s.NoError(err)
		s.False(has)

		has, err = s.c.Has(filterID3, msg)
		s.NoError(err)
		s.False(has)
	}

	messagesFilter2 := generateMessages(10)
	s.NoError(s.c.Put(filterID2, messagesFilter2))

	for _, msg := range messagesFilter2 {
		has, err := s.c.Has(filterID1, msg)
		s.NoError(err)
		s.False(has)

		has, err = s.c.Has(filterID2, msg)
		s.NoError(err)
		s.True(has)

		has, err = s.c.Has(filterID3, msg)
		s.NoError(err)
		s.False(has)
	}
}

func (s *DedupCacheTestSuite) TestCleaningUp() {
	const filterID = "filter1-id"
	// - 2 days
	s.c.now = func() time.Time { return time.Now().Add(-48 * time.Hour) }
	messages2DaysOld := generateMessages(10)
	s.NoError(s.c.Put(filterID, messages2DaysOld))

	for _, msg := range messages2DaysOld {
		has, err := s.c.Has(filterID, msg)
		s.NoError(err)
		s.True(has)
	}

	// - 1 days
	s.c.now = func() time.Time { return time.Now().Add(-24 * time.Hour) }
	messages1DayOld := generateMessages(10)
	s.NoError(s.c.Put(filterID, messages1DayOld))

	for _, msg := range messages2DaysOld {
		has, err := s.c.Has(filterID, msg)
		s.NoError(err)
		s.True(has)
	}

	for _, msg := range messages1DayOld {
		has, err := s.c.Has(filterID, msg)
		s.NoError(err)
		s.True(has)
	}

	// now
	s.c.now = time.Now
	messagesToday := generateMessages(10)
	s.NoError(s.c.Put(filterID, messagesToday))

	for _, msg := range messages2DaysOld {
		has, err := s.c.Has(filterID, msg)
		s.NoError(err)
		s.False(has)
	}

	for _, msg := range messages1DayOld {
		has, err := s.c.Has(filterID, msg)
		s.NoError(err)
		s.True(has)
	}

	for _, msg := range messagesToday {
		has, err := s.c.Has(filterID, msg)
		s.NoError(err)
		s.True(has)
	}
}
