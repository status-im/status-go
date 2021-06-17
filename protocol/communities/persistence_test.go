package communities

import (
	"github.com/status-im/status-go/protocol/protobuf"
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/protocol/sqlite"
)

func TestPersistenceSuite(t *testing.T) {
	suite.Run(t, new(PersistenceSuite))
}

type PersistenceSuite struct {
	suite.Suite

	db *Persistence
}

func (s *PersistenceSuite) SetupTest() {
	s.db = nil

	dbPath, err := ioutil.TempFile("", "")
	s.NoError(err, "creating temp file for db")

	db, err := sqlite.Open(dbPath.Name(), "")
	s.NoError(err, "creating sqlite db instance")

	s.db = &Persistence{ db: db }
}

func (s *PersistenceSuite) TestShouldHandleSyncCommunity() {
	sc := &protobuf.SyncCommunity{
		Id:                   []byte("0x123456"),
		PrivateKey:           []byte("0xfedcba"),
		Description:          []byte("this is a description"),
		Joined:               true,
		Verified:             true,
		Clock:                uint64(time.Now().Unix()),
	}

	// check an empty db to see if a community should be synced
	should, err := s.db.ShouldHandleSyncCommunity(sc)
	s.NoError(err, "SaveSyncCommunity")
	s.True(should)

	// add a new community to the db
	err = s.db.saveRawCommunityRow(fromSyncCommunityProtobuf(sc))
	s.NoError(err, "saveRawCommunityRow")

	rcrs, err := s.db.getAllCommunitiesRaw()
	s.NoError(err, "should have no error from getAllCommunitiesRaw")
	s.Len(rcrs, 2, "length of all communities raw should be 2")

	// check again to see is the community should be synced
	sc.Clock--
	should, err = s.db.ShouldHandleSyncCommunity(sc)
	s.NoError(err, "SaveSyncCommunity")
	s.False(should)
}
