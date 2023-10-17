package protocol

import (
	"context"
	"errors"
	"image"
	"image/png"
	"os"
	"testing"

	userimage "github.com/status-im/status-go/images"
	"github.com/status-im/status-go/services/browsers"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/tt"
)

const statusChatID = "status"
const removedChatID = "deactivated"

func TestMessengerInstallationSuite(t *testing.T) {
	suite.Run(t, new(MessengerInstallationSuite))
}

type MessengerInstallationSuite struct {
	MessengerBaseTestSuite
}

func (s *MessengerInstallationSuite) TestReceiveInstallation() {
	theirMessenger, err := newMessengerWithKey(s.shh, s.privateKey, s.logger, nil)
	s.Require().NoError(err)

	err = theirMessenger.SetInstallationMetadata(theirMessenger.installationID, &multidevice.InstallationMetadata{
		Name:       "their-name",
		DeviceType: "their-device-type",
	})
	s.Require().NoError(err)
	response, err := theirMessenger.SendPairInstallation(context.Background(), nil)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Chats(), 1)
	s.Require().False(response.Chats()[0].Active)

	// Wait for the message to reach its destination
	response, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Installations) > 0 },
		"installation not received",
	)

	s.Require().NoError(err)
	actualInstallation := response.Installations[0]
	s.Require().Equal(theirMessenger.installationID, actualInstallation.ID)
	s.Require().NotNil(actualInstallation.InstallationMetadata)
	s.Require().Equal("their-name", actualInstallation.InstallationMetadata.Name)
	s.Require().Equal("their-device-type", actualInstallation.InstallationMetadata.DeviceType)

	err = s.m.EnableInstallation(theirMessenger.installationID)
	s.Require().NoError(err)

	contactKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	contact, err := BuildContactFromPublicKey(&contactKey.PublicKey)
	s.Require().NoError(err)
	response, err = s.m.AddContact(context.Background(), &requests.AddContact{ID: contact.ID})
	s.Require().NoError(err)

	s.Require().Len(response.Contacts, 1)
	s.Require().Equal(response.Contacts[0].ID, contact.ID)

	// Wait for the message to reach its destination
	response, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool { return len(r.Contacts) == 1 && r.Contacts[0].ID == contact.ID },
		"contact not received",
	)
	s.Require().NoError(err)

	actualContact := response.Contacts[0]
	s.Require().Equal(contact.ID, actualContact.ID)
	s.Require().True(actualContact.added())

	// Simulate update from contact
	contact.LastUpdated = 10
	contact.DisplayName = "display-name"

	s.Require().NoError(s.m.persistence.SaveContacts([]*Contact{contact}))
	// Trigger syncing of contact
	err = s.m.syncContact(context.Background(), contact, s.m.dispatchMessage)
	s.Require().NoError(err)

	// Wait for the message to reach its destination
	_, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) == 1 &&
				r.Contacts[0].ID == contact.ID &&
				// Make sure lastupdated is **not** synced
				actualContact.LastUpdated == 0 &&
				r.Contacts[0].DisplayName == "display-name"
		},
		"contact not received",
	)
	s.Require().NoError(err)

	chat := CreatePublicChat(statusChatID, s.m.transport)
	err = s.m.SaveChat(chat)
	s.Require().NoError(err)

	response, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool { return len(r.Chats()) > 0 },
		"sync chat not received",
	)

	s.Require().NoError(err)

	actualChat := response.Chats()[0]
	s.Require().Equal(statusChatID, actualChat.ID)
	s.Require().True(actualChat.Active)
	s.Require().NoError(theirMessenger.Shutdown())
}

func (s *MessengerInstallationSuite) TestSyncInstallation() {

	// add contact
	contactKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	contact, err := BuildContactFromPublicKey(&contactKey.PublicKey)
	s.Require().NoError(err)

	// mock added as mutual contact
	contact.LastUpdated = 1
	contact.ContactRequestReceived(1)
	s.m.allContacts.Store(contact.ID, contact)

	contact.LocalNickname = "Test Nickname"
	_, err = s.m.AddContact(context.Background(), &requests.AddContact{ID: contact.ID})
	s.Require().NoError(err)
	_, err = s.m.SetContactLocalNickname(&requests.SetContactLocalNickname{ID: types.Hex2Bytes(contact.ID), Nickname: contact.LocalNickname})
	s.Require().NoError(err)

	//add bookmark
	bookmark := browsers.Bookmark{
		Name:    "status official site",
		URL:     "https://status.im",
		Removed: false,
	}
	_, err = s.m.browserDatabase.StoreBookmark(bookmark)
	s.Require().NoError(err)

	// add chat
	chat := CreatePublicChat(statusChatID, s.m.transport)
	err = s.m.SaveChat(chat)
	s.Require().NoError(err)

	// Create group chat
	response, err := s.m.CreateGroupChatWithMembers(context.Background(), "group", []string{})
	s.NoError(err)
	s.Require().Len(response.Chats(), 1)

	ourGroupChat := response.Chats()[0]

	err = s.m.SaveChat(ourGroupChat)
	s.NoError(err)

	// Generate test image bigger than BannerDim
	testImage := image.NewRGBA(image.Rect(0, 0, 20, 10))

	tmpTestFilePath := s.T().TempDir() + "/test.png"
	file, err := os.Create(tmpTestFilePath)
	s.NoError(err)
	defer file.Close()

	err = png.Encode(file, testImage)
	s.Require().NoError(err)

	groupImg := userimage.CroppedImage{
		ImagePath: tmpTestFilePath,
		X:         1,
		Y:         1,
		Width:     10,
		Height:    5,
	}

	// Add image to chat
	response, err = s.m.EditGroupChat(context.Background(), ourGroupChat.ID, "test_admin_group", "#FF00FF", groupImg)
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Equal("test_admin_group", response.Chats()[0].Name)
	s.Require().Equal("#FF00FF", response.Chats()[0].Color)
	ourGroupChat = response.Chats()[0]

	// Create second group chat and deactivate it
	response, err = s.m.CreateGroupChatWithMembers(context.Background(), "deactivated-group", []string{})
	s.NoError(err)
	s.Require().Len(response.Chats(), 1)

	ourDeactivatedGroupChat := response.Chats()[0]
	err = s.m.SaveChat(ourDeactivatedGroupChat)
	s.NoError(err)
	_, err = s.m.deactivateChat(ourDeactivatedGroupChat.ID, 0, true, true)
	s.NoError(err)

	// Create Alice for the 1-1 chat
	alice := s.newMessenger()
	_, err = alice.Start()
	s.Require().NoError(err)
	defer alice.Shutdown() // nolint: errcheck

	// Create 1-1 chat
	ourOneOneChat := CreateOneToOneChat("Our 1TO1", &alice.identity.PublicKey, alice.transport)
	err = s.m.SaveChat(ourOneOneChat)
	s.Require().NoError(err)

	// add and deactivate chat
	chat2 := CreatePublicChat(removedChatID, s.m.transport)
	chat2.DeletedAtClockValue = 1
	err = s.m.SaveChat(chat2)
	s.Require().NoError(err)
	_, err = s.m.deactivateChat(removedChatID, 0, true, true)
	s.Require().NoError(err)

	// pair
	theirMessenger, err := newMessengerWithKey(s.shh, s.privateKey, s.logger, nil)
	s.Require().NoError(err)
	err = theirMessenger.SaveChat(chat2)
	s.Require().NoError(err)

	err = theirMessenger.SetInstallationMetadata(theirMessenger.installationID, &multidevice.InstallationMetadata{
		Name:       "their-name",
		DeviceType: "their-device-type",
	})
	s.Require().NoError(err)
	response, err = theirMessenger.SendPairInstallation(context.Background(), nil)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Chats(), 1)
	s.Require().False(response.Chats()[0].Active)

	// Wait for the message to reach its destination
	response, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Installations) > 0 },
		"installation not received",
	)

	s.Require().NoError(err)
	actualInstallation := response.Installations[0]
	s.Require().Equal(theirMessenger.installationID, actualInstallation.ID)
	s.Require().NotNil(actualInstallation.InstallationMetadata)
	s.Require().Equal("their-name", actualInstallation.InstallationMetadata.Name)
	s.Require().Equal("their-device-type", actualInstallation.InstallationMetadata.DeviceType)

	err = s.m.EnableInstallation(theirMessenger.installationID)
	s.Require().NoError(err)

	// sync
	err = s.m.SyncDevices(context.Background(), "ens-name", "profile-image", nil)
	s.Require().NoError(err)

	var allChats []*Chat
	var actualContact *Contact
	var bookmarks []*browsers.Bookmark
	// Wait for the message to reach its destination
	err = tt.RetryWithBackOff(func() error {
		var err error
		response, err = theirMessenger.RetrieveAll()
		if err != nil {
			return err
		}

		allChats = append(allChats, response.Chats()...)
		if len(response.Contacts) == 1 {
			actualContact = response.Contacts[0]
		}
		bookmarks = append(bookmarks, response.GetBookmarks()...)

		if len(allChats) >= 5 && actualContact != nil && len(bookmarks) >= 1 {
			return nil
		}

		return errors.New("not received all chats & contacts & bookmarks yet")

	})

	s.Require().NoError(err)

	var statusChat *Chat
	var groupChat *Chat
	var removedGroupChat *Chat
	var oneToOneChat *Chat
	var removedChat *Chat
	for _, c := range allChats {
		if c.ID == statusChatID {
			statusChat = c
		}
		if c.ID == ourGroupChat.ID {
			groupChat = c
		}
		if c.ID == ourDeactivatedGroupChat.ID {
			removedGroupChat = c
		}
		if c.ID == ourOneOneChat.ID {
			oneToOneChat = c
		}
		if c.ID == removedChatID {
			removedChat = c
		}
	}

	s.Require().NotNil(statusChat)
	s.Require().NotNil(groupChat)
	s.Require().NotNil(removedGroupChat)
	s.Require().NotNil(oneToOneChat)

	s.Require().Equal(ourGroupChat.Name, groupChat.Name)
	s.Require().True(ourGroupChat.Active)

	s.Require().Equal(ourDeactivatedGroupChat.Name, removedGroupChat.Name)
	s.Require().False(removedGroupChat.Active)

	s.Require().Equal("", oneToOneChat.Name) // We set 1-1 chat names to "" because the name is not good
	s.Require().True(oneToOneChat.Active)

	s.Require().True(actualContact.added())
	s.Require().Equal("Test Nickname", actualContact.LocalNickname)
	s.Require().True(actualContact.hasAddedUs())
	s.Require().True(actualContact.mutual())

	bookmarks, err = theirMessenger.browserDatabase.GetBookmarks()
	s.Require().NoError(err)
	s.Require().Equal(1, len(bookmarks))

	s.Require().NoError(theirMessenger.Shutdown())

	s.Require().NotNil(removedChat)
	s.Require().False(removedChat.Active)

}

func (s *MessengerInstallationSuite) TestSyncInstallationNewMessages() {

	bob1 := s.m
	// pair
	bob2, err := newMessengerWithKey(s.shh, s.privateKey, s.logger, nil)
	s.Require().NoError(err)
	alice := s.newMessenger()

	err = bob2.SetInstallationMetadata(bob2.installationID, &multidevice.InstallationMetadata{
		Name:       "their-name",
		DeviceType: "their-device-type",
	})
	s.Require().NoError(err)
	response, err := bob2.SendPairInstallation(context.Background(), nil)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Chats(), 1)
	s.Require().False(response.Chats()[0].Active)

	// Wait for the message to reach its destination
	response, err = WaitOnMessengerResponse(
		bob1,
		func(r *MessengerResponse) bool { return len(r.Installations) > 0 },
		"installation not received",
	)

	s.Require().NoError(err)
	actualInstallation := response.Installations[0]
	s.Require().Equal(bob2.installationID, actualInstallation.ID)
	err = bob1.EnableInstallation(bob2.installationID)
	s.Require().NoError(err)

	// send a message from bob1 to alice, it should be received on both bob1 and bob2

	alicePkString := types.EncodeHex(crypto.FromECDSAPub(&alice.identity.PublicKey))
	chat := CreateOneToOneChat(alicePkString, &alice.identity.PublicKey, bob1.transport)
	s.Require().NoError(bob1.SaveChat(chat))

	inputMessage := buildTestMessage(*chat)
	_, err = s.m.SendChatMessage(context.Background(), inputMessage)
	s.Require().NoError(err)

	// Wait for the message to reach its destination
	_, err = WaitOnMessengerResponse(
		bob2,
		func(r *MessengerResponse) bool { return len(r.Messages()) > 0 },
		"message not received",
	)
	s.Require().NoError(err)
	s.Require().NoError(bob2.Shutdown())
	s.Require().NoError(alice.Shutdown())
}
