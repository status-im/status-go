package protocol

import (
	"context"
	"errors"

	_ "github.com/mutecomm/go-sqlcipher/v4" // require go-sqlcipher that overrides default implementation

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/tt"
)

func (s *MessengerSuite) checkMessageSeen(messageID string, expectedSeen bool) {
	message, err := s.m.MessageByID(messageID)

	s.Require().NoError(err)
	s.Require().Equal(expectedSeen, message.Seen)
}

func (s *MessengerSuite) retrieveAllWithRetry(errorMessage string) (*MessengerResponse, error) {
	var response *MessengerResponse
	var err error

	retryFunc := func() error {
		response, err = s.m.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.messages) == 0 {
			return errors.New(errorMessage)
		}
		return nil
	}

	err = tt.RetryWithBackOff(retryFunc)
	return response, err
}

func (s *MessengerSuite) TestMarkMessageAsUnreadWhenMessageListContainsSingleMessage() {
	chat := CreatePublicChat("test-chat-1", s.m.transport)

	chat.UnviewedMessagesCount = 2
	chat.UnviewedMentionsCount = 2
	chat.Highlight = true

	err := s.m.SaveChat(chat)
	s.Require().NoError(err)

	inputMessage1 := buildTestMessage(*chat)
	inputMessage1.ID = "1"
	inputMessage1.Seen = true
	inputMessage1.Mentioned = true

	err = s.m.SaveMessages([]*common.Message{inputMessage1})
	s.Require().NoError(err)

	_, err = s.m.MarkAllRead(context.Background(), chat.ID)
	s.Require().NoError(err)

	chats := s.m.allChats

	actualChat, ok := chats.Load(chat.ID)

	s.Require().True(ok)
	s.Require().Equal(uint(0), actualChat.UnviewedMessagesCount)
	s.Require().Equal(uint(0), actualChat.UnviewedMentionsCount)
	s.Require().False(actualChat.Highlight)
	s.checkMessageSeen(inputMessage1.ID, true)

	count, countWithMentions, notifications, err := s.m.MarkMessageAsUnread(chat.ID, inputMessage1.ID)
	s.Require().NoError(err)
	s.Require().Equal(uint64(1), count)
	s.Require().Equal(uint64(1), countWithMentions)
	s.Require().Len(notifications, 0)

	chats = s.m.allChats

	actualChat, ok = chats.Load(chat.ID)
	s.Require().True(ok)
	s.Require().Equal(uint(1), actualChat.UnviewedMessagesCount)
	s.Require().Equal(uint(1), actualChat.UnviewedMentionsCount)
	s.Require().False(actualChat.Highlight)
	s.checkMessageSeen(inputMessage1.ID, false)
}

func (s *MessengerSuite) TestMarkMessageAsUnreadWhenMessageListContainsSeveralMessages() {
	chat := CreatePublicChat("test-chat-2", s.m.transport)

	chat.UnviewedMessagesCount = 0
	chat.UnviewedMentionsCount = 0
	chat.Highlight = true

	err := s.m.SaveChat(chat)
	s.Require().NoError(err)

	inputMessage1 := buildTestMessage(*chat)
	inputMessage1.ID = "1"
	inputMessage1.Seen = true
	inputMessage1.Mentioned = true
	inputMessage1.Timestamp = 1

	inputMessage2 := buildTestMessage(*chat)
	inputMessage2.ID = "2"
	inputMessage2.Seen = true
	inputMessage2.Mentioned = false
	inputMessage2.Timestamp = 2

	inputMessage3 := buildTestMessage(*chat)
	inputMessage3.ID = "3"
	inputMessage3.Seen = true
	inputMessage3.Mentioned = false
	inputMessage3.Timestamp = 3

	err = s.m.SaveMessages([]*common.Message{
		inputMessage1,
		inputMessage2,
		inputMessage3,
	})
	s.Require().NoError(err)

	_, err = s.m.MarkAllRead(context.Background(), chat.ID)
	s.Require().NoError(err)

	chats := s.m.allChats
	actualChat, ok := chats.Load(chat.ID)

	s.Require().True(ok)
	s.Require().Equal(uint(0), actualChat.UnviewedMessagesCount)
	s.Require().Equal(uint(0), actualChat.UnviewedMentionsCount)
	s.Require().False(actualChat.Highlight)
	s.checkMessageSeen(inputMessage1.ID, true)
	s.checkMessageSeen(inputMessage2.ID, true)
	s.checkMessageSeen(inputMessage3.ID, true)

	count, countWithMentions, notifications, err := s.m.MarkMessageAsUnread(chat.ID, inputMessage2.ID)
	s.Require().NoError(err)

	// count is 2 because the messages are layout the following way :
	//
	// inputMessage1   read    <-- mentioned
	// ----------------------------- <-- marker
	// inputMessage2   unread
	// inputMessage3   unread
	//
	// And the inputMessage3 has greater timestamp than inputMessage2
	// Similarly, the mentioned message is inputMessage1, therefore the
	// countWithMentions is 0
	s.Require().Equal(uint64(2), count)
	s.Require().Equal(uint64(0), countWithMentions)
	s.Require().Len(notifications, 0)

	chats = s.m.allChats
	actualChat, ok = chats.Load(chat.ID)

	s.Require().True(ok)
	s.Require().Equal(uint(2), actualChat.UnviewedMessagesCount)
	s.Require().Equal(uint(0), actualChat.UnviewedMentionsCount)
	s.Require().False(actualChat.Highlight)
	s.checkMessageSeen(inputMessage1.ID, true)
	s.checkMessageSeen(inputMessage2.ID, false)
	s.checkMessageSeen(inputMessage3.ID, false)
}

func (s *MessengerSuite) TestMarkMessageAsUnreadWhenMessageIsAlreadyInUnreadState() {
	chat := CreatePublicChat("test-chat-3", s.m.transport)

	chat.UnviewedMessagesCount = 1
	chat.UnviewedMentionsCount = 0
	chat.Highlight = true

	err := s.m.SaveChat(chat)
	s.Require().NoError(err)

	inputMessage1 := buildTestMessage(*chat)
	inputMessage1.ID = "1"
	inputMessage1.Seen = false
	inputMessage1.Mentioned = false

	err = s.m.SaveMessages([]*common.Message{inputMessage1})
	s.Require().NoError(err)

	chats := s.m.allChats
	actualChat, ok := chats.Load(chat.ID)

	s.Require().True(ok)
	s.Require().Equal(uint(1), actualChat.UnviewedMessagesCount)
	s.Require().Equal(uint(0), actualChat.UnviewedMentionsCount)
	s.Require().True(actualChat.Highlight)
	s.checkMessageSeen(inputMessage1.ID, false)

	count, countWithMentions, notifications, err := s.m.MarkMessageAsUnread(chat.ID, inputMessage1.ID)
	s.Require().NoError(err)
	s.Require().Equal(uint64(1), count)
	s.Require().Equal(uint64(0), countWithMentions)
	s.Require().Len(notifications, 0)

	chats = s.m.allChats
	actualChat, ok = chats.Load(chat.ID)

	s.Require().True(ok)
	s.Require().Equal(uint(1), actualChat.UnviewedMessagesCount)
	s.Require().Equal(uint(0), actualChat.UnviewedMentionsCount)
	s.Require().False(actualChat.Highlight)
	s.checkMessageSeen(inputMessage1.ID, false)
}

func (s *MessengerSuite) TestMarkMessageAsUnreadInOneChatDoesNotImpactOtherChats() {
	chat1 := CreatePublicChat("test-chat-1", s.m.transport)
	err := s.m.SaveChat(chat1)
	s.Require().NoError(err)

	chat2 := CreatePublicChat("test-chat-2", s.m.transport)
	err = s.m.SaveChat(chat2)
	s.Require().NoError(err)

	chat1.UnviewedMessagesCount = 2
	chat1.UnviewedMentionsCount = 0
	chat1.Highlight = true

	inputMessage1 := buildTestMessage(*chat1)
	inputMessage1.ID = "1"
	inputMessage1.Seen = true
	inputMessage1.Mentioned = false

	inputMessage2 := buildTestMessage(*chat1)
	inputMessage2.ID = "2"
	inputMessage2.Seen = true
	inputMessage2.Mentioned = false

	err = s.m.SaveMessages([]*common.Message{inputMessage1, inputMessage2})
	s.Require().NoError(err)

	chat2.UnviewedMessagesCount = 1
	chat2.UnviewedMentionsCount = 0
	chat2.Highlight = true

	inputMessage3 := buildTestMessage(*chat2)
	inputMessage3.ID = "3"
	inputMessage3.Seen = false
	inputMessage3.Mentioned = false

	err = s.m.SaveMessages([]*common.Message{inputMessage3})
	s.Require().NoError(err)

	_, err = s.m.MarkAllRead(context.Background(), chat1.ID)
	s.Require().NoError(err)
	s.checkMessageSeen(inputMessage1.ID, true)
	s.checkMessageSeen(inputMessage2.ID, true)

	count, countWithMentions, notifications, err := s.m.MarkMessageAsUnread(chat1.ID, inputMessage1.ID)

	s.Require().NoError(err)
	s.Require().Equal(uint(1), chat2.UnviewedMessagesCount)
	s.Require().Equal(uint64(2), count)
	s.Require().Equal(uint64(0), countWithMentions)
	s.Require().Len(notifications, 0)

	chats := s.m.allChats
	actualChat, ok := chats.Load(chat1.ID)

	s.Require().True(ok)
	s.Require().Equal(uint(2), actualChat.UnviewedMessagesCount)
	s.Require().Equal(uint(0), actualChat.UnviewedMentionsCount)
	s.Require().False(actualChat.Highlight)
	s.checkMessageSeen(inputMessage1.ID, false)
	s.checkMessageSeen(inputMessage2.ID, false)

	actualChat, ok = chats.Load(chat2.ID)
	s.Require().True(ok)
	s.Require().Equal(uint(1), actualChat.UnviewedMessagesCount)
	s.Require().Equal(uint(0), actualChat.UnviewedMentionsCount)
	s.Require().True(actualChat.Highlight)

	s.checkMessageSeen(inputMessage3.ID, false)
}

func (s *MessengerSuite) TestMarkMessageWithNotificationAsUnreadInCommunityChatSetsNotificationAsUnread() {
	other := s.newMessenger()

	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
		Name:        "status",
		Color:       "#ffffff",
		Description: "This is just a test description for the community",
	}

	response, err := other.CreateCommunity(description, true)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	community := response.Communities()[0]
	communityChat := response.Chats()[0]

	_, err = community.AddMember(&s.m.identity.PublicKey, []protobuf.CommunityMember_Roles{})
	s.Require().NoError(err)

	err = other.communitiesManager.SaveCommunity(community)
	s.Require().NoError(err)

	advertiseCommunityToUserOldWay(&s.Suite, community, other, s.m)

	inputMessage1 := buildTestMessage(*communityChat)
	inputMessage1.ChatId = communityChat.ID
	inputMessage1.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	inputMessage1.Text = "Hello @" + common.EveryoneMentionTag + " !"

	sendResponse, err := other.SendChatMessage(context.Background(), inputMessage1)
	s.NoError(err)
	s.Require().Len(sendResponse.Messages(), 1)

	response, err = s.retrieveAllWithRetry("message from other chatter not received")

	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)
	s.Require().Len(response.ActivityCenterNotifications(), 1)
	s.Require().False(response.ActivityCenterNotifications()[0].Read)

	response, err = s.m.MarkAllRead(context.Background(), communityChat.ID)

	s.Require().NoError(err)
	s.Require().Len(response.ActivityCenterNotifications(), 1)
	s.Require().True(response.ActivityCenterNotifications()[0].Read)

	count, countWithMentions, notifications, err := s.m.MarkMessageAsUnread(communityChat.ID, inputMessage1.ID)

	s.Require().NoError(err)
	s.Require().Equal(count, uint64(1))
	s.Require().Equal(countWithMentions, uint64(1))
	s.Require().Len(notifications, 1)
	s.Require().False(notifications[0].Read)
}
