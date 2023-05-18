package protocol

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/crypto"
	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"
)

func TestMessengerDeleteMessageForEveryoneSuite(t *testing.T) {
	suite.Run(t, new(MessengerDeleteMessageForEveryoneSuite))
}

type MessengerDeleteMessageForEveryoneSuite struct {
	suite.Suite
	admin     *Messenger
	moderator *Messenger
	bob       *Messenger
	shh       types.Waku
	logger    *zap.Logger
}

func (s *MessengerDeleteMessageForEveryoneSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())

	s.admin = s.newMessenger()
	s.bob = s.newMessenger()
	s.moderator = s.newMessenger()
	_, err := s.admin.Start()
	s.Require().NoError(err)
	_, err = s.bob.Start()
	s.Require().NoError(err)
	_, err = s.moderator.Start()
	s.Require().NoError(err)
}

func (s *MessengerDeleteMessageForEveryoneSuite) TearDownTest() {
	s.Require().NoError(s.admin.Shutdown())
	s.Require().NoError(s.bob.Shutdown())
	_ = s.logger.Sync()
}

func (s *MessengerDeleteMessageForEveryoneSuite) newMessenger() *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	messenger, err := newMessengerWithKey(s.shh, privateKey, s.logger, nil)
	s.Require().NoError(err)
	return messenger
}

func (s *MessengerDeleteMessageForEveryoneSuite) TestDeleteMessageForEveryone() {
	community := s.createCommunity()
	communityChat := s.createCommunityChat(community)
	s.inviteAndJoin(community, s.moderator)
	s.inviteAndJoin(community, s.bob)

	response, err := s.admin.AddRoleToMember(&requests.AddRoleToMember{
		CommunityID: community.ID(),
		User:        common.PubkeyToHexBytes(s.moderator.IdentityPublicKey()),
		Role:        protobuf.CommunityMember_ROLE_MODERATE_CONTENT,
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

	ctx := context.Background()
	inputMessage := &common.Message{ChatMessage: &protobuf.ChatMessage{}}
	inputMessage.ChatId = communityChat.ID
	inputMessage.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	inputMessage.Text = "some text"
	_, err = s.bob.SendChatMessage(ctx, inputMessage)
	s.Require().NoError(err)

	response, err = WaitOnMessengerResponse(s.moderator, func(response *MessengerResponse) bool {
		return len(response.Messages()) > 0
	}, "messages not received")
	s.Require().NoError(err)
	message := response.Messages()[0]
	s.Require().Equal(inputMessage.Text, message.Text)

	deleteMessageResponse, err := s.moderator.DeleteMessageAndSend(ctx, message.ID)
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

func (s *MessengerDeleteMessageForEveryoneSuite) createCommunity() *communities.Community {
	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
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
			Access: protobuf.CommunityPermissions_NO_MEMBERSHIP,
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

func (s *MessengerDeleteMessageForEveryoneSuite) inviteAndJoin(community *communities.Community, target *Messenger) {
	response, err := s.admin.InviteUsersToCommunity(&requests.InviteUsersToCommunity{
		CommunityID: community.ID(),
		Users:       []types.HexBytes{common.PubkeyToHexBytes(&target.identity.PublicKey)},
	})
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	community = response.Communities()[0]
	s.Require().True(community.HasMember(&target.identity.PublicKey))

	_, err = WaitOnMessengerResponse(target, func(response *MessengerResponse) bool {
		return len(response.Communities()) > 0
	}, "community not received")
	s.Require().NoError(err)

	response, err = target.JoinCommunity(context.Background(), community.ID())
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().True(response.Communities()[0].Joined())
	s.Require().Len(response.Chats(), 1)

	s.Require().NoError(target.SaveChat(response.Chats()[0]))

	_, err = WaitOnMessengerResponse(target, func(response *MessengerResponse) bool {
		return len(response.Messages()) > 0
	}, "message 'You have been invited to community' not received")
	s.Require().NoError(err)
}
