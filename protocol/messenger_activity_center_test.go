package protocol

import (
	"context"
	"crypto/ecdsa"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/server"
	"github.com/status-im/status-go/waku"
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
	joinCommunity(&s.Suite, community, owner, user, request)
}

type MessengerActivityCenterMessageSuite struct {
	suite.Suite
	m          *Messenger        // main instance of Messenger
	privateKey *ecdsa.PrivateKey // private key for the main instance of Messenger
	// If one wants to send messages between different instances of Messenger,
	// a single waku service should be shared.
	shh    types.Waku
	logger *zap.Logger
}

func (s *MessengerActivityCenterMessageSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())

	s.m = s.newMessenger()
	s.privateKey = s.m.identity
	_, err := s.m.Start()
	s.Require().NoError(err)
}

func (s *MessengerActivityCenterMessageSuite) TearDownTest() {
	s.Require().NoError(s.m.Shutdown())
}

func (s *MessengerActivityCenterMessageSuite) newMessenger() *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	messenger, err := newMessengerWithKey(s.shh, privateKey, s.logger, nil)
	s.Require().NoError(err)
	return messenger
}

func (s *MessengerActivityCenterMessageSuite) TestDeleteOneToOneChat() {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
	defer theirMessenger.Shutdown() // nolint: errcheck

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
	defer bob.Shutdown() // nolint: errcheck

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
	defer bob.Shutdown() // nolint: errcheck

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
