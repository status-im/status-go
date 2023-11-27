package protocol

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
)

func TestMessengerSendImagesAlbumSuite(t *testing.T) {
	suite.Run(t, new(MessengerSendImagesAlbumSuite))
}

type MessengerSendImagesAlbumSuite struct {
	MessengerBaseTestSuite
}

func (s *MessengerSendImagesAlbumSuite) advertiseCommunityTo(community *communities.Community, user *Messenger) {
	advertiseCommunityTo(&s.Suite, community, s.m, user)
}

func (s *MessengerSendImagesAlbumSuite) joinCommunity(community *communities.Community, user *Messenger) {
	request := &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	joinCommunity(&s.Suite, community, s.m, user, request, "")
}

func (s *MessengerSendImagesAlbumSuite) TestAlbumImageMessagesSend() {
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

	const messageCount = 3
	var album []*common.Message

	for i := 0; i < messageCount; i++ {
		image, err := buildImageWithoutAlbumIDMessage(*ourChat)
		s.NoError(err)
		album = append(album, image)
	}

	err = s.m.SaveChat(ourChat)
	s.NoError(err)
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
}

func (s *MessengerSendImagesAlbumSuite) TestAlbumImageMessagesWithMentionSend() {
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

	const messageCount = 3
	var album []*common.Message

	for i := 0; i < messageCount; i++ {
		outgoingMessage, err := buildImageWithoutAlbumIDMessage(*ourChat)
		s.NoError(err)
		outgoingMessage.Mentioned = true
		outgoingMessage.Text = "hey @" + common.PubkeyToHex(&theirMessenger.identity.PublicKey)
		album = append(album, outgoingMessage)
	}

	err = s.m.SaveChat(ourChat)
	s.NoError(err)
	response, err := s.m.SendChatMessages(context.Background(), album)
	s.NoError(err)
	s.Require().Equal(messageCount, len(response.Messages()), "it returns the messages")
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), messageCount)

	response, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool { return len(r.messages) > 0 },
		"no messages",
	)

	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), messageCount)

	for _, message := range response.Messages() {
		image := message.GetImage()
		s.Require().NotNil(image, "Message.ID=%s", message.ID)
		s.Require().NotEmpty(image.AlbumId, "Message.ID=%s", message.ID)
	}

	s.Require().Equal(uint(1), response.Chats()[0].UnviewedMessagesCount, "Just one unread message")
}

// This test makes sure that if you get a mention with an image ina  community, it sends it correctly and has a notif
func (s *MessengerSendImagesAlbumSuite) TestSingleImageMessageWithMentionInCommunitySend() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	defer TearDownMessenger(&s.Suite, theirMessenger)

	community, chat := createCommunity(&s.Suite, s.m)

	s.advertiseCommunityTo(community, theirMessenger)

	s.joinCommunity(community, theirMessenger)

	const messageCount = 1
	var album []*common.Message

	for i := 0; i < messageCount; i++ {
		outgoingMessage, err := buildImageWithoutAlbumIDMessage(*chat)
		s.NoError(err)
		outgoingMessage.Mentioned = true
		outgoingMessage.Text = "hey @" + common.PubkeyToHex(&theirMessenger.identity.PublicKey)
		album = append(album, outgoingMessage)
	}

	err = s.m.SaveChat(chat)
	s.NoError(err)
	response, err := s.m.SendChatMessages(context.Background(), album)
	s.NoError(err)
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
	s.Require().Len(response.ActivityCenterNotifications(), messageCount)

	for _, message := range response.Messages() {
		image := message.GetImage()
		s.Require().NotNil(image, "Message.ID=%s", message.ID)
		s.Require().Empty(image.AlbumId)
	}

	s.Require().Equal(uint(1), response.Chats()[0].UnviewedMessagesCount, "Just one unread message")
}

func (s *MessengerSendImagesAlbumSuite) TestAlbumImageEditText() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)

	theirChat := CreateOneToOneChat("Their 1TO1", &s.privateKey.PublicKey, s.m.transport)
	err = theirMessenger.SaveChat(theirChat)
	s.Require().NoError(err)

	ourChat := CreateOneToOneChat("Our 1TO1", &theirMessenger.identity.PublicKey, s.m.transport)
	err = s.m.SaveChat(ourChat)
	s.Require().NoError(err)

	const messageCount = 3
	var album []*common.Message

	for i := 0; i < messageCount; i++ {
		outgoingMessage, err := buildImageWithoutAlbumIDMessage(*ourChat)
		s.NoError(err)
		outgoingMessage.Text = "You can edit me now"
		album = append(album, outgoingMessage)
	}

	err = s.m.SaveChat(ourChat)
	s.NoError(err)
	response, err := s.m.SendChatMessages(context.Background(), album)
	s.NoError(err)
	s.Require().Equal(messageCount, len(response.Messages()), "it returns the messages")
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), messageCount)

	response, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool { return len(r.messages) > 0 },
		"no messages",
	)

	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), messageCount)

	for _, message := range response.Messages() {
		image := message.GetImage()
		s.Require().NotNil(image, "Message.ID=%s", message.ID)
		s.Require().NotEmpty(image.AlbumId, "Message.ID=%s", message.ID)
	}

	firstMessageID, err := types.DecodeHex(album[0].ID)
	s.Require().NoError(err)

	editedText := "edited"
	editedMessage := &requests.EditMessage{
		ID:   firstMessageID,
		Text: editedText,
	}

	sendResponse, err := s.m.EditMessage(context.Background(), editedMessage)

	s.Require().NoError(err)
	s.Require().Len(sendResponse.Messages(), messageCount)

	for _, message := range sendResponse.Messages() {
		s.Require().NotEmpty(message.EditedAt)
		s.Require().Equal(message.Text, editedText)
	}

	response, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool { return len(r.messages) > 0 },
		"no messages",
	)

	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), messageCount)

	for _, message := range response.Messages() {
		s.Require().NotEmpty(message.EditedAt)
		s.Require().Equal(message.Text, editedText)
	}
}

// This test makes sure that if you get a mention with an album of images in a community, it sends it correctly and has correct AC notif with album
func (s *MessengerSendImagesAlbumSuite) TestAlbumImagesMessageWithMentionInCommunitySend() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	defer TearDownMessenger(&s.Suite, theirMessenger)

	community, chat := createCommunity(&s.Suite, s.m)

	s.advertiseCommunityTo(community, theirMessenger)

	s.joinCommunity(community, theirMessenger)

	const messageCount = 3
	var album []*common.Message

	for i := 0; i < messageCount; i++ {
		outgoingMessage, err := buildImageWithoutAlbumIDMessage(*chat)
		s.NoError(err)
		outgoingMessage.Mentioned = true
		outgoingMessage.Text = "hey @" + common.PubkeyToHex(&theirMessenger.identity.PublicKey)
		album = append(album, outgoingMessage)
	}

	err = s.m.SaveChat(chat)
	s.NoError(err)
	response, err := s.m.SendChatMessages(context.Background(), album)
	s.NoError(err)
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
	s.Require().Len(response.ActivityCenterNotifications(), 1)

	for _, notif := range response.ActivityCenterNotifications() {
		s.Require().Equal(messageCount, len(notif.AlbumMessages), "AC notification should have AlbumMessages")
	}

	for _, message := range response.Messages() {
		image := message.GetImage()
		s.Require().NotNil(image, "Message.ID=%s", message.ID)
		s.Require().NotEmpty(image.AlbumId)
	}

	s.Require().Equal(uint(1), response.Chats()[0].UnviewedMessagesCount, "Just one unread message")
}
