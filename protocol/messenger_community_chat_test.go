package protocol

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
)

func TestMessengerCommunityChatsUnitSuite(t *testing.T) {
	suite.Run(t, new(MessengerCommunityChatsUnitSuite))
}

type MessengerCommunityChatsUnitSuite struct {
	MessengerBaseTestSuite
}

func (s *MessengerCommunityChatsUnitSuite) TestCreateCommunityChat_GlobalCommunityContentTopicFilterSet() {
	// Create a community first
	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
		Name:        "test community",
		Color:       "#ffffff",
		Description: "test community description",
	}

	communityResponse, err := s.m.CreateCommunity(description, false)
	s.Require().NoError(err)
	s.Require().NotNil(communityResponse)
	s.Require().Len(communityResponse.Communities(), 1)

	community := communityResponse.Communities()[0]

	// Verify that the global community content topic filter is set
	// The universal/global community content topic filter should be created for the community
	universalChatFilter := s.m.messaging.ChatFilterByChatID(community.UniversalChatID())
	s.Require().NotNil(universalChatFilter, "Global community content topic filter should be set for community")

	// Test creating a community chat
	communityChat := &protobuf.CommunityChat{
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_AUTO_ACCEPT,
		},
		Identity: &protobuf.ChatIdentity{
			DisplayName: "test-chat",
			Description: "test chat description",
		},
	}

	response, err := s.m.CreateCommunityChat(community.ID(), communityChat)

	// Verify the operation succeeded
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.Chats(), 1)

	// Verify the global community content topic filter is still set after creating the chat
	universalChatFilterAfter := s.m.messaging.ChatFilterByChatID(community.UniversalChatID())
	s.Require().NotNil(universalChatFilterAfter, "Global community content topic filter should remain set after creating community chat")
	s.Require().Equal(universalChatFilter.FilterID, universalChatFilterAfter.FilterID, "Global community content topic filter should remain the same")
}

func (s *MessengerCommunityChatsUnitSuite) TestEditCommunityChat_GlobalCommunityContentTopicFilterSet() {
	// Create a community and a chat first
	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
		Name:        "test community",
		Color:       "#ffffff",
		Description: "test community description",
	}

	communityResponse, err := s.m.CreateCommunity(description, false)
	s.Require().NoError(err)
	community := communityResponse.Communities()[0]

	// Verify that the global community content topic filter is set
	universalChatFilter := s.m.messaging.ChatFilterByChatID(community.UniversalChatID())
	s.Require().NotNil(universalChatFilter, "Global community content topic filter should be set for community")

	// Create initial chat
	initialChat := &protobuf.CommunityChat{
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_AUTO_ACCEPT,
		},
		Identity: &protobuf.ChatIdentity{
			DisplayName: "initial-chat",
			Description: "initial chat description",
		},
	}

	createResponse, err := s.m.CreateCommunityChat(community.ID(), initialChat)
	s.Require().NoError(err)
	s.Require().Len(createResponse.Chats(), 1)

	createdChat := createResponse.Chats()[0]
	chatID := createdChat.CommunityChatID()

	// Edit the chat
	editedChat := &protobuf.CommunityChat{
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_MANUAL_ACCEPT,
		},
		Identity: &protobuf.ChatIdentity{
			DisplayName: "edited-chat",
			Description: "edited chat description",
		},
	}

	response, err := s.m.EditCommunityChat(community.ID(), chatID, editedChat)

	// Verify the operation succeeded
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.Chats(), 1)

	// Verify the global community content topic filter is still set after editing the chat
	universalChatFilterAfter := s.m.messaging.ChatFilterByChatID(community.UniversalChatID())
	s.Require().NotNil(universalChatFilterAfter, "Global community content topic filter should remain set after editing community chat")
	s.Require().Equal(universalChatFilter.FilterID, universalChatFilterAfter.FilterID, "Global community content topic filter should remain the same")
}
