package protocol

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/crypto"
	"github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
)

func TestMessengerLocalBackupSuite(t *testing.T) {
	suite.Run(t, new(MessengerLocalBackupSuite))
}

type MessengerLocalBackupSuite struct {
	MessengerBaseTestSuite
}

func makeMutualContacts(lhs *Messenger, rhs *Messenger) error {
	if err := makeMutualContact(lhs, &rhs.identity.PublicKey); err != nil {
		return err
	}
	return makeMutualContact(rhs, &lhs.identity.PublicKey)
}

func (s *MessengerLocalBackupSuite) TestLocalBackup() {
	// Create bob1
	bob1 := s.anotherMessenger()
	defer TearDownMessenger(&s.Suite, bob1)

	// Create bob2
	bob2 := s.anotherMessenger()
	defer TearDownMessenger(&s.Suite, bob2)

	// Enable message backup on both accounts
	err := bob1.settings.SaveSetting(settings.MessagesBackupEnabled.GetReactName(), true)
	s.Require().NoError(err)

	err = bob2.settings.SaveSetting(settings.MessagesBackupEnabled.GetReactName(), true)
	s.Require().NoError(err)

	ctx := context.Background()

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

	// Send message on community
	chatID := response.Chats()[0].ID
	inputMessage := common.NewMessage()
	inputMessage.ChatId = chatID
	inputMessage.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	inputMessage.Text = "some text"

	_, err = bob1.SendChatMessage(ctx, inputMessage)
	s.Require().NoError(err)

	// Pin message
	pinMessage := common.NewPinMessage()
	pinMessage.ChatId = chatID
	pinMessage.MessageId = inputMessage.ID
	pinMessage.Pinned = true
	sendResponse, err := bob1.SendPinMessage(ctx, pinMessage)
	s.Require().NoError(err)
	s.Require().Len(sendResponse.PinMessages(), 1)

	// Send markdown message
	mdMessage := common.NewMessage()
	mdMessage.ChatId = chatID
	mdMessage.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	mdMessage.Text = "some *markdown* text"

	_, err = bob1.SendChatMessage(ctx, mdMessage)
	s.Require().NoError(err)

	// Send image on community
	file, err := os.Open("../_assets/tests/test.jpg")
	s.Require().NoError(err)
	defer file.Close()

	payload, err := io.ReadAll(file)
	s.Require().NoError(err)

	imageMessage := common.NewMessage()
	imageMessage.ChatId = chatID
	imageMessage.ContentType = protobuf.ChatMessage_IMAGE

	image := protobuf.ImageMessage{
		Payload: payload,
		Format:  protobuf.ImageFormat_JPEG,
		Width:   1200,
		Height:  1000,
		AlbumId: "",
	}
	imageMessage.Payload = &protobuf.ChatMessage_Image{Image: &image}
	imageMessage.Text = "some image"

	_, err = bob1.SendChatMessage(ctx, imageMessage)
	s.Require().NoError(err)

	// Send sticker on community
	stickerMessage := common.NewMessage()
	stickerMessage.ChatId = chatID
	stickerMessage.ContentType = protobuf.ChatMessage_STICKER
	stickerMessage.Text = "some sticker"
	stickerMessage.Payload = &protobuf.ChatMessage_Sticker{
		Sticker: &protobuf.StickerMessage{
			Pack: 1,
			Hash: "some-hash",
		},
	}
	_, err = bob1.SendChatMessage(ctx, stickerMessage)
	s.Require().NoError(err)

	// Send emoji on community
	emojiMessage := common.NewMessage()
	emojiMessage.ChatId = chatID
	emojiMessage.ContentType = protobuf.ChatMessage_EMOJI
	emojiMessage.Text = ":+1:"
	_, err = bob1.SendChatMessage(ctx, emojiMessage)
	s.Require().NoError(err)

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

	makeMutualContacts(bob1, alice)

	aliceContact := bob1.GetContactByID(alice.selfContact.ID)
	err = bob1.persistence.SaveContact(aliceContact, nil)
	s.Require().NoError(err)

	ourOneOneChat := CreateOneToOneChat("Our 1TO1", &alice.identity.PublicKey, alice.getTimesource())
	err = bob1.SaveChat(ourOneOneChat)
	s.Require().NoError(err)

	theirChat := CreateOneToOneChat("Their 1TO1", &bob1.identity.PublicKey, bob1.getTimesource())
	err = alice.SaveChat(theirChat)
	s.Require().NoError(err)

	// Send transaction command to Alice
	transactionMessage := common.NewMessage()
	transactionMessage.ChatId = ourOneOneChat.ID
	transactionMessage.ContentType = protobuf.ChatMessage_TRANSACTION_COMMAND
	transactionMessage.Text = "some transaction"
	_, err = bob1.SendChatMessage(ctx, transactionMessage)
	s.Require().NoError(err)

	// Alice sends a message to bob1
	inputMessage = buildTestMessage(*theirChat)
	inputMessage.Text = "some text from alice"
	sendResponse, err = alice.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)
	s.Require().Len(sendResponse.Messages(), 1)

	response, err = WaitOnMessengerResponse(
		bob1,
		func(r *MessengerResponse) bool {
			return len(r.messages) > 0 && r.Messages()[0].Text == "some text from alice"
		},
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)

	// Validate contacts on bob1
	contact1 := bob1.GetContactByID(contactID1)
	s.Require().NotNil(contact1)
	s.Require().Equal(ContactRequestStateSent, contact1.ContactRequestLocalState)
	s.Require().Equal(ContactRequestStateNone, contact1.ContactRequestRemoteState)
	s.Require().True(contact1.added())

	contact2 := bob1.GetContactByID(contactID2)
	s.Require().NotNil(contact2)
	s.Require().Equal(ContactRequestStateSent, contact2.ContactRequestLocalState)
	s.Require().Equal(ContactRequestStateNone, contact2.ContactRequestRemoteState)
	s.Require().True(contact2.added())

	aliceContact = bob1.GetContactByID(alice.selfContact.ID)
	s.Require().NotNil(aliceContact)
	s.Require().Equal(ContactRequestStateSent, aliceContact.ContactRequestLocalState)
	s.Require().Equal(ContactRequestStateReceived, aliceContact.ContactRequestRemoteState)
	s.Require().True(aliceContact.added())

	// Check that bob2 has no contacts
	s.Require().Len(bob2.Contacts(), 0)

	// -------------------- BACKUP --------------------
	// Backup
	marshalledBackup, err := bob1.ExportBackup()
	s.Require().NoError(err)

	// Import the backup file and process it
	err = bob2.ImportBackup(marshalledBackup)
	s.Require().NoError(err)

	// -------------------- VALIDATE BACKUP --------------------
	// Validate contacts on bob2
	contact1 = bob2.GetContactByID(contactID1)
	s.Require().NotNil(contact1)
	s.Require().Equal(ContactRequestStateSent, contact1.ContactRequestLocalState)
	s.Require().Equal(ContactRequestStateNone, contact1.ContactRequestRemoteState)
	s.Require().True(contact1.added())

	contact2 = bob2.GetContactByID(contactID2)
	s.Require().NotNil(contact2)
	s.Require().Equal(ContactRequestStateSent, contact2.ContactRequestLocalState)
	s.Require().Equal(ContactRequestStateNone, contact2.ContactRequestRemoteState)
	s.Require().True(contact2.added())

	aliceContact = bob2.GetContactByID(alice.selfContact.ID)
	s.Require().NotNil(aliceContact)
	s.Require().Equal(ContactRequestStateSent, aliceContact.ContactRequestLocalState)
	s.Require().Equal(ContactRequestStateReceived, aliceContact.ContactRequestRemoteState)
	s.Require().True(aliceContact.added())

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

	// Validate messages
	messages, err := bob2.persistence.AllMessagesForBackup()
	s.Require().NoError(err)
	s.Require().Len(messages, 15)

	// Build a map for easier assertions
	messageMap := make(map[string]*protobuf.BackedUpMessage)
	for _, msg := range messages {
		if msg.ContentType == int64(protobuf.ChatMessage_SYSTEM_MESSAGE_PINNED_MESSAGE) && msg.Text == "" {
			// For system pinned message, Text is empty so we use a custom key
			messageMap["systemPin"] = msg
			continue
		}
		messageMap[msg.Text] = msg
	}

	// Assert each message type exists and has expected properties
	textMsg, ok := messageMap["some text"]
	s.Require().True(ok)
	s.Require().Equal(int64(protobuf.ChatMessage_TEXT_PLAIN), textMsg.ContentType)
	s.Require().Equal(bob2.selfContact.ID, textMsg.PinnedBy)

	mdMsg, ok := messageMap["some *markdown* text"]
	s.Require().True(ok)
	s.Require().Equal(int64(protobuf.ChatMessage_TEXT_PLAIN), mdMsg.ContentType)

	imageMsg, ok := messageMap["some image"]
	s.Require().True(ok)
	s.Require().Equal(int64(protobuf.ChatMessage_IMAGE), imageMsg.ContentType)

	stickerMsg, ok := messageMap["some sticker"]
	s.Require().True(ok)
	s.Require().Equal(int64(protobuf.ChatMessage_STICKER), stickerMsg.ContentType)

	emojiMsg, ok := messageMap[":+1:"]
	s.Require().True(ok)
	s.Require().Equal(int64(protobuf.ChatMessage_EMOJI), emojiMsg.ContentType)

	txMsg, ok := messageMap["some transaction"]
	s.Require().True(ok)
	s.Require().Equal(int64(protobuf.ChatMessage_TRANSACTION_COMMAND), txMsg.ContentType)

	aliceMsg, ok := messageMap["some text from alice"]
	s.Require().True(ok)
	s.Require().Equal(int64(protobuf.ChatMessage_TEXT_PLAIN), aliceMsg.ContentType)

	systemPinMsg, ok := messageMap["systemPin"]
	s.Require().True(ok)
	s.Require().Equal("", systemPinMsg.Text)

	// Validate pinned messages
	pinnedMessages, _, err := bob2.PinnedMessageByChatID(chatID, "", 10)
	s.Require().NoError(err)
	s.Require().Len(pinnedMessages, 1)
	s.Require().Equal(bob2.selfContact.ID, pinnedMessages[0].PinnedBy)

}
