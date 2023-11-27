package protocol

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/server"
)

func TestMessengerDeleteMessageSuite(t *testing.T) {
	suite.Run(t, new(MessengerDeleteMessageSuite))
}

type MessengerDeleteMessageSuite struct {
	MessengerBaseTestSuite
}

func (s *MessengerDeleteMessageSuite) TestDeleteMessage() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	defer TearDownMessenger(&s.Suite, theirMessenger)

	theirChat := CreateOneToOneChat("Their 1TO1", &s.privateKey.PublicKey, s.m.transport)
	err = theirMessenger.SaveChat(theirChat)
	s.Require().NoError(err)

	ourChat := CreateOneToOneChat("Our 1TO1", &theirMessenger.identity.PublicKey, s.m.transport)
	err = s.m.SaveChat(ourChat)
	s.Require().NoError(err)

	inputMessage := buildTestMessage(*theirChat)
	sendResponse, err := theirMessenger.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)
	s.Require().Len(sendResponse.Messages(), 1)

	messageID := sendResponse.Messages()[0].ID

	response, err := WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.messages) > 0 },
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)

	ogMessage := sendResponse.Messages()[0]

	sendResponse, err = theirMessenger.DeleteMessageAndSend(context.Background(), ogMessage.ID)

	s.Require().NoError(err)
	s.Require().Len(sendResponse.Messages(), 0)
	s.Require().Len(sendResponse.RemovedMessages(), 1)
	s.Require().Equal(messageID, sendResponse.RemovedMessages()[0].MessageID)
	s.Require().Equal(sendResponse.RemovedMessages()[0].DeletedBy, "")
	s.Require().Len(sendResponse.Chats(), 1)
	// LastMessage is marked as deleted
	lastRemovedMessage := sendResponse.Chats()[0].LastMessage
	s.Require().Equal(lastRemovedMessage.ID, messageID)
	s.Require().Equal(lastRemovedMessage.Deleted, true)

	// Main instance user attempts to delete the message it received from theirMessenger
	_, err = s.m.DeleteMessageAndSend(context.Background(), ogMessage.ID)

	s.Require().ErrorContains(err, "can't find chat")
}

func (s *MessengerDeleteMessageSuite) TestDeleteMessagePreviousLastMessage() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	defer TearDownMessenger(&s.Suite, theirMessenger)

	theirChat := CreateOneToOneChat("Their 1TO1", &s.privateKey.PublicKey, s.m.transport)
	err = theirMessenger.SaveChat(theirChat)
	s.Require().NoError(err)

	ourChat := CreateOneToOneChat("Our 1TO1", &theirMessenger.identity.PublicKey, s.m.transport)
	err = s.m.SaveChat(ourChat)
	s.Require().NoError(err)

	inputMessage1 := buildTestMessage(*theirChat)
	sendResponse, err := theirMessenger.SendChatMessage(context.Background(), inputMessage1)
	s.NoError(err)
	s.Require().Len(sendResponse.Messages(), 1)

	inputMessage2 := buildTestMessage(*theirChat)
	sendResponse, err = theirMessenger.SendChatMessage(context.Background(), inputMessage2)
	s.NoError(err)
	s.Require().Len(sendResponse.Messages(), 1)

	messageID := sendResponse.Messages()[0].ID

	response, err := WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.messages) > 0 },
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 2)

	ogMessage := sendResponse.Messages()[0]

	sendResponse, err = theirMessenger.DeleteMessageAndSend(context.Background(), ogMessage.ID)

	s.Require().NoError(err)
	s.Require().Len(sendResponse.Messages(), 0)
	s.Require().Len(sendResponse.RemovedMessages(), 1)
	s.Require().Equal(messageID, sendResponse.RemovedMessages()[0].MessageID)
	s.Require().Len(sendResponse.Chats(), 1)
	// LastMessage is updated and marked as deleted
	s.Require().NotNil(sendResponse.Chats()[0].LastMessage)
	s.Require().Equal(inputMessage2.ID, sendResponse.Chats()[0].LastMessage.ID)
	s.Require().Equal(sendResponse.Chats()[0].LastMessage.Deleted, true)

}

func (s *MessengerDeleteMessageSuite) TestDeleteWrongMessageType() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	defer TearDownMessenger(&s.Suite, theirMessenger)

	theirChat := CreateOneToOneChat("Their 1TO1", &s.privateKey.PublicKey, s.m.transport)
	err = theirMessenger.SaveChat(theirChat)
	s.Require().NoError(err)

	ourChat := CreateOneToOneChat("Our 1TO1", &theirMessenger.identity.PublicKey, s.m.transport)
	err = s.m.SaveChat(ourChat)
	s.Require().NoError(err)

	inputMessage := buildTestGapMessage(*theirChat)
	sendResponse, err := theirMessenger.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)
	s.Require().Len(sendResponse.Messages(), 1)

	ogMessage := sendResponse.Messages()[0]

	// Delete should not work
	_, err = theirMessenger.DeleteMessageAndSend(context.Background(), ogMessage.ID)

	s.Require().Equal(ErrInvalidDeleteTypeAuthor, err)
}

// TODO fix activity center notifications not being deleted when a message is deleted

func (s *MessengerDeleteMessageSuite) TestDeleteMessageFirstThenMessage() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	defer TearDownMessenger(&s.Suite, theirMessenger)

	theirChat := CreateOneToOneChat("Their 1TO1", &s.privateKey.PublicKey, s.m.transport)
	err = theirMessenger.SaveChat(theirChat)
	s.Require().NoError(err)

	contact, err := BuildContactFromPublicKey(&theirMessenger.identity.PublicKey)
	s.Require().NoError(err)

	ourChat := CreateOneToOneChat("Our 1TO1", &theirMessenger.identity.PublicKey, s.m.transport)
	err = s.m.SaveChat(ourChat)
	s.Require().NoError(err)
	messageID := "message-id"

	inputMessage := buildTestMessage(*theirChat)
	inputMessage.Clock = 1
	deleteMessage := &DeleteMessage{
		DeleteMessage: &protobuf.DeleteMessage{
			Clock:       2,
			MessageType: protobuf.MessageType_ONE_TO_ONE,
			MessageId:   messageID,
			ChatId:      theirChat.ID,
		},
		From: common.PubkeyToHex(&theirMessenger.identity.PublicKey),
	}

	state := &ReceivedMessageState{
		Response: &MessengerResponse{},
	}

	// Handle Delete first
	err = s.m.handleDeleteMessage(state, deleteMessage)
	s.Require().NoError(err)

	// // Handle chat message
	state = &ReceivedMessageState{
		Response: &MessengerResponse{},
		CurrentMessageState: &CurrentMessageState{
			MessageID:        messageID,
			WhisperTimestamp: s.m.getTimesource().GetCurrentTime(),
			Contact:          contact,
			PublicKey:        &theirMessenger.identity.PublicKey,
		},
	}
	err = s.m.HandleChatMessage(state, inputMessage.ChatMessage, nil, false)
	s.Require().NoError(err)
	s.Require().Len(state.Response.Messages(), 0) // Message should not be added to response
	s.Require().Len(state.Response.RemovedMessages(), 0)
	s.Require().Nil(state.Response.Chats()[0].LastMessage)
}

func (s *MessengerDeleteMessageSuite) TestDeleteImageMessage() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	defer TearDownMessenger(&s.Suite, theirMessenger)

	theirChat := CreateOneToOneChat("Their 1TO1", &s.privateKey.PublicKey, s.m.transport)
	err = theirMessenger.SaveChat(theirChat)
	s.Require().NoError(err)

	ourChat := CreateOneToOneChat("Our 1TO1", &theirMessenger.identity.PublicKey, s.m.transport)
	err = s.m.SaveChat(ourChat)
	s.Require().NoError(err)

	messageCount := 3
	var album []*common.Message
	for i := 0; i < messageCount; i++ {
		image, err := buildImageWithoutAlbumIDMessage(*ourChat)
		s.NoError(err)
		album = append(album, image)
	}

	response, err := s.m.SendChatMessages(context.Background(), album)
	s.NoError(err)

	// Check that album count was the number of the images sent
	imagesCount := uint32(0)
	for _, message := range response.Messages() {
		if message.ContentType == protobuf.ChatMessage_IMAGE {
			imagesCount++
		}
	}
	for _, message := range response.Messages() {
		s.Require().NotNil(message.GetImage())
		s.Require().Equal(message.GetImage().AlbumImagesCount, imagesCount)
	}

	s.Require().Equal(messageCount, len(response.Messages()), "it returns the messages")
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), messageCount)

	response, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool { return len(r.messages) == messageCount },
		"no messages",
	)

	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), messageCount)
	for _, message := range response.Messages() {
		image := message.GetImage()
		s.Require().NotNil(image, "Message.ID=%s", message.ID)
		s.Require().Equal(image.AlbumImagesCount, imagesCount)
		s.Require().NotEmpty(image.AlbumId, "Message.ID=%s", message.ID)
	}

	firstMessageID := response.Messages()[0].ID
	sendResponse, err := s.m.DeleteMessageAndSend(context.Background(), firstMessageID)

	s.Require().NoError(err)
	s.Require().Len(sendResponse.Messages(), 0)
	s.Require().Len(sendResponse.RemovedMessages(), 3)
	s.Require().Equal(sendResponse.RemovedMessages()[0].DeletedBy, "")
	s.Require().Len(sendResponse.Chats(), 1)

	// LastMessage marked as deleted
	s.Require().Equal(sendResponse.Chats()[0].LastMessage.ID, album[2].ID)
	s.Require().Equal(sendResponse.Chats()[0].LastMessage.Deleted, true)

	// Main instance user attempts to delete the message it received from theirMessenger
	_, err = theirMessenger.DeleteMessageAndSend(context.Background(), firstMessageID)

	s.Require().ErrorContains(err, "can't find chat")
}

func (s *MessengerDeleteMessageSuite) TestDeleteImageMessageFirstThenMessage() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	defer TearDownMessenger(&s.Suite, theirMessenger)

	theirChat := CreateOneToOneChat("Their 1TO1", &s.privateKey.PublicKey, s.m.transport)
	err = theirMessenger.SaveChat(theirChat)
	s.Require().NoError(err)

	contact, err := BuildContactFromPublicKey(&theirMessenger.identity.PublicKey)
	s.Require().NoError(err)

	ourChat := CreateOneToOneChat("Our 1TO1", &theirMessenger.identity.PublicKey, s.m.transport)
	err = s.m.SaveChat(ourChat)
	s.Require().NoError(err)
	messageID1 := "message-id1"
	messageID2 := "message-id2"
	albumID := "album-id1"

	messageCount := 2
	var album []*common.Message
	for i := 0; i < messageCount; i++ {
		image, err := buildImageWithAlbumIDMessage(*ourChat, albumID)
		image.Clock = 1
		s.NoError(err)
		album = append(album, image)
	}

	deleteMessage := &DeleteMessage{
		DeleteMessage: &protobuf.DeleteMessage{
			Clock:       2,
			MessageType: protobuf.MessageType_ONE_TO_ONE,
			MessageId:   messageID1,
			ChatId:      theirChat.ID,
		},
		From: common.PubkeyToHex(&theirMessenger.identity.PublicKey),
	}

	state := &ReceivedMessageState{
		Response: &MessengerResponse{},
	}

	// Handle Delete first
	err = s.m.handleDeleteMessage(state, deleteMessage)
	s.Require().NoError(err)

	// Handle first image message
	state = &ReceivedMessageState{
		Response: &MessengerResponse{},
		CurrentMessageState: &CurrentMessageState{
			MessageID:        messageID1,
			WhisperTimestamp: s.m.getTimesource().GetCurrentTime(),
			Contact:          contact,
			PublicKey:        &theirMessenger.identity.PublicKey,
		},
	}
	err = s.m.HandleChatMessage(state, album[0].ChatMessage, nil, false)
	s.Require().NoError(err)
	s.Require().Len(state.Response.Messages(), 0) // Message should not be added to response
	s.Require().Len(state.Response.RemovedMessages(), 0)
	s.Require().Nil(state.Response.Chats()[0].LastMessage)

	// Handle second  image message
	state = &ReceivedMessageState{
		Response: &MessengerResponse{},
		CurrentMessageState: &CurrentMessageState{
			MessageID:        messageID2,
			WhisperTimestamp: s.m.getTimesource().GetCurrentTime(),
			Contact:          contact,
			PublicKey:        &theirMessenger.identity.PublicKey,
		},
	}
	err = s.m.HandleChatMessage(state, album[1].ChatMessage, nil, false)
	s.Require().NoError(err)
	s.Require().Len(state.Response.Messages(), 0) // Message should not be added to response even if we didn't delete that ID
	s.Require().Len(state.Response.RemovedMessages(), 0)
	s.Require().Nil(state.Response.Chats()[0].LastMessage)
}

func (s *MessengerDeleteMessageSuite) TestDeleteMessageWithAMention() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	defer TearDownMessenger(&s.Suite, theirMessenger)

	theirChat := CreateOneToOneChat("Their 1TO1", &s.privateKey.PublicKey, s.m.transport)
	err = theirMessenger.SaveChat(theirChat)
	s.Require().NoError(err)

	ourChat := CreateOneToOneChat("Our 1TO1", &theirMessenger.identity.PublicKey, s.m.transport)
	err = s.m.SaveChat(ourChat)
	s.Require().NoError(err)

	inputMessage := buildTestMessage(*theirChat)
	inputMessage.Text = "text with a mention @" + common.PubkeyToHex(&s.privateKey.PublicKey)
	sendResponse, err := theirMessenger.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)
	s.Require().Len(sendResponse.Messages(), 1)

	messageID := sendResponse.Messages()[0].ID

	response, err := WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.messages) == 1 },
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)
	// Receiver (us) is mentioned
	s.Require().Equal(int(response.Chats()[0].UnviewedMessagesCount), 1)
	s.Require().Equal(int(response.Chats()[0].UnviewedMentionsCount), 1)
	s.Require().True(response.Messages()[0].Mentioned)

	deleteMessage := &DeleteMessage{
		DeleteMessage: &protobuf.DeleteMessage{
			Clock:       2,
			MessageType: protobuf.MessageType_ONE_TO_ONE,
			MessageId:   messageID,
			ChatId:      theirChat.ID,
		},
		From: common.PubkeyToHex(&theirMessenger.identity.PublicKey),
	}

	state := &ReceivedMessageState{
		Response: &MessengerResponse{},
	}

	// Handle Delete first
	err = s.m.handleDeleteMessage(state, deleteMessage)

	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)
	// Receiver (us) is  no longer mentioned
	s.Require().Equal(int(state.Response.Chats()[0].UnviewedMessagesCount), 0)
	s.Require().Equal(int(state.Response.Chats()[0].UnviewedMentionsCount), 0)
}

// This test makes sure the UnviewMessageCount doesn't go below 0 in a very rare case where the Chat could be marked
// as read but the message still unseen (Seen == false)
func (s *MessengerDeleteMessageSuite) TestDeleteMessageAndChatIsAlreadyRead() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	defer TearDownMessenger(&s.Suite, theirMessenger)

	theirChat := CreateOneToOneChat("Their 1TO1", &s.privateKey.PublicKey, s.m.transport)
	err = theirMessenger.SaveChat(theirChat)
	s.Require().NoError(err)

	ourChat := CreateOneToOneChat("Our 1TO1", &theirMessenger.identity.PublicKey, s.m.transport)
	err = s.m.SaveChat(ourChat)
	s.Require().NoError(err)

	inputMessage := buildTestMessage(*theirChat)
	sendResponse, err := theirMessenger.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)
	s.Require().Len(sendResponse.Messages(), 1)

	response, err := WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.messages) == 1 },
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Equal(response.Chats()[0].UnviewedMessagesCount, uint(1))
	s.Require().Len(response.Messages(), 1)

	// Force UnviewedMessagesCount to 0 to test if the uint validation is done correctly
	ourChat.UnviewedMessagesCount = 0
	err = s.m.saveChat(ourChat)

	s.Require().NoError(err)

	ogMessage := sendResponse.Messages()[0]

	deleteMessage := &DeleteMessage{
		DeleteMessage: &protobuf.DeleteMessage{
			Clock:       2,
			MessageType: protobuf.MessageType_ONE_TO_ONE,
			MessageId:   ogMessage.ID,
			ChatId:      theirChat.ID,
		},
		From: common.PubkeyToHex(&theirMessenger.identity.PublicKey),
	}

	state := &ReceivedMessageState{
		Response: &MessengerResponse{},
	}

	// Handle Delete first
	err = s.m.handleDeleteMessage(state, deleteMessage)

	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)
	// Receiver (us) no longer has unread messages and it's not negative
	s.Require().Equal(0, int(state.Response.Chats()[0].UnviewedMessagesCount))
}

func (s *MessengerDeleteMessageSuite) TestDeleteMessageReplyToImage() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	defer TearDownMessenger(&s.Suite, theirMessenger)

	theirChat := CreateOneToOneChat("Their 1TO1", &s.privateKey.PublicKey, s.m.transport)
	err = theirMessenger.SaveChat(theirChat)
	s.Require().NoError(err)

	ourChat := CreateOneToOneChat("Our 1TO1", &theirMessenger.identity.PublicKey, s.m.transport)
	err = s.m.SaveChat(ourChat)
	s.Require().NoError(err)

	inputMessage := buildTestMessage(*theirChat)
	sendResponse, err := theirMessenger.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)
	s.Require().Len(sendResponse.Messages(), 1)

	response, err := WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.messages) == 1 },
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)

	ogMessage := sendResponse.Messages()[0]

	// create an http server
	mediaServer, err := server.NewMediaServer(nil, nil, nil)
	s.Require().NoError(err)
	s.Require().NotNil(mediaServer)
	s.Require().NoError(mediaServer.Start())

	theirMessenger.httpServer = mediaServer

	// We reply to our own message with an image
	imageMessage, err := buildImageWithoutAlbumIDMessage(*theirChat)
	s.NoError(err)

	imageMessage.ResponseTo = ogMessage.ID

	_, err = theirMessenger.SendChatMessages(context.Background(), []*common.Message{imageMessage})
	s.NoError(err)

	// We check that the URL is correctly returned
	sendResponse, err = theirMessenger.DeleteMessageAndSend(context.Background(), ogMessage.ID)

	s.Require().NoError(err)
	s.Require().Len(sendResponse.Messages(), 1)
	s.Require().NotEmpty(sendResponse.Messages()[0].ImageLocalURL)
}

func (s *MessengerDeleteMessageSuite) TestDeleteMessageForMeReplyToImage() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	defer TearDownMessenger(&s.Suite, theirMessenger)

	theirChat := CreateOneToOneChat("Their 1TO1", &s.privateKey.PublicKey, s.m.transport)
	err = theirMessenger.SaveChat(theirChat)
	s.Require().NoError(err)

	ourChat := CreateOneToOneChat("Our 1TO1", &theirMessenger.identity.PublicKey, s.m.transport)
	err = s.m.SaveChat(ourChat)
	s.Require().NoError(err)

	inputMessage := buildTestMessage(*theirChat)
	sendResponse, err := theirMessenger.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)
	s.Require().Len(sendResponse.Messages(), 1)

	response, err := WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.messages) == 1 },
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)

	ogMessage := sendResponse.Messages()[0]

	// create an http server
	mediaServer, err := server.NewMediaServer(nil, nil, nil)
	s.Require().NoError(err)
	s.Require().NotNil(mediaServer)
	s.Require().NoError(mediaServer.Start())

	theirMessenger.httpServer = mediaServer

	// We reply to our own message with an image
	imageMessage, err := buildImageWithoutAlbumIDMessage(*theirChat)
	s.NoError(err)

	imageMessage.ResponseTo = ogMessage.ID

	_, err = theirMessenger.SendChatMessages(context.Background(), []*common.Message{imageMessage})
	s.NoError(err)

	// We check that the URL is correctly returned
	sendResponse, err = theirMessenger.DeleteMessageForMeAndSync(context.Background(), theirChat.ID, ogMessage.ID)

	s.Require().NoError(err)
	messages := sendResponse.Messages()
	s.Require().Len(messages, 2)

	var deletedMessage, replyMessage *common.Message
	if messages[0].ID == ogMessage.ID {
		deletedMessage = messages[0]
		replyMessage = messages[1]
	} else {
		deletedMessage = messages[1]
		replyMessage = messages[0]
	}

	s.Require().True(deletedMessage.DeletedForMe)
	s.Require().NotEmpty(replyMessage.ImageLocalURL)
}
