package peersyncing

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/t/helpers"
)

func TestPeerSyncingSuite(t *testing.T) {
	suite.Run(t, new(PeerSyncingSuite))
}

type PeerSyncingSuite struct {
	suite.Suite
	p *PeerSyncing
}

func (s *PeerSyncingSuite) SetupTest() {
	db, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	s.Require().NoError(err)

	err = sqlite.Migrate(db)
	s.Require().NoError(err)

	s.p = New(Config{Database: db})
}

var testGroupID = []byte("group-id")

func (s *PeerSyncingSuite) TestBasic() {

	syncMessage := SyncMessage{
		ID:        []byte("test-id"),
		GroupID:   testGroupID,
		Type:      SyncMessageCommunityType,
		Payload:   []byte("test"),
		Timestamp: 1,
	}

	s.Require().NoError(s.p.Add(syncMessage))

	allMessages, err := s.p.AvailableMessages()

	s.Require().NoError(err)
	s.Require().Len(allMessages, 1)

	byGroupID, err := s.p.AvailableMessagesByGroupID(syncMessage.GroupID, 10)

	s.Require().NoError(err)
	s.Require().Len(byGroupID, 1)

	byGroupID, err = s.p.AvailableMessagesByGroupID([]byte("random-group-id"), 10)

	s.Require().NoError(err)
	s.Require().Len(byGroupID, 0)

	newSyncMessage := SyncMessage{
		ID:        []byte("test-id-2"),
		GroupID:   testGroupID,
		Type:      SyncMessageCommunityType,
		Payload:   []byte("test-2"),
		Timestamp: 2,
	}

	wantedMessages, err := s.p.OnOffer([]SyncMessage{syncMessage, newSyncMessage})
	s.Require().NoError(err)

	s.Require().Len(wantedMessages, 1)
	s.Require().Equal(newSyncMessage.ID, wantedMessages[0].ID)
}

func (s *PeerSyncingSuite) TestOrderAndLimit() {

	syncMessage1 := SyncMessage{
		ID:        []byte("test-id-1"),
		GroupID:   testGroupID,
		Type:      SyncMessageCommunityType,
		Payload:   []byte("test"),
		Timestamp: 1,
	}

	syncMessage2 := SyncMessage{
		ID:        []byte("test-id-2"),
		GroupID:   testGroupID,
		Type:      SyncMessageCommunityType,
		Payload:   []byte("test"),
		Timestamp: 2,
	}

	syncMessage3 := SyncMessage{
		ID:        []byte("test-id-3"),
		GroupID:   testGroupID,
		Type:      SyncMessageCommunityType,
		Payload:   []byte("test"),
		Timestamp: 3,
	}

	syncMessage4 := SyncMessage{
		ID:        []byte("test-id-4"),
		GroupID:   testGroupID,
		Type:      SyncMessageCommunityType,
		Payload:   []byte("test"),
		Timestamp: 4,
	}

	s.Require().NoError(s.p.Add(syncMessage1))
	s.Require().NoError(s.p.Add(syncMessage2))
	s.Require().NoError(s.p.Add(syncMessage3))
	s.Require().NoError(s.p.Add(syncMessage4))

	byGroupID, err := s.p.AvailableMessagesByGroupID(testGroupID, 10)

	s.Require().NoError(err)
	s.Require().Len(byGroupID, 4)

	s.Require().Equal(syncMessage1.ID, byGroupID[3].ID)
	s.Require().Equal(syncMessage2.ID, byGroupID[2].ID)
	s.Require().Equal(syncMessage3.ID, byGroupID[1].ID)
	s.Require().Equal(syncMessage4.ID, byGroupID[0].ID)

	byGroupID, err = s.p.AvailableMessagesByGroupID(testGroupID, 3)

	s.Require().NoError(err)
	s.Require().Len(byGroupID, 3)

	s.Require().Equal(syncMessage2.ID, byGroupID[2].ID)
	s.Require().Equal(syncMessage3.ID, byGroupID[1].ID)
	s.Require().Equal(syncMessage4.ID, byGroupID[0].ID)
}
