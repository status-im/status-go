package protocol

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
)

func TestMessengerDeleteMessageForEveryoneSuite(t *testing.T) {
	suite.Run(t, new(MessengerDeleteMessageForEveryoneSuite))
}

type MessengerDeleteMessageForEveryoneSuite struct {
	CommunitiesMessengerTestSuiteBase
	admin     *Messenger
	moderator *Messenger
	bob       *Messenger
}

func (s *MessengerDeleteMessageForEveryoneSuite) SetupTest() {
	s.CommunitiesMessengerTestSuiteBase.SetupTest()
	s.admin = s.newMessenger("", []string{})
	s.bob = s.newMessenger(bobPassword, []string{bobPassword})
	s.moderator = s.newMessenger(aliceAccountAddress, []string{aliceAddress1})

	_, err := s.admin.Start()
	s.Require().NoError(err)
	_, err = s.bob.Start()
	s.Require().NoError(err)
	_, err = s.moderator.Start()
	s.Require().NoError(err)
}

func (s *MessengerDeleteMessageForEveryoneSuite) TearDownTest() {
	TearDownMessenger(&s.Suite, s.admin)
	TearDownMessenger(&s.Suite, s.bob)
	TearDownMessenger(&s.Suite, s.moderator)
	s.CommunitiesMessengerTestSuiteBase.TearDownTest()
}

func (s *MessengerDeleteMessageForEveryoneSuite) testSendAndDeleteMessage(messageToSend *common.Message, shouldError bool) {
	ctx := context.Background()
	sendResponse, err := s.bob.SendChatMessage(ctx, messageToSend)
	s.NoError(err)
	s.Require().Len(sendResponse.Messages(), 1)

	response, err := WaitOnMessengerResponse(s.moderator, func(response *MessengerResponse) bool {
		return len(response.Messages()) > 0
	}, "messages not received")
	s.Require().NoError(err)
	message := response.Messages()[0]
	s.Require().Equal(messageToSend.Text, message.Text)

	deleteMessageResponse, err := s.moderator.DeleteMessageAndSend(ctx, message.ID)
	if shouldError {
		s.Require().Error(err)
		return
	}
	s.Require().NoError(err)

	response, err = WaitOnMessengerResponse(s.bob, func(response *MessengerResponse) bool {
		return len(response.RemovedMessages()) > 0
	}, "removed messages not received")
	s.Require().Equal(deleteMessageResponse.RemovedMessages()[0].DeletedBy, contactIDFromPublicKey(s.moderator.IdentityPublicKey()))

	s.Require().NoError(err)
	s.Require().Len(response.ActivityCenterNotifications(), 1)
	s.Require().True(response.ActivityCenterNotifications()[0].Deleted)

	message, err = s.bob.MessageByID(message.ID)
	s.Require().NoError(err)
	s.Require().True(message.Deleted)
}

func (s *MessengerDeleteMessageForEveryoneSuite) TestDeleteMessageForEveryone() {
	community := s.createCommunity()
	communityChat := s.createCommunityChat(community)

	advertiseCommunityTo(&s.Suite, community, s.admin, s.moderator)
	joinCommunity(&s.Suite, community.ID(), s.admin, s.moderator, aliceAccountAddress, []string{aliceAddress1})

	advertiseCommunityTo(&s.Suite, community, s.admin, s.bob)
	joinCommunity(&s.Suite, community.ID(), s.admin, s.bob, bobPassword, []string{bobAddress})

	response, err := s.admin.AddRoleToMember(&requests.AddRoleToMember{
		CommunityID: community.ID(),
		User:        common.PubkeyToHexBytes(s.moderator.IdentityPublicKey()),
		Role:        protobuf.CommunityMember_ROLE_ADMIN,
	})
	s.Require().NoError(err)
	s.Require().Len(response.Communities(), 1)

	_, err = WaitOnMessengerResponse(s.moderator, func(response *MessengerResponse) bool {
		return len(response.Communities()) > 0
	}, "community description changed message not received")
	s.Require().NoError(err)
	_, err = WaitOnMessengerResponse(s.bob, func(response *MessengerResponse) bool {
		return len(response.Communities()) > 0
	}, "community description changed message not received")
	s.Require().NoError(err)

	// // Normal message
	inputMessage := common.NewMessage()
	inputMessage.ChatId = communityChat.ID
	inputMessage.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	inputMessage.Text = "some text"

	s.testSendAndDeleteMessage(inputMessage, false)

	// // Bridge message
	bridgeMessage := buildTestMessage(*communityChat)
	bridgeMessage.ContentType = protobuf.ChatMessage_BRIDGE_MESSAGE
	bridgeMessage.Payload = &protobuf.ChatMessage_BridgeMessage{
		BridgeMessage: &protobuf.BridgeMessage{
			BridgeName:      "discord",
			UserName:        "user1",
			UserAvatar:      "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAADIAAAAyCAIAAACRXR/mAAAAjklEQVR4nOzXwQmFMBAAUZXUYh32ZB32ZB02sxYQQSZGsod55/91WFgSS0RM+SyjA56ZRZhFmEWYRRT6h+M6G16zrxv6fdJpmUWYRbxsYr13dKfanpN0WmYRZhGzXz6AWYRZRIfbaX26fT9Jk07LLMIsosPt9I/dTDotswizCG+nhFmEWYRZhFnEHQAA///z1CFkYamgfQAAAABJRU5ErkJggg==",
			UserID:          "123",
			Content:         "text1",
			MessageID:       "456",
			ParentMessageID: "789",
		},
	}
	s.testSendAndDeleteMessage(bridgeMessage, false)

	// Gap message cannot be deleted
	gapMessage := buildTestMessage(*communityChat)
	gapMessage.ContentType = protobuf.ChatMessage_SYSTEM_MESSAGE_GAP
	s.testSendAndDeleteMessage(gapMessage, true)
}

func (s *MessengerDeleteMessageForEveryoneSuite) createCommunity() *communities.Community {
	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}
	response, err := s.admin.CreateCommunity(description, false)

	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.Chats(), 0)

	return response.Communities()[0]
}

func (s *MessengerDeleteMessageForEveryoneSuite) createCommunityChat(community *communities.Community) *Chat {
	orgChat := &protobuf.CommunityChat{
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_AUTO_ACCEPT,
		},
		Identity: &protobuf.ChatIdentity{
			DisplayName: "status-core",
			Emoji:       "",
			Description: "status-core community chatToModerator",
		},
	}

	response, err := s.admin.CreateCommunityChat(community.ID(), orgChat)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.Chats(), 1)
	return response.Chats()[0]
}
