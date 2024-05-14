package peersyncing

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/eth-node/types"
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

var testCommunityID = []byte("community-id")

func (s *PeerSyncingSuite) TestBasic() {

	syncMessage := SyncMessage{
		ID:        []byte("test-id"),
		ChatID:    testCommunityID,
		Type:      SyncMessageCommunityType,
		Payload:   []byte("test"),
		Timestamp: 1,
	}

	s.Require().NoError(s.p.Add(syncMessage))

	allMessages, err := s.p.AvailableMessages()

	s.Require().NoError(err)
	s.Require().Len(allMessages, 1)

	byChatID, err := s.p.AvailableMessagesMapByChatIDs([][]byte{syncMessage.ChatID}, 10)

	s.Require().NoError(err)
	s.Require().Len(byChatID, 1)

	byChatID, err = s.p.AvailableMessagesMapByChatIDs([][]byte{[]byte("random-group-id")}, 10)

	s.Require().NoError(err)
	s.Require().Len(byChatID, 0)

	newSyncMessage := SyncMessage{
		ID:        []byte("test-id-2"),
		ChatID:    testCommunityID,
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
		ChatID:    testCommunityID,
		Type:      SyncMessageCommunityType,
		Payload:   []byte("test"),
		Timestamp: 1,
	}

	syncMessage2 := SyncMessage{
		ID:        []byte("test-id-2"),
		ChatID:    testCommunityID,
		Type:      SyncMessageCommunityType,
		Payload:   []byte("test"),
		Timestamp: 2,
	}

	syncMessage3 := SyncMessage{
		ID:        []byte("test-id-3"),
		ChatID:    testCommunityID,
		Type:      SyncMessageCommunityType,
		Payload:   []byte("test"),
		Timestamp: 3,
	}

	syncMessage4 := SyncMessage{
		ID:        []byte("test-id-4"),
		ChatID:    testCommunityID,
		Type:      SyncMessageCommunityType,
		Payload:   []byte("test"),
		Timestamp: 4,
	}

	s.Require().NoError(s.p.Add(syncMessage1))
	s.Require().NoError(s.p.Add(syncMessage2))
	s.Require().NoError(s.p.Add(syncMessage3))
	s.Require().NoError(s.p.Add(syncMessage4))

	byChatID, err := s.p.AvailableMessagesMapByChatIDs([][]byte{testCommunityID}, 10)

	s.Require().NoError(err)
	s.Require().Len(byChatID, 1)
	s.Require().Len(byChatID[types.Bytes2Hex(testCommunityID)], 4)

	byChatID, err = s.p.AvailableMessagesMapByChatIDs([][]byte{testCommunityID}, 3)

	s.Require().NoError(err)
	s.Require().Len(byChatID, 1)
	s.Require().Len(byChatID[types.Bytes2Hex(testCommunityID)], 3)
}
