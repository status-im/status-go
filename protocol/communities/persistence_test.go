package communities

import (
	"crypto/ecdsa"
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
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

	s.db = &Persistence{db: db}
}

func (s *PersistenceSuite) TestShouldHandleSyncCommunity() {
	sc := &protobuf.SyncCommunity{
		Id:          []byte("0x123456"),
		PrivateKey:  []byte("0xfedcba"),
		Description: []byte("this is a description"),
		Joined:      true,
		Verified:    true,
		Clock:       uint64(time.Now().Unix()),
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

	// check again to see is the community should be synced
	sc.Clock++
	sc.Clock++
	should, err = s.db.ShouldHandleSyncCommunity(sc)
	s.NoError(err, "SaveSyncCommunity")
	s.True(should)
}

func (s *PersistenceSuite) TestSetSyncClock() {
	sc := &protobuf.SyncCommunity{
		Id:          []byte("0x123456"),
		PrivateKey:  []byte("0xfedcba"),
		Description: []byte("this is a description"),
		Joined:      true,
		Verified:    true,
	}

	// add a new community to the db
	err := s.db.saveRawCommunityRow(fromSyncCommunityProtobuf(sc))
	s.NoError(err, "saveRawCommunityRow")

	// retrieve row from db synced_at must be zero
	rcr, err := s.db.getRawCommunityRow(sc.Id)
	s.NoError(err, "getRawCommunityRow")
	s.Zero(rcr.SyncedAt, "synced_at must be zero value")

	// Set the synced_at value
	clock := uint64(time.Now().Unix())
	err = s.db.SetSyncClock(sc.Id, clock)
	s.NoError(err, "SetSyncClock")

	// Retrieve row from db and check clock matches synced_at value
	rcr, err = s.db.getRawCommunityRow(sc.Id)
	s.NoError(err, "getRawCommunityRow")
	s.Equal(clock, rcr.SyncedAt, "synced_at must equal the value of the clock")

	// Set Synced At with an older clock value
	olderClock := clock - uint64(256)
	err = s.db.SetSyncClock(sc.Id, olderClock)
	s.NoError(err, "SetSyncClock")

	// Retrieve row from db and check olderClock matches synced_at value
	rcr, err = s.db.getRawCommunityRow(sc.Id)
	s.NoError(err, "getRawCommunityRow")
	s.NotEqual(olderClock, rcr.SyncedAt, "synced_at must not equal the value of the olderClock value")

	// Set Synced At with a newer clock value
	newerClock := clock + uint64(512)
	err = s.db.SetSyncClock(sc.Id, newerClock)
	s.NoError(err, "SetSyncClock")

	// Retrieve row from db and check olderClock matches synced_at value
	rcr, err = s.db.getRawCommunityRow(sc.Id)
	s.NoError(err, "getRawCommunityRow")
	s.Equal(newerClock, rcr.SyncedAt, "synced_at must equal the value of the newerClock value")
}

func (s *PersistenceSuite) TestSetPrivateKey() {
	sc := &protobuf.SyncCommunity{
		Id:          []byte("0x123456"),
		Description: []byte("this is a description"),
		Joined:      true,
		Verified:    true,
	}

	// add a new community to the db with no private key
	err := s.db.saveRawCommunityRow(fromSyncCommunityProtobuf(sc))
	s.NoError(err, "saveRawCommunityRow")

	// retrieve row from db, private key must be zero
	rcr, err := s.db.getRawCommunityRow(sc.Id)
	s.NoError(err, "getRawCommunityRow")
	s.Zero(rcr.PrivateKey, "private key must be zero value")

	// Set private key
	pk, err := crypto.GenerateKey()
	s.NoError(err, "crypto.GenerateKey")
	err = s.db.SetPrivateKey(sc.Id, pk)
	s.NoError(err, "SetPrivateKey")

	// retrieve row from db again, private key must match the given key
	rcr, err = s.db.getRawCommunityRow(sc.Id)
	s.NoError(err, "getRawCommunityRow")
	s.Equal(crypto.FromECDSA(pk), rcr.PrivateKey, "private key must match given key")
}

func (s *PersistenceSuite) TestJoinedAndPendingCommunitiesWithRequests() {
	identity, err := crypto.GenerateKey()
	s.NoError(err, "crypto.GenerateKey shouldn't give any error")

	clock := uint64(time.Now().Unix())

	// Add a new community that we have joined
	com := s.makeNewCommunity(identity)
	com.Join()
	sc, err := com.ToSyncCommunityProtobuf(clock)
	s.NoError(err, "Community.ToSyncCommunityProtobuf shouldn't give any error")
	err = s.db.saveRawCommunityRow(fromSyncCommunityProtobuf(sc))
	s.NoError(err, "saveRawCommunityRow")

	// Add a new community that we have requested to join, but not yet joined
	com2 := s.makeNewCommunity(identity)
	err = s.db.SaveCommunity(com2)
	s.NoError(err, "SaveCommunity shouldn't give any error")

	rtj := &RequestToJoin{
		ID:          types.HexBytes{1, 2, 3, 4, 5, 6, 7, 8},
		PublicKey:   common.PubkeyToHex(&identity.PublicKey),
		Clock:       clock,
		CommunityID: com2.ID(),
		State:       RequestToJoinStatePending,
	}
	err = s.db.SaveRequestToJoin(rtj)
	s.NoError(err, "SaveRequestToJoin shouldn't give any error")

	comms, err := s.db.JoinedAndPendingCommunitiesWithRequests(&identity.PublicKey)
	s.NoError(err, "JoinedAndPendingCommunitiesWithRequests shouldn't give any error")
	s.Len(comms, 2, "Should have 2 communities")

	for _, comm := range comms {
		switch comm.IDString() {
		case com.IDString():
			s.Len(comm.RequestsToJoin(), 0, "Should have no RequestsToJoin")
		case com2.IDString():
			rtjs := comm.RequestsToJoin()
			s.Len(rtjs, 1, "Should have one RequestsToJoin")
			s.Equal(rtjs[0], rtj, "RequestToJoin should match the Request stored in the db")
		}
	}
}

func (s *PersistenceSuite) makeNewCommunity(identity *ecdsa.PrivateKey) *Community {
	comPrivKey, err := crypto.GenerateKey()
	s.NoError(err, "crypto.GenerateKey shouldn't give any error")

	com, err := New(Config{
		MemberIdentity: &identity.PublicKey,
		PrivateKey:     comPrivKey,
		ID:             &comPrivKey.PublicKey,
	})
	s.NoError(err, "New shouldn't give any error")

	md, err := com.MarshaledDescription()
	s.NoError(err, "Community.MarshaledDescription shouldn't give any error")
	com.config.MarshaledCommunityDescription = md

	return com
}
