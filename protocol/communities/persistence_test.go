package communities

import (
	"crypto/ecdsa"
	"database/sql"
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/appdatabase"
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

	db, err := appdatabase.InitializeDB(dbPath.Name(), "", sqlite.ReducedKDFIterationsNumber)
	s.NoError(err, "creating sqlite db instance")

	err = sqlite.Migrate(db)
	s.NoError(err, "protocol migrate")

	s.db = &Persistence{db: db}
}

func (s *PersistenceSuite) TestSaveCommunity() {
	id, err := crypto.GenerateKey()
	s.Require().NoError(err)

	// there is one community inserted by default
	communities, err := s.db.AllCommunities(&id.PublicKey)
	s.Require().NoError(err)
	s.Require().Len(communities, 1)

	community := Community{
		config: &Config{
			PrivateKey:           id,
			ID:                   &id.PublicKey,
			Joined:               true,
			Spectated:            true,
			Verified:             true,
			CommunityDescription: &protobuf.CommunityDescription{},
		},
	}
	s.Require().NoError(s.db.SaveCommunity(&community))

	communities, err = s.db.AllCommunities(&id.PublicKey)
	s.Require().NoError(err)
	s.Require().Len(communities, 2)
	s.Equal(types.HexBytes(crypto.CompressPubkey(&id.PublicKey)), communities[1].ID())
	s.Equal(true, communities[1].Joined())
	s.Equal(true, communities[1].Spectated())
	s.Equal(true, communities[1].Verified())
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
	sc, err := com.ToSyncCommunityProtobuf(clock, nil)
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

func (s *PersistenceSuite) TestSaveRequestToLeave() {
	rtl := &RequestToLeave{
		ID:          []byte("0x123456"),
		PublicKey:   "0xffffff",
		Clock:       2,
		CommunityID: []byte("0x654321"),
	}

	err := s.db.SaveRequestToLeave(rtl)
	s.NoError(err)

	// older clocks should not be saved
	rtl.Clock = 1
	err = s.db.SaveRequestToLeave(rtl)
	s.Error(err)
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

func (s *PersistenceSuite) TestGetSyncedRawCommunity() {
	sc := &protobuf.SyncCommunity{
		Id:          []byte("0x123456"),
		PrivateKey:  []byte("0xfedcba"),
		Description: []byte("this is a description"),
		Joined:      true,
		Verified:    true,
		Spectated:   true,
	}

	// add a new community to the db
	err := s.db.saveRawCommunityRowWithoutSyncedAt(fromSyncCommunityProtobuf(sc))
	s.NoError(err, "saveRawCommunityRow")

	// retrieve row from db synced_at must be zero
	rcr, err := s.db.getRawCommunityRow(sc.Id)
	s.NoError(err, "getRawCommunityRow")
	s.Zero(rcr.SyncedAt, "synced_at must be zero value")

	// retrieve synced row from db, should fail
	src, err := s.db.getSyncedRawCommunity(sc.Id)
	s.EqualError(err, sql.ErrNoRows.Error())
	s.Nil(src)

	// Set the synced_at value
	clock := uint64(time.Now().Unix())
	err = s.db.SetSyncClock(sc.Id, clock)
	s.NoError(err, "SetSyncClock")

	// retrieve row from db synced_at must not be zero
	rcr, err = s.db.getRawCommunityRow(sc.Id)
	s.NoError(err, "getRawCommunityRow")
	s.NotZero(rcr.SyncedAt, "synced_at must be zero value")

	// retrieve synced row from db, should succeed
	src, err = s.db.getSyncedRawCommunity(sc.Id)
	s.NoError(err)
	s.NotNil(src)
	s.Equal(clock, src.SyncedAt)
}

func (s *PersistenceSuite) TestGetCommunitiesSettings() {
	settings := []CommunitySettings{
		{CommunityID: "0x01", HistoryArchiveSupportEnabled: false},
		{CommunityID: "0x02", HistoryArchiveSupportEnabled: true},
		{CommunityID: "0x03", HistoryArchiveSupportEnabled: false},
	}

	for i := range settings {
		stg := settings[i]
		err := s.db.SaveCommunitySettings(stg)
		s.NoError(err)
	}

	rst, err := s.db.GetCommunitiesSettings()
	s.NoError(err)
	s.Equal(settings, rst)
}

func (s *PersistenceSuite) TestSaveCommunitySettings() {
	settings := CommunitySettings{CommunityID: "0x01", HistoryArchiveSupportEnabled: false}
	err := s.db.SaveCommunitySettings(settings)
	s.NoError(err)
	rst, err := s.db.GetCommunitiesSettings()
	s.NoError(err)
	s.Equal(1, len(rst))
}

func (s *PersistenceSuite) TestDeleteCommunitySettings() {
	settings := CommunitySettings{CommunityID: "0x01", HistoryArchiveSupportEnabled: false}

	err := s.db.SaveCommunitySettings(settings)
	s.NoError(err)

	rst, err := s.db.GetCommunitiesSettings()
	s.NoError(err)
	s.Equal(1, len(rst))
	s.NoError(s.db.DeleteCommunitySettings(types.HexBytes{0x01}))
	rst2, err := s.db.GetCommunitiesSettings()
	s.NoError(err)
	s.Equal(0, len(rst2))
}

func (s *PersistenceSuite) TestUpdateCommunitySettings() {
	settings := []CommunitySettings{
		{CommunityID: "0x01", HistoryArchiveSupportEnabled: true},
		{CommunityID: "0x02", HistoryArchiveSupportEnabled: false},
	}

	s.NoError(s.db.SaveCommunitySettings(settings[0]))
	s.NoError(s.db.SaveCommunitySettings(settings[1]))

	settings[0].HistoryArchiveSupportEnabled = true
	settings[1].HistoryArchiveSupportEnabled = false

	s.NoError(s.db.UpdateCommunitySettings(settings[0]))
	s.NoError(s.db.UpdateCommunitySettings(settings[1]))

	rst, err := s.db.GetCommunitiesSettings()
	s.NoError(err)
	s.Equal(settings, rst)
}

func (s *PersistenceSuite) TestGetCommunityTokens() {
	tokens, err := s.db.GetCommunityTokens("123")
	s.Require().NoError(err)
	s.Require().Len(tokens, 0)

	token := CommunityToken{
		CommunityID:        "123",
		TokenType:          protobuf.CommunityTokenType_ERC721,
		Address:            "0x123",
		Name:               "StatusToken",
		Symbol:             "STT",
		Description:        "desc",
		Supply:             123,
		InfiniteSupply:     false,
		Transferable:       true,
		RemoteSelfDestruct: true,
		ChainID:            1,
		DeployState:        InProgress,
		Base64Image:        "ABCD",
	}

	token2 := CommunityToken{
		CommunityID:        "345",
		TokenType:          protobuf.CommunityTokenType_ERC721,
		Address:            "0x345",
		Name:               "StatusToken",
		Symbol:             "STT",
		Description:        "desc",
		Supply:             345,
		InfiniteSupply:     false,
		Transferable:       true,
		RemoteSelfDestruct: true,
		ChainID:            2,
		DeployState:        Failed,
		Base64Image:        "QWERTY",
	}

	err = s.db.AddCommunityToken(&token)
	s.Require().NoError(err)
	err = s.db.AddCommunityToken(&token2)
	s.Require().NoError(err)

	tokens, err = s.db.GetCommunityTokens("123")
	s.Require().NoError(err)
	s.Require().Len(tokens, 1)
	s.Require().Equal(token, *tokens[0])

	err = s.db.UpdateCommunityTokenState(1, "0x123", Deployed)
	s.Require().NoError(err)
	tokens, err = s.db.GetCommunityTokens("123")
	s.Require().NoError(err)
	s.Require().Len(tokens, 1)
	s.Require().Equal(Deployed, tokens[0].DeployState)
}
