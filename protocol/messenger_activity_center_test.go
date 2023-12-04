package protocol

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/server"
)

func TestMessengerActivityCenterMessageSuite(t *testing.T) {
	suite.Run(t, new(MessengerActivityCenterMessageSuite))
}

func (s *MessengerActivityCenterMessageSuite) createCommunity(owner *Messenger) (*communities.Community, *Chat) {
	return createCommunity(&s.Suite, owner)
}

func (s *MessengerActivityCenterMessageSuite) advertiseCommunityTo(community *communities.Community, owner *Messenger, user *Messenger) {
	advertiseCommunityTo(&s.Suite, community, owner, user)
}

func (s *MessengerActivityCenterMessageSuite) joinCommunity(community *communities.Community, owner *Messenger, user *Messenger) {
	request := &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	joinCommunity(&s.Suite, community, owner, user, request, "")
}

type MessengerActivityCenterMessageSuite struct {
	MessengerBaseTestSuite
}

func (s *MessengerActivityCenterMessageSuite) TestDeleteOneToOneChat() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	defer TearDownMessenger(&s.Suite, theirMessenger)

	theirChat := CreateOneToOneChat("Their 1TO1", &s.privateKey.PublicKey, s.m.transport)
	err = theirMessenger.SaveChat(theirChat)
	s.Require().NoError(err)

	r := &requests.SendContactRequest{
		ID:      s.m.myHexIdentity(),
		Message: "hello",
	}
	sendResponse, err := theirMessenger.SendContactRequest(context.Background(), r)
	s.NoError(err)
	s.Require().Len(sendResponse.Messages(), 2)

	response, err := WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.messages) > 0 },
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 2)
	s.Require().Len(response.ActivityCenterNotifications(), 1)

	request := &requests.DeactivateChat{ID: response.Chats()[0].ID}
	response, err = s.m.DeactivateChat(request)
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	deletedChat := response.Chats()[0]
	s.Require().NotEmpty(deletedChat.DeletedAtClockValue)

	// Make sure deleted at clock value is greater
	theirChat.LastClockValue = deletedChat.DeletedAtClockValue + 1
	err = theirMessenger.SaveChat(theirChat)
	s.Require().NoError(err)

	// Send another message
	inputMessage := buildTestMessage(*theirChat)
	sendResponse, err = theirMessenger.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)
	s.Require().Len(sendResponse.Messages(), 1)

	response, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Chats()) > 0 },
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
}

func (s *MessengerActivityCenterMessageSuite) TestEveryoneMentionTag() {
	alice := s.m
	bob := s.newMessenger()
	_, err := bob.Start()
	s.Require().NoError(err)
	defer TearDownMessenger(&s.Suite, bob)

	// Create a community
	community, chat := s.createCommunity(bob)
	s.Require().NotNil(community)
	s.Require().NotNil(chat)

	// Alice joins the community
	s.advertiseCommunityTo(community, bob, alice)
	s.joinCommunity(community, bob, alice)

	// alice sends a community message
	inputMessage := common.NewMessage()
	inputMessage.ChatId = chat.ID
	inputMessage.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	inputMessage.Text = "Good news, @" + common.EveryoneMentionTag + " !"

	response, err := alice.SendChatMessage(context.Background(), inputMessage)
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 1)
	s.Require().True(response.Messages()[0].Mentioned)

	response, err = WaitOnMessengerResponse(
		bob,
		func(r *MessengerResponse) bool { return len(r.Messages()) >= 1 },
		"no messages",
	)

	s.Require().NoError(err)

	s.Require().Len(response.Messages(), 1)

	s.Require().True(response.Messages()[0].Mentioned)
	s.Require().Len(response.ActivityCenterNotifications(), 1)
	s.Require().Equal(ActivityCenterNotificationTypeMention, response.ActivityCenterNotifications()[0].Type)
}

func (s *MessengerActivityCenterMessageSuite) TestReplyWithImage() {
	alice := s.m
	bob := s.newMessenger()
	_, err := bob.Start()
	s.Require().NoError(err)
	defer TearDownMessenger(&s.Suite, bob)

	// create an http server
	mediaServer, err := server.NewMediaServer(nil, nil, nil)
	s.Require().NoError(err)
	s.Require().NotNil(mediaServer)
	s.Require().NoError(mediaServer.Start())
	alice.httpServer = mediaServer

	// Create a community
	community, chat := s.createCommunity(bob)
	s.Require().NotNil(community)
	s.Require().NotNil(chat)

	// Alice joins the community
	s.advertiseCommunityTo(community, bob, alice)
	s.joinCommunity(community, bob, alice)

	// Alice sends a community message
	inputMessage := common.NewMessage()
	inputMessage.ChatId = chat.ID
	inputMessage.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	inputMessage.Text = "test message"

	response, err := alice.SendChatMessage(context.Background(), inputMessage)
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 1)
	response, err = WaitOnMessengerResponse(
		bob,
		func(r *MessengerResponse) bool { return len(r.Messages()) == 1 },
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 1)

	// bob sends a reply with an image
	inputMessage, err = buildImageWithAlbumIDMessage(*chat, "0x34")
	s.Require().NoError(err)

	inputMessage.Text = "test message reply"
	inputMessage.ResponseTo = response.Messages()[0].ID

	response, err = bob.SendChatMessage(context.Background(), inputMessage)
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 2)
	response, err = WaitOnMessengerResponse(
		alice,
		func(r *MessengerResponse) bool { return len(r.Messages()) == 2 },
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().Len(response.ActivityCenterNotifications(), 1)

	// verify the new message
	var newMessage *common.Message
	for _, m := range response.Messages() {
		if m.Text == "test message reply" {
			newMessage = m
		}
	}
	s.Require().NotNil(newMessage)
	s.Require().Equal(protobuf.ChatMessage_IMAGE, newMessage.ContentType)
	s.Require().NotEmpty(newMessage.ImageLocalURL)

	s.Require().NotNil(response.ActivityCenterNotifications()[0].Message)
	s.Require().NotEmpty(response.ActivityCenterNotifications()[0].Message.ImageLocalURL)
	s.Require().Equal(ActivityCenterNotificationTypeReply, response.ActivityCenterNotifications()[0].Type)

	notifResponse, err := alice.ActivityCenterNotifications(ActivityCenterNotificationsRequest{
		Limit:         8,
		ReadType:      ActivityCenterQueryParamsReadAll,
		ActivityTypes: []ActivityCenterType{ActivityCenterNotificationTypeReply},
	})
	s.Require().NoError(err)
	s.Require().Len(notifResponse.Notifications, 1)
	s.Require().NotNil(notifResponse.Notifications[0].Message)
	s.Require().NotEmpty(notifResponse.Notifications[0].Message.ImageLocalURL)
}

func (s *MessengerActivityCenterMessageSuite) TestMuteCommunityActivityCenterNotifications() {
	alice := s.m
	bob := s.newMessenger()
	_, err := bob.Start()
	s.Require().NoError(err)

	// Create a community
	community, chat := s.createCommunity(bob)
	s.Require().NotNil(community)
	s.Require().NotNil(chat)

	// Alice joins the community
	s.advertiseCommunityTo(community, bob, alice)
	s.joinCommunity(community, bob, alice)

	// Bob mutes the community
	time, err := bob.MuteAllCommunityChats(&requests.MuteCommunity{
		CommunityID: community.ID(),
		MutedType:   MuteTillUnmuted,
	})
	s.Require().NoError(err)
	s.Require().NotNil(time)

	bobCommunity, err := bob.GetCommunityByID(community.ID())
	s.Require().NoError(err)
	s.Require().True(bobCommunity.Muted())

	// alice sends a community message
	inputMessage := common.NewMessage()
	inputMessage.ChatId = chat.ID
	inputMessage.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	inputMessage.Text = "Good news, @" + common.EveryoneMentionTag + " !"

	response, err := alice.SendChatMessage(context.Background(), inputMessage)
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 1)
	s.Require().True(response.Messages()[0].Mentioned)

	response, err = WaitOnMessengerResponse(
		bob,
		func(r *MessengerResponse) bool { return len(r.Messages()) == 1 },
		"no messages",
	)

	// Bob still receives it, but no AC notif
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 1)
	s.Require().True(response.Messages()[0].Mentioned)
	s.Require().Len(response.ActivityCenterNotifications(), 0)
}

func (s *MessengerActivityCenterMessageSuite) TestReadCommunityOverviewNotifications() {
	alice := s.m
	bob := s.newMessenger()
	_, err := bob.Start()
	s.Require().NoError(err)
	defer bob.Shutdown() // nolint: errcheck

	// Create a community
	community, chat := s.createCommunity(bob)
	s.Require().NotNil(community)
	s.Require().NotNil(chat)

	// Alice joins the community
	s.advertiseCommunityTo(community, bob, alice)
	s.joinCommunity(community, bob, alice)

	// Mark community overview notification read
	err = alice.DismissActivityCenterNotificationsByCommunity(context.Background(), &requests.DismissCommunityNotifications{CommunityID: community.ID()})
	s.Require().NoError(err)

	response, err := alice.GetActivityCenterState()
	s.Require().NoError(err)
	s.Require().Equal(response.HasSeen, true)
}

func (s *MessengerActivityCenterMessageSuite) prepareCommunityChannelWithMentionAndReply() (*Messenger, *Messenger, *common.Message, *common.Message, *communities.Community) {
	alice := s.m
	bob := s.newMessenger()
	_, err := bob.Start()
	s.Require().NoError(err)
	defer TearDownMessenger(&s.Suite, bob)

	// Create a community
	community, chat := s.createCommunity(bob)
	s.Require().NotNil(community)
	s.Require().NotNil(chat)

	// Alice joins the community
	s.advertiseCommunityTo(community, bob, alice)
	s.joinCommunity(community, bob, alice)

	// Bob sends a mention message
	mentionMessage := common.NewMessage()
	mentionMessage.ChatId = chat.ID
	mentionMessage.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	mentionMessage.Text = "Good news, @" + common.EveryoneMentionTag + " !"

	response, err := bob.SendChatMessage(context.Background(), mentionMessage)
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 1)
	s.Require().True(response.Messages()[0].Mentioned)

	// check alice got the mention message
	response, err = WaitOnMessengerResponse(
		alice,
		func(r *MessengerResponse) bool {
			return len(r.Messages()) == 1 && len(r.ActivityCenterNotifications()) == 1 &&
				r.Messages()[0].ID == r.ActivityCenterNotifications()[0].Message.ID &&
				r.ActivityCenterNotifications()[0].Type == ActivityCenterNotificationTypeMention
		},
		"no messages",
	)
	s.Require().NoError(err)

	s.Require().False(response.ActivityCenterNotifications()[0].Read)
	s.Require().Equal(response.ActivityCenterNotifications()[0].ID.String(), response.ActivityCenterNotifications()[0].Message.ID)
	mentionMessage = response.Messages()[0]

	// Alice sends a community message
	inputMessage := common.NewMessage()
	inputMessage.ChatId = chat.ID
	inputMessage.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	inputMessage.Text = "test message"

	response, err = alice.SendChatMessage(context.Background(), inputMessage)
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 1)

	// Check the community message is received by Bob
	_, err = WaitOnMessengerResponse(
		bob,
		func(r *MessengerResponse) bool { return len(r.Messages()) == 1 },
		"no messages",
	)
	s.Require().NoError(err)

	// Bob sends a reply message
	replyMessage := common.NewMessage()
	replyMessage.ChatId = chat.ID
	replyMessage.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	replyMessage.Text = "test message reply"
	replyMessage.ResponseTo = response.Messages()[0].ID

	response, err = bob.SendChatMessage(context.Background(), replyMessage)
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 2)

	// Check Alice got the reply message
	response, err = WaitOnMessengerResponse(
		alice,
		func(r *MessengerResponse) bool {
			return len(r.Messages()) == 2 && len(r.ActivityCenterNotifications()) == 1 &&
				(r.Messages()[0].ID == r.ActivityCenterNotifications()[0].Message.ID ||
					r.Messages()[1].ID == r.ActivityCenterNotifications()[0].Message.ID) &&
				r.ActivityCenterNotifications()[0].Type == ActivityCenterNotificationTypeReply
		},
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().False(response.ActivityCenterNotifications()[0].Read)

	// There is an extra message with reply
	if response.Messages()[0].ID == response.ActivityCenterNotifications()[0].Message.ID {
		replyMessage = response.Messages()[0]
	} else if response.Messages()[1].ID == response.ActivityCenterNotifications()[0].Message.ID {
		replyMessage = response.Messages()[1]
	} else {
		s.Error(errors.New("can't find corresponding message in the response"))
	}

	s.confirmMentionAndReplyNotificationsRead(alice, mentionMessage, replyMessage, false)

	return alice, bob, mentionMessage, replyMessage, community
}

func (s *MessengerActivityCenterMessageSuite) confirmMentionAndReplyNotificationsRead(user *Messenger, mentionMessage *common.Message, replyMessage *common.Message, read bool) {
	// Confirm reply notification
	notifResponse, err := user.ActivityCenterNotifications(ActivityCenterNotificationsRequest{
		Limit:         8,
		ReadType:      ActivityCenterQueryParamsReadAll,
		ActivityTypes: []ActivityCenterType{ActivityCenterNotificationTypeReply},
	})
	s.Require().NoError(err)
	s.Require().Len(notifResponse.Notifications, 1)
	s.Require().Equal(read, notifResponse.Notifications[0].Read)

	// Confirm mention notification
	notifResponse, err = user.ActivityCenterNotifications(ActivityCenterNotificationsRequest{
		Limit:         8,
		ReadType:      ActivityCenterQueryParamsReadAll,
		ActivityTypes: []ActivityCenterType{ActivityCenterNotificationTypeMention},
	})
	s.Require().NoError(err)
	s.Require().Len(notifResponse.Notifications, 1)
	s.Require().Equal(read, notifResponse.Notifications[0].Read)
}

func (s *MessengerActivityCenterMessageSuite) TestMarkMessagesSeenMarksNotificationsRead() {
	alice, _, mentionMessage, replyMessage, _ := s.prepareCommunityChannelWithMentionAndReply()

	_, _, notifications, err := alice.MarkMessagesSeen(replyMessage.ChatId, []string{mentionMessage.ID, replyMessage.ID})

	s.Require().NoError(err)
	s.Require().Len(notifications, 2)
	s.Require().True(notifications[0].Read)
	s.Require().True(notifications[1].Read)

	s.confirmMentionAndReplyNotificationsRead(alice, mentionMessage, replyMessage, true)
}

func (s *MessengerActivityCenterMessageSuite) TestMarkAllReadMarksNotificationsRead() {
	alice, _, mentionMessage, replyMessage, _ := s.prepareCommunityChannelWithMentionAndReply()

	response, err := alice.MarkAllRead(context.Background(), mentionMessage.ChatId)

	s.Require().NoError(err)
	s.Require().Len(response.ActivityCenterNotifications(), 2)
	s.Require().True(response.ActivityCenterNotifications()[0].Read)
	s.Require().True(response.ActivityCenterNotifications()[1].Read)

	s.confirmMentionAndReplyNotificationsRead(alice, mentionMessage, replyMessage, true)
}

func (s *MessengerActivityCenterMessageSuite) TestMarkAllReadInCommunityMarksNotificationsRead() {
	alice, _, mentionMessage, replyMessage, community := s.prepareCommunityChannelWithMentionAndReply()

	response, err := alice.MarkAllReadInCommunity(context.Background(), community.IDString())

	s.Require().NoError(err)
	s.Require().Len(response.ActivityCenterNotifications(), 2)
	s.Require().True(response.ActivityCenterNotifications()[0].Read)
	s.Require().True(response.ActivityCenterNotifications()[1].Read)

	s.confirmMentionAndReplyNotificationsRead(alice, mentionMessage, replyMessage, true)
}

func (s *MessengerActivityCenterMessageSuite) TestMarkAllActivityCenterNotificationsReadMarksMessagesAsSeen() {
	alice, _, mentionMessage, replyMessage, _ := s.prepareCommunityChannelWithMentionAndReply()

	response, err := alice.MarkAllActivityCenterNotificationsRead(context.Background())

	s.Require().NoError(err)
	s.Require().Len(response.ActivityCenterNotifications(), 3)
	s.Require().True(response.ActivityCenterNotifications()[0].Read)
	s.Require().True(response.ActivityCenterNotifications()[1].Read)
	s.Require().True(response.ActivityCenterNotifications()[2].Read)

	s.confirmMentionAndReplyNotificationsRead(alice, mentionMessage, replyMessage, true)
}

func (s *MessengerActivityCenterMessageSuite) TestAliceDoesNotReceiveCommunityNotificationsBeforeJoined() {
	alice := s.m
	bob := s.newMessenger()
	_, err := bob.Start()
	s.Require().NoError(err)
	defer bob.Shutdown() // nolint: errcheck

	// Create a community
	community, chat := s.createCommunity(bob)
	s.Require().NotNil(community)
	s.Require().NotNil(chat)

	// Bob sends a mention message
	mentionMessage := common.NewMessage()
	mentionMessage.ChatId = chat.ID
	mentionMessage.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	mentionMessage.Text = "Good news, @" + common.EveryoneMentionTag + " !"

	response, err := bob.SendChatMessage(context.Background(), mentionMessage)
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 1)
	s.Require().True(response.Messages()[0].Mentioned)

	// Alice joins the community
	s.advertiseCommunityTo(community, bob, alice)
	s.joinCommunity(community, bob, alice)

	// Bob sends an another mention message
	mentionMessage = common.NewMessage()
	mentionMessage.ChatId = chat.ID
	mentionMessage.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	mentionMessage.Text = "Welcome new joiners, @" + common.EveryoneMentionTag + " !"

	response, err = bob.SendChatMessage(context.Background(), mentionMessage)
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 1)
	s.Require().True(response.Messages()[0].Mentioned)

	// check alice got only second mention message
	response, err = WaitOnMessengerResponse(
		alice,
		func(r *MessengerResponse) bool {
			return len(r.Messages()) == 1 && len(r.ActivityCenterNotifications()) == 1 &&
				r.Messages()[0].ID == r.ActivityCenterNotifications()[0].Message.ID &&
				r.ActivityCenterNotifications()[0].Type == ActivityCenterNotificationTypeMention
		},
		"no messages",
	)
	s.Require().NoError(err)

	s.Require().False(response.ActivityCenterNotifications()[0].Read)
	s.Require().Equal(response.ActivityCenterNotifications()[0].ID.String(), response.ActivityCenterNotifications()[0].Message.ID)
}
