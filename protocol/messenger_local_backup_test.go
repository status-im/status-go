package protocol

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/crypto"
	"github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
)

func TestMessengerLocalBackupSuite(t *testing.T) {
	suite.Run(t, new(MessengerLocalBackupSuite))
}

type MessengerLocalBackupSuite struct {
	MessengerBaseTestSuite
}

func (s *MessengerLocalBackupSuite) TestLocalBackup() {
	// Create bob1
	bob1 := s.anotherMessenger()
	defer TearDownMessenger(&s.Suite, bob1)

	// Create bob2
	bob2 := s.anotherMessenger()
	defer TearDownMessenger(&s.Suite, bob2)

	// -------------------- CONTACTS --------------------
	// Create 2 contacts
	contact1Key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	contactID1 := types.EncodeHex(crypto.FromECDSAPub(&contact1Key.PublicKey))

	_, err = bob1.AddContact(context.Background(), &requests.AddContact{ID: contactID1})
	s.Require().NoError(err)

	contact2Key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	contactID2 := types.EncodeHex(crypto.FromECDSAPub(&contact2Key.PublicKey))

	_, err = bob1.AddContact(context.Background(), &requests.AddContact{ID: contactID2})
	s.Require().NoError(err)

	s.Require().Len(bob1.Contacts(), 2)

	// Validate contacts on bob1
	actualContacts := bob1.Contacts()
	if actualContacts[0].ID == contactID1 {
		s.Require().Equal(actualContacts[0].ID, contactID1)
		s.Require().Equal(actualContacts[1].ID, contactID2)
	} else {
		s.Require().Equal(actualContacts[0].ID, contactID2)
		s.Require().Equal(actualContacts[1].ID, contactID1)
	}
	s.Require().Equal(ContactRequestStateSent, actualContacts[0].ContactRequestLocalState)
	s.Require().Equal(ContactRequestStateSent, actualContacts[1].ContactRequestLocalState)
	s.Require().True(actualContacts[0].added())
	s.Require().True(actualContacts[1].added())

	// Check that bob2 has no contacts
	s.Require().Len(bob2.Contacts(), 0)

	//-------------------- COMMUNITIES --------------------
	// Create a community
	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}
	response, err := bob1.CreateCommunity(description, true)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	// Check bob2
	communities, err := bob2.Communities()
	s.Require().NoError(err)
	s.Require().Len(communities, 0)

	// --------------------- LEFT COMMUNITY --------------------
	// Create another community
	description = &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_MANUAL_ACCEPT,
		Name:        "other-status",
		Color:       "#fffff4",
		Description: "other status community description",
	}

	response, err = bob1.CreateCommunity(description, true)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	newCommunity := response.Communities()[0]

	// Leave community
	response, err = bob1.LeaveCommunity(newCommunity.ID())
	s.Require().NoError(err)
	s.Require().NotNil(response)

	// Check bob2
	communities, err = bob2.Communities()
	s.Require().NoError(err)
	s.Require().Len(communities, 0)

	// --------------------- CHATS --------------------
	// Create a group chat
	response, err = bob1.CreateGroupChatWithMembers(context.Background(), "group", []string{})
	s.NoError(err)
	s.Require().Len(response.Chats(), 1)

	ourGroupChat := response.Chats()[0]

	err = bob1.SaveChat(ourGroupChat)
	s.NoError(err)

	// Create a one-to-one chat
	alice := s.newMessenger()
	defer TearDownMessenger(&s.Suite, alice)

	ourOneOneChat := CreateOneToOneChat("Our 1TO1", &alice.identity.PublicKey, alice.getTimesource())
	err = bob1.SaveChat(ourOneOneChat)
	s.Require().NoError(err)

	// -------------------- BACKUP --------------------
	// Backup
	marshalledBackup, err := bob1.ExportBackup()
	s.Require().NoError(err)

	// Import the backup file and process it
	err = bob2.ImportBackup(marshalledBackup)
	s.Require().NoError(err)

	// -------------------- VALIDATE BACKUP --------------------
	// Validate contacts on bob2
	s.Require().Len(bob2.Contacts(), 2)

	// Validate communities on bob2
	communities, err = bob2.JoinedCommunities()
	s.Require().NoError(err)
	s.Require().Len(communities, 1)

	// Validate chats on bob2
	// Group chat
	chat, ok := bob2.allChats.Load(ourGroupChat.ID)
	s.Require().True(ok)
	s.Require().Equal(ourGroupChat.Name, chat.Name)

	// One on one chat
	chat, ok = bob2.allChats.Load(ourOneOneChat.ID)
	s.Require().True(ok)
	s.Require().True(chat.Active)
	s.Require().Equal("", chat.Name)
}
