package protocol

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/protocol/contacts"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	"github.com/status-im/status-go/internal/protocol/requests"
)

type MessengerThreadsSuite struct {
	MessengerBaseTestSuite
}

func TestMessengerThreadsSuite(t *testing.T) {
	suite.Run(t, new(MessengerThreadsSuite))
}

func (s *MessengerThreadsSuite) SetupTest() {
	s.MessengerBaseTestSuite.SetupTest()
	// Enable threads feature for tests
	s.m.featureFlags.Threads = true
}

func (s *MessengerThreadsSuite) createJoinedOneToOneThreadChat() *Chat {
	receiver := s.newMessenger()
	receiver.featureFlags.Threads = true

	receiverChat := CreateOneToOneChat("thread-sender", &s.m.identity.PublicKey, receiver.getTimesource())
	s.Require().NoError(receiver.SaveChat(receiverChat))
	_, err := receiver.Join(receiverChat)
	s.Require().NoError(err)

	senderChat := CreateOneToOneChat("thread-receiver", &receiver.identity.PublicKey, s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(senderChat))
	_, err = s.m.Join(senderChat)
	s.Require().NoError(err)

	return senderChat
}

func (s *MessengerThreadsSuite) TestCreateThreadRequiresParentMessage() {
	chat := CreateOneToOneChat("test-user", &s.m.identity.PublicKey, s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat))

	// Attempt to create thread for non-existent parent
	_, err := s.m.CreateThread(chat.ID, "non-existent-parent")
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "parent message not found")
}

func (s *MessengerThreadsSuite) TestSenderCanCreateThreadRequiresPrivilegedCommunityMember() {
	communityResponse, err := s.m.CreateCommunity(&requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
		Name:        "test community",
		Color:       "#ffffff",
		Description: "test community description",
	}, false)
	s.Require().NoError(err)

	community := communityResponse.Communities()[0]
	chatResponse, err := s.m.CreateCommunityChat(community.ID(), &protobuf.CommunityChat{
		Permissions: &protobuf.CommunityPermissions{Access: protobuf.CommunityPermissions_AUTO_ACCEPT},
		Identity:    &protobuf.ChatIdentity{DisplayName: "test chat"},
	})
	s.Require().NoError(err)
	chat := chatResponse.Chats()[0]

	allowed, err := s.m.senderCanCreateThread(chat, &s.m.identity.PublicKey)
	s.Require().NoError(err)
	s.Require().True(allowed)

	nonMember, err := crypto.GenerateKey()
	s.Require().NoError(err)
	allowed, err = s.m.senderCanCreateThread(chat, &nonMember.PublicKey)
	s.Require().NoError(err)
	s.Require().False(allowed)
}

func (s *MessengerThreadsSuite) TestCreateThreadSucceedsWithExistingParent() {
	chat := CreateOneToOneChat("test-user", &s.m.identity.PublicKey, s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat))

	// Save parent message
	parentMsg := buildTestMessage(*chat)
	parentMsg.ID = "parent-id"
	parentMsg.Text = "This is a test thread message"
	parentMsg.ChatMessage.Text = "This is a test thread message"
	s.Require().NoError(s.m.SaveMessages([]*common.Message{parentMsg}))

	// Create thread
	response, err := s.m.CreateThread(chat.ID, "parent-id")
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Threads(), 1)

	thread := response.Threads()[0]
	s.Require().Equal("parent-id", thread.ThreadID)
	s.Require().Equal(chat.ID, thread.ChatID)
	s.Require().Equal("parent-id", thread.ParentMessageID)
	// Name should be normalized from parent text (trimmed to 40 chars)
	s.Require().Equal("This is a test thread message", thread.Name)
}

func (s *MessengerThreadsSuite) TestStartThreadFromNewMessageWithExplicitName() {
	chat := s.createJoinedOneToOneThreadChat()

	message := buildTestMessage(*chat)
	message.Text = "Full message in the new thread"
	message.ChatMessage.Text = message.Text

	response, err := s.m.StartThreadFromNewMessage(context.Background(), &requests.StartThreadFromNewMessage{
		Message:    message,
		ThreadName: "  A custom\nthread name  ",
	})
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 2)
	s.Require().Len(response.Threads(), 1)

	thread := response.Threads()[0]
	s.Require().Equal("A custom thread name", thread.Name)

	var root *common.Message
	for _, responseMessage := range response.Messages() {
		if responseMessage.ID == thread.ThreadID {
			root = responseMessage
			break
		}
	}
	s.Require().NotNil(root)
	s.Require().Equal("A custom thread name", root.Text)
	s.Require().Equal(protobuf.ChatMessage_TEXT_PLAIN, root.ContentType)
	s.Require().Empty(root.GetThreadId())
	s.Require().Empty(root.ResponseTo)

	s.Require().Equal(thread.ThreadID, message.GetThreadId())
	s.Require().Equal(thread.ThreadID, message.ResponseTo)

	threadMessages, cursor, err := s.m.MessageByChatID(chat.ID, thread.ThreadID, "", 10)
	s.Require().NoError(err)
	s.Require().Empty(cursor)
	s.Require().Len(threadMessages, 1)
	s.Require().Equal(message.ID, threadMessages[0].ID)
}

func (s *MessengerThreadsSuite) TestStartThreadFromNewMessageUsesNormalizedMessageTextWhenNameOmitted() {
	chat := s.createJoinedOneToOneThreadChat()

	message := buildTestMessage(*chat)
	message.Text = "  " + strings.Repeat("é", 51) + "  "
	message.ChatMessage.Text = message.Text

	response, err := s.m.StartThreadFromNewMessage(context.Background(), &requests.StartThreadFromNewMessage{Message: message})
	s.Require().NoError(err)
	s.Require().Len(response.Threads(), 1)

	expectedRootText := strings.Repeat("é", maxThreadNameLength)
	thread := response.Threads()[0]
	s.Require().Equal(expectedRootText, thread.Name)
	for _, responseMessage := range response.Messages() {
		if responseMessage.ID == thread.ThreadID {
			s.Require().Equal(expectedRootText, responseMessage.Text)
			return
		}
	}
	s.T().Fatal("thread root message was not returned")
}

func (s *MessengerThreadsSuite) TestStartThreadFromNewMessagePreservesReplyPreviews() {
	chat := s.createJoinedOneToOneThreadChat()

	message := buildTestMessage(*chat)
	message.Text = "First message with previews"
	message.ChatMessage.Text = message.Text
	message.EnsName = "alice.eth"
	message.LinkPreviews = []common.LinkPreview{{
		Type:  protobuf.UnfurledLink_LINK,
		URL:   "https://example.com",
		Title: "Example",
	}}
	message.StatusLinkPreviews = []common.StatusLinkPreview{{
		URL: "https://status.app/u/alice",
		Contact: &common.StatusContactLinkPreview{
			PublicKey:   crypto.PubkeyToHex(&s.m.identity.PublicKey),
			DisplayName: "Alice",
		},
	}}

	response, err := s.m.StartThreadFromNewMessage(context.Background(), &requests.StartThreadFromNewMessage{Message: message})
	s.Require().NoError(err)
	s.Require().Len(response.Threads(), 1)

	threadID := response.Threads()[0].ThreadID
	threadMessages, cursor, err := s.m.MessageByChatID(chat.ID, threadID, "", 10)
	s.Require().NoError(err)
	s.Require().Empty(cursor)
	s.Require().Len(threadMessages, 1)

	reply := threadMessages[0]
	s.Require().Equal(message.ID, reply.ID)
	s.Require().Len(reply.UnfurledLinks, 1)
	s.Require().Equal("https://example.com", reply.UnfurledLinks[0].Url)
	s.Require().Equal("Example", reply.UnfurledLinks[0].Title)
	s.Require().NotNil(reply.UnfurledStatusLinks)
	s.Require().Len(reply.UnfurledStatusLinks.UnfurledStatusLinks, 1)
	statusPreview := reply.UnfurledStatusLinks.UnfurledStatusLinks[0]
	s.Require().Equal("https://status.app/u/alice", statusPreview.Url)
	s.Require().NotNil(statusPreview.GetContact())
	s.Require().Equal("Alice", statusPreview.GetContact().DisplayName)
}

func (s *MessengerThreadsSuite) TestStartThreadFromNewMessageRequiresThreadNameOrMessageText() {
	chat := CreateOneToOneChat("test-user", &s.m.identity.PublicKey, s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat))

	message := buildTestMessage(*chat)
	message.Text = " \n "
	message.ChatMessage.Text = message.Text

	_, err := s.m.StartThreadFromNewMessage(context.Background(), &requests.StartThreadFromNewMessage{Message: message})
	s.Require().EqualError(err, "thread name or message text is required")

	messages, cursor, err := s.m.MessageByChatID(chat.ID, "", "", 10)
	s.Require().NoError(err)
	s.Require().Empty(cursor)
	s.Require().Empty(messages)
}

func (s *MessengerThreadsSuite) TestCreateThreadFailsWhenAlreadyExists() {
	chat := CreateOneToOneChat("test-user", &s.m.identity.PublicKey, s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat))

	// Save parent message
	parentMsg := buildTestMessage(*chat)
	parentMsg.ID = "parent-id"
	parentMsg.Text = "Parent"
	parentMsg.ChatMessage.Text = "Parent"
	s.Require().NoError(s.m.SaveMessages([]*common.Message{parentMsg}))

	// Create thread first time
	response, err := s.m.CreateThread(chat.ID, "parent-id")
	s.Require().NoError(err)
	s.Require().Len(response.Threads(), 1)

	// Attempt to create thread again
	_, err = s.m.CreateThread(chat.ID, "parent-id")
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "thread already exists for this message")
}

func (s *MessengerThreadsSuite) TestCreateThreadFailsWhenFeatureFlagDisabled() {
	s.m.featureFlags.Threads = false

	chat := CreateOneToOneChat("test-user", &s.m.identity.PublicKey, s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat))

	// Save parent message
	parentMsg := buildTestMessage(*chat)
	parentMsg.ID = "parent-id"
	parentMsg.Text = "Parent"
	parentMsg.ChatMessage.Text = "Parent"
	s.Require().NoError(s.m.SaveMessages([]*common.Message{parentMsg}))

	// Attempt to create thread when disabled
	_, err := s.m.CreateThread(chat.ID, "parent-id")
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "threads feature is disabled")
}

func (s *MessengerThreadsSuite) TestThreadsByChatID() {
	chat := CreateOneToOneChat("test-user", &s.m.identity.PublicKey, s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat))

	// Save two parent messages
	parent1 := buildTestMessage(*chat)
	parent1.ID = "parent-1"
	parent1.Text = "First thread"
	parent1.ChatMessage.Text = "First thread"
	parent1.Clock = 1

	parent2 := buildTestMessage(*chat)
	parent2.ID = "parent-2"
	parent2.Text = "Second thread"
	parent2.ChatMessage.Text = "Second thread"
	parent2.Clock = 2

	s.Require().NoError(s.m.SaveMessages([]*common.Message{parent1, parent2}))

	// Create two threads
	_, err := s.m.CreateThread(chat.ID, "parent-1")
	s.Require().NoError(err)
	_, err = s.m.CreateThread(chat.ID, "parent-2")
	s.Require().NoError(err)

	// List threads
	threads, err := s.m.ThreadsByChatID(chat.ID)
	s.Require().NoError(err)
	s.Require().Len(threads, 2)
	s.Require().Equal("parent-2", threads[0].ThreadID)
	s.Require().Equal("parent-1", threads[1].ThreadID)

	// Verify thread data
	threadIDs := make(map[string]*Thread)
	for _, thread := range threads {
		threadIDs[thread.ThreadID] = thread
	}
	s.Require().Contains(threadIDs, "parent-1")
	s.Require().Contains(threadIDs, "parent-2")
	s.Require().Equal("First thread", threadIDs["parent-1"].Name)
	s.Require().Equal("Second thread", threadIDs["parent-2"].Name)
}

func (s *MessengerThreadsSuite) TestThreadsByChatIDs() {
	otherKey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	thirdKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	chat1 := CreateOneToOneChat("test-user-1", &s.m.identity.PublicKey, s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat1))
	chat2 := CreateOneToOneChat("test-user-2", &otherKey.PublicKey, s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat2))
	chatWithoutThreads := CreateOneToOneChat("test-user-3", &thirdKey.PublicKey, s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chatWithoutThreads))

	parent1 := buildTestMessage(*chat1)
	parent1.ID = "parent-1"
	parent1.Text = "First thread"
	parent1.ChatMessage.Text = "First thread"

	parent2 := buildTestMessage(*chat2)
	parent2.ID = "parent-2"
	parent2.Text = "Second thread"
	parent2.ChatMessage.Text = "Second thread"

	s.Require().NoError(s.m.SaveMessages([]*common.Message{parent1, parent2}))

	_, err = s.m.CreateThread(chat1.ID, "parent-1")
	s.Require().NoError(err)
	_, err = s.m.CreateThread(chat2.ID, "parent-2")
	s.Require().NoError(err)

	threads, err := s.m.ThreadsByChatIDs([]string{chat1.ID, chat2.ID, chatWithoutThreads.ID})
	s.Require().NoError(err)
	s.Require().Len(threads, 2)

	byChat := make(map[string]*Thread)
	for _, thread := range threads {
		byChat[thread.ChatID] = thread
	}
	s.Require().Equal("parent-1", byChat[chat1.ID].ThreadID)
	s.Require().Equal("parent-2", byChat[chat2.ID].ThreadID)
	s.Require().NotContains(byChat, chatWithoutThreads.ID)
	s.Require().Zero(byChat[chat1.ID].MessagesCount)
	s.Require().Zero(byChat[chat1.ID].ParticipantsCount)
	s.Require().Empty(byChat[chat1.ID].ParticipantsPreviewIDs)
	s.Require().Nil(byChat[chat1.ID].LastMessage)

	// A chat that was never requested must not leak in.
	threads, err = s.m.ThreadsByChatIDs([]string{chat1.ID})
	s.Require().NoError(err)
	s.Require().Len(threads, 1)
	s.Require().Equal(chat1.ID, threads[0].ChatID)

	threads, err = s.m.ThreadsByChatIDs(nil)
	s.Require().NoError(err)
	s.Require().Empty(threads)
}

func (s *MessengerThreadsSuite) TestThreadsByChatIDsBatchesAndGloballyOrdersResults() {
	chatIDs := make([]string, 0, 1000)
	chatIDs = append(chatIDs, "chat-z")
	for i := 0; i < 998; i++ {
		chatIDs = append(chatIDs, fmt.Sprintf("chat-middle-%03d", i))
	}
	chatIDs = append(chatIDs, "chat-a")

	s.Require().NoError(s.m.persistence.UpsertThread("thread-z", "chat-z", "parent-z", "Zeta"))
	s.Require().NoError(s.m.persistence.UpsertThread("thread-a", "chat-a", "parent-a", "Alpha"))

	threads, err := s.m.ThreadsByChatIDs(chatIDs)
	s.Require().NoError(err)
	s.Require().Equal([]string{"chat-a", "chat-z"}, []string{threads[0].ChatID, threads[1].ChatID})
}

func (s *MessengerThreadsSuite) TestThreadsIncludeUnreadCounts() {
	chat := CreateOneToOneChat("test-user", &s.m.identity.PublicKey, s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat))

	parentMsg := buildTestMessage(*chat)
	parentMsg.ID = "parent-id"
	parentMsg.From = "parent-author"
	parentMsg.Text = "Parent"
	parentMsg.ChatMessage.Text = "Parent"
	s.Require().NoError(s.m.SaveMessages([]*common.Message{parentMsg}))

	_, err := s.m.CreateThread(chat.ID, "parent-id")
	s.Require().NoError(err)

	threadID := "parent-id"
	unreadReply := buildTestMessage(*chat)
	unreadReply.ID = "reply-unread"
	unreadReply.Text = "Unread reply"
	unreadReply.ChatMessage.Text = "Unread reply"
	unreadReply.ChatMessage.ThreadId = &threadID
	unreadReply.ResponseTo = "parent-id"
	unreadReply.Mentioned = true
	unreadReply.Seen = false

	seenReply := buildTestMessage(*chat)
	seenReply.ID = "reply-seen"
	seenReply.Text = "Seen reply"
	seenReply.ChatMessage.Text = "Seen reply"
	seenReply.ChatMessage.ThreadId = &threadID
	seenReply.ResponseTo = "parent-id"
	seenReply.Mentioned = true
	seenReply.Seen = true

	s.Require().NoError(s.m.SaveMessages([]*common.Message{unreadReply, seenReply}))

	threads, err := s.m.ThreadsByChatID(chat.ID)
	s.Require().NoError(err)
	s.Require().Len(threads, 1)
	s.Require().Equal(uint(1), threads[0].UnviewedMessagesCount)
	s.Require().Equal(uint(1), threads[0].UnviewedMentionsCount)
	s.Require().Zero(threads[0].MessagesCount)
	s.Require().Empty(threads[0].ParticipantsPreviewIDs)
	s.Require().Nil(threads[0].LastMessage)

	thread, err := s.m.persistence.ThreadWithSummaryByID(chat.ID, threadID)
	s.Require().NoError(err)
	s.Require().Equal(uint(1), thread.UnviewedMessagesCount)
	s.Require().Equal(uint(1), thread.UnviewedMentionsCount)
	s.Require().Equal(uint(3), thread.MessagesCount)
	s.Require().NotEmpty(thread.ParticipantsPreviewIDs)
	s.Require().NotNil(thread.LastMessage)
}

func (s *MessengerThreadsSuite) TestThreadSummariesByParentMessageIDsHonorsParticipantPreviewLimit() {
	chat := CreateOneToOneChat("test-user", &s.m.identity.PublicKey, s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat))

	parentMsg := buildTestMessage(*chat)
	parentMsg.ID = "parent-id"
	parentMsg.From = "creator"
	s.Require().NoError(s.m.SaveMessages([]*common.Message{parentMsg}))
	s.Require().NoError(s.m.persistence.UpsertThread("thread-id", chat.ID, parentMsg.ID, "Thread"))

	threadID := "thread-id"
	var replies []*common.Message
	for index, participant := range []string{"alice", "bob", "carol"} {
		reply := buildTestMessage(*chat)
		reply.ID = participant
		reply.From = participant
		reply.Clock = uint64(index + 1)
		reply.ChatMessage.ThreadId = &threadID
		replies = append(replies, reply)
	}
	s.Require().NoError(s.m.SaveMessages(replies))

	threads, err := s.m.ThreadSummariesByParentMessageIDs(chat.ID, []string{parentMsg.ID}, 2)
	s.Require().NoError(err)
	s.Require().Len(threads, 1)
	s.Require().Equal(uint(4), threads[0].ParticipantsCount)
	s.Require().Equal([]string{"creator", "carol"}, threads[0].ParticipantsPreviewIDs)
}

func (s *MessengerThreadsSuite) TestMessagesWithThreadSummariesMatchesPaginatedReads() {
	chat := CreateOneToOneChat("test-user", &s.m.identity.PublicKey, s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat))
	for i := 0; i < 5; i++ {
		parent := buildTestMessage(*chat)
		parent.ID = fmt.Sprintf("parent-%d", i)
		parent.Clock = uint64(i + 1)
		parent.From = "creator"
		s.Require().NoError(s.m.SaveMessages([]*common.Message{parent}))
		if i == 0 {
			continue // Include a page with no threads.
		}
		threadID := fmt.Sprintf("thread-%d", i)
		s.Require().NoError(s.m.persistence.UpsertThread(threadID, chat.ID, parent.ID, "Thread"))
		for j := 0; j < 3; j++ {
			reply := buildTestMessage(*chat)
			reply.ID = fmt.Sprintf("reply-%d-%d", i, j)
			reply.From = fmt.Sprintf("author-%d", j)
			reply.Clock = uint64(10 + j)
			reply.ChatMessage.ThreadId = &threadID
			s.Require().NoError(s.m.SaveMessages([]*common.Message{reply}))
		}
	}

	cursor := ""
	seen := 0
	for {
		messages, nextCursor, err := s.m.MessageByChatID(chat.ID, "", cursor, 2)
		s.Require().NoError(err)
		page, err := s.m.MessagesWithThreadSummaries(chat.ID, cursor, 2, 2)
		s.Require().NoError(err)
		s.Require().Equal(messages, page.Messages)
		s.Require().Equal(nextCursor, page.Cursor)
		s.Require().Empty(page.ThreadSummariesError)
		parentIDs := make([]string, 0, len(messages))
		for _, message := range messages {
			s.Require().NotNil(message.HasThread)
			s.Require().Equal(message.ID != "parent-0", *message.HasThread)
			parentIDs = append(parentIDs, message.ID)
		}
		threads, err := s.m.ThreadSummariesByParentMessageIDs(chat.ID, parentIDs, 2)
		s.Require().NoError(err)
		s.Require().Equal(threads, page.Threads)
		for _, thread := range page.Threads {
			s.Require().Equal(uint(4), thread.MessagesCount)
			s.Require().Equal(uint(4), thread.ParticipantsCount)
			s.Require().Len(thread.ParticipantsPreviewIDs, 2)
		}
		seen += len(page.Messages)
		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}
	s.Require().Equal(5, seen)
}

func (s *MessengerThreadsSuite) TestMessagesWithThreadSummariesEmptyAndDisabled() {
	chat := CreateOneToOneChat("test-user", &s.m.identity.PublicKey, s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat))
	page, err := s.m.MessagesWithThreadSummaries(chat.ID, "", 20, 6)
	s.Require().NoError(err)
	s.Require().Empty(page.Messages)
	s.Require().NotNil(page.Threads)
	s.Require().Empty(page.Threads)
	parent := buildTestMessage(*chat)
	parent.ID = "parent"
	s.Require().NoError(s.m.SaveMessages([]*common.Message{parent}))
	s.Require().NoError(s.m.persistence.UpsertThread("thread", chat.ID, parent.ID, "Thread"))
	s.m.featureFlags.Threads = false
	page, err = s.m.MessagesWithThreadSummaries(chat.ID, "", 20, 6)
	s.Require().NoError(err)
	s.Require().Len(page.Messages, 1)
	s.Require().Empty(page.Threads)
	s.Require().Empty(page.ThreadSummariesError)
	_, err = s.m.MessagesWithThreadSummaries("missing-chat", "", 20, 6)
	s.Require().ErrorIs(err, ErrChatNotFound)
}

func (s *MessengerThreadsSuite) TestMessagesWithThreadSummariesPreservesMessagesOnSummaryFailure() {
	chat := CreateOneToOneChat("test-user", &s.m.identity.PublicKey, s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat))
	parent := buildTestMessage(*chat)
	parent.ID = "parent"
	s.Require().NoError(s.m.SaveMessages([]*common.Message{parent}))
	// Break only the summary query in this disposable test database.
	_, err := s.m.persistence.db.Exec("DROP TABLE threads")
	s.Require().NoError(err)
	page, err := s.m.MessagesWithThreadSummaries(chat.ID, "", 20, 6)
	s.Require().NoError(err)
	s.Require().Len(page.Messages, 1)
	s.Require().Equal(parent.ID, page.Messages[0].ID)
	s.Require().Empty(page.Threads)
	s.Require().NotEmpty(page.ThreadSummariesError)
}

func (s *MessengerThreadsSuite) TestMessagesWithThreadSummariesSkipsPagesWithoutThreads() {
	chat := CreateOneToOneChat("test-user", &s.m.identity.PublicKey, s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat))
	parent := buildTestMessage(*chat)
	parent.ID = "plain-parent"
	s.Require().NoError(s.m.SaveMessages([]*common.Message{parent}))
	// Presence still works, but any summary metadata query would fail.
	_, err := s.m.persistence.db.Exec("ALTER TABLE threads RENAME COLUMN name TO unavailable_name")
	s.Require().NoError(err)
	page, err := s.m.MessagesWithThreadSummaries(chat.ID, "", 20, 6)
	s.Require().NoError(err)
	s.Require().Len(page.Messages, 1)
	s.Require().NotNil(page.Messages[0].HasThread)
	s.Require().False(*page.Messages[0].HasThread)
	s.Require().Empty(page.Threads)
	s.Require().Empty(page.ThreadSummariesError)
}

func (s *MessengerThreadsSuite) TestReactionsByMessageIDsSupportsTimeline() {
	s.Require().NoError(s.m.SaveChat(&Chat{ID: "timeline", ChatType: ChatTypeTimeline}))
	profileID := "@" + contacts.ContactIDFromPublicKey(&s.m.identity.PublicKey)
	s.Require().NoError(s.m.SaveMessages([]*common.Message{{
		ID: "profile-message", LocalChatID: profileID, From: testPK,
		ChatMessage: &protobuf.ChatMessage{Text: "Profile", Clock: 1},
	}}))
	s.Require().NoError(s.m.persistence.SaveEmojiReaction(&EmojiReaction{
		From: testPK, LocalChatID: profileID,
		EmojiReaction: &protobuf.EmojiReaction{MessageId: "profile-message", ChatId: profileID, Emoji: "1f600"},
	}))
	s.Require().NoError(s.m.persistence.SaveEmojiReaction(&EmojiReaction{
		From: "unrelated", LocalChatID: "unrelated-chat",
		EmojiReaction: &protobuf.EmojiReaction{MessageId: "profile-message", ChatId: "unrelated-chat", Emoji: "1f600"},
	}))
	reactions, err := s.m.EmojiReactionsByChatIDMessageIDs("timeline", []string{"profile-message"})
	s.Require().NoError(err)
	s.Require().Len(reactions, 1)
	s.Require().Equal(profileID, reactions[0].LocalChatID)
	_, err = s.m.EmojiReactionsByChatIDMessageIDs("missing-chat", []string{"profile-message"})
	s.Require().ErrorIs(err, ErrChatNotFound)
}

func (s *MessengerThreadsSuite) TestThreadSummaryUsesExplicitThreadAndParentIDs() {
	chat := CreateOneToOneChat("test-user", &s.m.identity.PublicKey, s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat))

	parentMsg := buildTestMessage(*chat)
	parentMsg.ID = "parent-id"
	parentMsg.From = "parent-author"
	parentMsg.Text = "Parent"
	parentMsg.ChatMessage.Text = "Parent"
	s.Require().NoError(s.m.SaveMessages([]*common.Message{parentMsg}))
	s.Require().NoError(s.m.persistence.UpsertThread("thread-id", chat.ID, parentMsg.ID, "Thread"))

	threadID := "thread-id"
	reply := buildTestMessage(*chat)
	reply.ID = "reply-id"
	reply.From = "reply-author"
	reply.Text = "Latest reply"
	reply.ChatMessage.Text = "Latest reply"
	reply.ChatMessage.ThreadId = &threadID
	s.Require().NoError(s.m.SaveMessages([]*common.Message{reply}))

	thread, err := s.m.persistence.ThreadWithSummaryByID(chat.ID, threadID)
	s.Require().NoError(err)
	s.Require().Equal(uint(2), thread.MessagesCount)
	s.Require().Equal([]string{"parent-author", "reply-author"}, thread.ParticipantsPreviewIDs)
	s.Require().NotNil(thread.LastMessage)
	s.Require().Equal("reply-author", thread.LastMessage.From)
	s.Require().Equal("Latest reply", thread.LastMessage.Text)
}

func (s *MessengerThreadsSuite) TestAffectedThreadSummaryReflectsLastMessageEditAndDelete() {
	chat := CreateOneToOneChat("test-user", &s.m.identity.PublicKey, s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat))

	parent := buildTestMessage(*chat)
	parent.ID = "parent-id"
	parent.From = "creator"
	parent.Clock = 1
	s.Require().NoError(s.m.SaveMessages([]*common.Message{parent}))
	s.Require().NoError(s.m.persistence.UpsertThread("thread-id", chat.ID, parent.ID, "Thread"))

	threadID := "thread-id"
	previousReply := buildTestMessage(*chat)
	previousReply.ID = "previous-reply"
	previousReply.From = "alice"
	previousReply.Text = "Previous reply"
	previousReply.ChatMessage.Text = "Previous reply"
	previousReply.Clock = 2
	previousReply.ChatMessage.ThreadId = &threadID

	latestReply := buildTestMessage(*chat)
	latestReply.ID = "latest-reply"
	latestReply.From = "bob"
	latestReply.Text = "Original text"
	latestReply.ChatMessage.Text = "Original text"
	latestReply.Clock = 3
	latestReply.ChatMessage.ThreadId = &threadID
	s.Require().NoError(s.m.SaveMessages([]*common.Message{previousReply, latestReply}))

	latestReply.Text = "Edited text"
	latestReply.ChatMessage.Text = "Edited text"
	s.Require().NoError(s.m.SaveMessages([]*common.Message{latestReply}))

	response := &MessengerResponse{}
	s.Require().NoError(s.m.addAffectedThreadToResponse(response, latestReply))
	s.Require().Len(response.Threads(), 1)
	s.Require().Equal("Edited text", response.Threads()[0].LastMessage.Text)

	latestReply.Deleted = true
	s.Require().NoError(s.m.SaveMessages([]*common.Message{latestReply}))
	response = &MessengerResponse{}
	s.Require().NoError(s.m.addAffectedThreadToResponse(response, latestReply))
	s.Require().Len(response.Threads(), 1)
	s.Require().Equal(uint(2), response.Threads()[0].MessagesCount)
	s.Require().Equal(uint(2), response.Threads()[0].ParticipantsCount)
	s.Require().Equal("Previous reply", response.Threads()[0].LastMessage.Text)
}

func (s *MessengerThreadsSuite) TestThreadSummaryOrdersParticipantsByMostRecentParticipation() {
	chat := CreateOneToOneChat("test-user", &s.m.identity.PublicKey, s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat))

	parentMsg := buildTestMessage(*chat)
	parentMsg.ID = "parent-id"
	parentMsg.From = "parent-author"
	s.Require().NoError(s.m.SaveMessages([]*common.Message{parentMsg}))
	s.Require().NoError(s.m.persistence.UpsertThread("thread-id", chat.ID, parentMsg.ID, "Thread"))

	threadID := "thread-id"
	olderReply := buildTestMessage(*chat)
	olderReply.ID = "older-reply"
	olderReply.From = "older-participant"
	olderReply.Clock = 10
	olderReply.ChatMessage.ThreadId = &threadID

	recentReply := buildTestMessage(*chat)
	recentReply.ID = "recent-reply"
	recentReply.From = "recent-participant"
	recentReply.Clock = 30
	recentReply.ChatMessage.ThreadId = &threadID

	latestOlderReply := buildTestMessage(*chat)
	latestOlderReply.ID = "latest-older-reply"
	latestOlderReply.From = "older-participant"
	latestOlderReply.Clock = 20
	latestOlderReply.ChatMessage.ThreadId = &threadID

	replies := []*common.Message{olderReply, recentReply, latestOlderReply}
	for index, participant := range []string{"participant-3", "participant-4", "participant-5", "participant-6", "participant-7"} {
		reply := buildTestMessage(*chat)
		reply.ID = participant
		reply.From = participant
		reply.Clock = uint64(19 - index)
		reply.ChatMessage.ThreadId = &threadID
		replies = append(replies, reply)
	}

	s.Require().NoError(s.m.SaveMessages(replies))

	thread, err := s.m.persistence.ThreadWithSummaryByID(chat.ID, threadID)
	s.Require().NoError(err)
	s.Require().Equal(uint(8), thread.ParticipantsCount)
	s.Require().Equal([]string{
		"parent-author",
		"recent-participant",
		"older-participant",
		"participant-3",
		"participant-4",
		"participant-5",
	}, thread.ParticipantsPreviewIDs)
}

func (s *MessengerThreadsSuite) TestThreadUnreadCountsIgnoreInvisibleRepliesAndCountOneToOneRepliesAsMentions() {
	chat := CreateOneToOneChat("test-user", &s.m.identity.PublicKey, s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat))

	parent := buildTestMessage(*chat)
	parent.ID = "parent-id"
	parent.Text = "Parent"
	parent.ChatMessage.Text = "Parent"
	s.Require().NoError(s.m.SaveMessages([]*common.Message{parent}))

	_, err := s.m.CreateThread(chat.ID, parent.ID)
	s.Require().NoError(err)

	threadID := parent.ID
	newReply := func(id string) *common.Message {
		reply := buildTestMessage(*chat)
		reply.ID = id
		reply.Seen = false
		reply.ChatMessage.ThreadId = &threadID
		return reply
	}

	visible := newReply("visible")
	hidden := newReply("hidden")
	deleted := newReply("deleted")
	deleted.Deleted = true
	deletedForMe := newReply("deleted-for-me")
	deletedForMe.DeletedForMe = true
	s.Require().NoError(s.m.SaveMessages([]*common.Message{visible, hidden, deleted, deletedForMe}))

	_, err = s.m.persistence.db.Exec(`UPDATE user_messages SET hide = 1 WHERE id = ?`, hidden.ID)
	s.Require().NoError(err)

	thread, err := s.m.persistence.ThreadByID(chat.ID, threadID)
	s.Require().NoError(err)
	s.Require().Equal(uint(1), thread.UnviewedMessagesCount)
	s.Require().Equal(uint(1), thread.UnviewedMentionsCount)

	threads, err := s.m.ThreadsByChatID(chat.ID)
	s.Require().NoError(err)
	s.Require().Len(threads, 1)
	s.Require().Equal(uint(1), threads[0].UnviewedMessagesCount)
	s.Require().Equal(uint(1), threads[0].UnviewedMentionsCount)

	batchThreads, err := s.m.ThreadsByChatIDs([]string{chat.ID})
	s.Require().NoError(err)
	s.Require().Len(batchThreads, 1)
	s.Require().Equal(uint(1), batchThreads[0].UnviewedMessagesCount)
	s.Require().Equal(uint(1), batchThreads[0].UnviewedMentionsCount)

	paginatedThreads, err := s.m.ThreadSummariesByParentMessageIDs(chat.ID, []string{parent.ID}, defaultThreadParticipantPreviewLimit)
	s.Require().NoError(err)
	s.Require().Len(paginatedThreads, 1)
	s.Require().Equal(uint(1), paginatedThreads[0].UnviewedMessagesCount)
	s.Require().Equal(uint(1), paginatedThreads[0].UnviewedMentionsCount)
}

func (s *MessengerThreadsSuite) TestAddThreadsToResponseUsesLocalChatIDForOneToOneChats() {
	receiver := s.m
	sender := s.newMessenger()
	receiver.featureFlags.Threads = true
	sender.featureFlags.Threads = true

	receiverChat := CreateOneToOneChat("sender", &sender.identity.PublicKey, receiver.getTimesource())
	senderChat := CreateOneToOneChat("receiver", &receiver.identity.PublicKey, sender.getTimesource())
	s.Require().NotEqual(receiverChat.ID, senderChat.ID)

	threadID := "parent-id"
	receivedReply := buildTestMessage(*senderChat)
	receivedReply.ID = "reply-id"
	receivedReply.Text = "Unread reply"
	receivedReply.ChatMessage.Text = "Unread reply"
	receivedReply.ChatMessage.ThreadId = &threadID
	receivedReply.LocalChatID = receiverChat.ID
	receivedReply.Mentioned = true
	receivedReply.Seen = false
	receivedReply.ThreadMetadataCreationAuthorized = true

	s.Require().NoError(receiver.persistence.SaveMessages([]*common.Message{receivedReply}))

	response := &MessengerResponse{}
	s.Require().NoError(receiver.addThreadsToResponse(response, []*common.Message{receivedReply}))
	s.Require().Len(response.Threads(), 1)
	s.Require().Equal(receiverChat.ID, response.Threads()[0].ChatID)
	s.Require().Equal(threadID, response.Threads()[0].ThreadID)
	s.Require().Equal(uint(1), response.Threads()[0].UnviewedMessagesCount)
	s.Require().Equal(uint(1), response.Threads()[0].UnviewedMentionsCount)
}

func (s *MessengerThreadsSuite) TestMessagesByThreadID() {
	chat := CreateOneToOneChat("test-user", &s.m.identity.PublicKey, s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat))

	// Save parent message
	parentMsg := buildTestMessage(*chat)
	parentMsg.ID = "parent-id"
	parentMsg.Text = "Parent"
	parentMsg.ChatMessage.Text = "Parent"
	s.Require().NoError(s.m.SaveMessages([]*common.Message{parentMsg}))

	// Create thread
	_, err := s.m.CreateThread(chat.ID, "parent-id")
	s.Require().NoError(err)

	// Save reply messages in thread
	threadID := "parent-id"
	reply1 := buildTestMessage(*chat)
	reply1.ID = "reply-1"
	reply1.Text = "First reply"
	reply1.ChatMessage.Text = "First reply"
	reply1.ChatMessage.ThreadId = &threadID
	reply1.ResponseTo = "parent-id"

	reply2 := buildTestMessage(*chat)
	reply2.ID = "reply-2"
	reply2.Text = "Second reply"
	reply2.ChatMessage.Text = "Second reply"
	reply2.ChatMessage.ThreadId = &threadID
	reply2.ResponseTo = "parent-id"

	s.Require().NoError(s.m.SaveMessages([]*common.Message{reply1, reply2}))

	// Retrieve thread messages
	msgs, cursor, err := s.m.MessageByChatID(chat.ID, "parent-id", "", 10)
	s.Require().NoError(err)
	s.Require().Len(msgs, 2)
	s.Require().Empty(cursor)

	// Verify message content
	for _, msg := range msgs {
		s.Require().Equal("parent-id", msg.GetThreadId())
		s.Require().True(msg.Text == "First reply" || msg.Text == "Second reply")
	}
}

func (s *MessengerThreadsSuite) TestSendChatMessageWithThread() {
	chat := CreateOneToOneChat("test-user", &s.m.identity.PublicKey, s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat))

	// Save parent message
	parentMsg := buildTestMessage(*chat)
	parentMsg.ID = "parent-id"
	parentMsg.Text = "Parent"
	parentMsg.ChatMessage.Text = "Parent"
	s.Require().NoError(s.m.SaveMessages([]*common.Message{parentMsg}))

	// Create thread
	_, err := s.m.CreateThread(chat.ID, "parent-id")
	s.Require().NoError(err)

	// Send message into thread
	threadID := "parent-id"
	msgToSend := buildTestMessage(*chat)
	msgToSend.Text = "Reply in thread"
	msgToSend.ChatMessage.Text = "Reply in thread"
	msgToSend.ChatMessage.ThreadId = &threadID

	response, err := s.m.SendChatMessage(context.Background(), msgToSend)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	// Verify thread still exists and can be retrieved
	threads, err := s.m.ThreadsByChatID(chat.ID)
	s.Require().NoError(err)
	s.Require().Len(threads, 1)
	s.Require().Equal("parent-id", threads[0].ThreadID)
}

func (s *MessengerThreadsSuite) TestSendMessageToThreadCreatesThreadIfNotExists() {
	chat := CreateOneToOneChat("test-user", &s.m.identity.PublicKey, s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat))

	// Save parent message
	parentMsg := buildTestMessage(*chat)
	parentMsg.ID = "parent-id"
	parentMsg.Text = "Parent"
	parentMsg.ChatMessage.Text = "Parent"
	s.Require().NoError(s.m.SaveMessages([]*common.Message{parentMsg}))

	// Verify thread doesn't exist initially
	threads, err := s.m.ThreadsByChatID(chat.ID)
	s.Require().NoError(err)
	s.Require().Len(threads, 0)

	// Create thread explicitly
	_, err = s.m.CreateThread(chat.ID, "parent-id")
	s.Require().NoError(err)

	// Now send message to that thread
	threadID := "parent-id"
	msgToSend := buildTestMessage(*chat)
	msgToSend.Text = "Reply in thread"
	msgToSend.ChatMessage.Text = "Reply in thread"
	msgToSend.ChatMessage.ThreadId = &threadID

	resp, err := s.m.SendChatMessage(context.Background(), msgToSend)
	s.Require().NoError(err)
	s.Require().NotNil(resp)

	// Thread should still exist
	threads, err = s.m.ThreadsByChatID(chat.ID)
	s.Require().NoError(err)
	s.Require().Len(threads, 1)
	s.Require().Equal("parent-id", threads[0].ThreadID)
}

func (s *MessengerThreadsSuite) TestSendThreadMessageRejectsUnsupportedChatWithExistingThread() {
	chat := CreatePublicChat("thread-unsupported-chat", s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat))

	threadID := "parent-id"
	s.Require().NoError(s.m.persistence.UpsertThread(threadID, chat.ID, threadID, "Parent"))

	message := buildTestMessage(*chat)
	message.ChatMessage.ThreadId = &threadID

	_, err := s.m.SendChatMessage(context.Background(), message)
	s.Require().ErrorIs(err, ErrThreadsNotSupportedForChatType)
}

func (s *MessengerThreadsSuite) TestReceivedThreadReplyDoesNotIncrementParentUnreadCount() {
	receiver := s.m
	receiver.featureFlags.Threads = true

	sender := s.newMessenger()
	sender.featureFlags.Threads = true

	receiverChat := CreateOneToOneChat("thread-unread-one-to-one-chat", &sender.identity.PublicKey, receiver.getTimesource())
	s.Require().NoError(receiver.SaveChat(receiverChat))
	_, err := receiver.Join(receiverChat)
	s.Require().NoError(err)

	senderChat := CreateOneToOneChat("thread-unread-one-to-one-chat", &receiver.identity.PublicKey, sender.getTimesource())
	s.Require().NoError(sender.SaveChat(senderChat))
	_, err = sender.Join(senderChat)
	s.Require().NoError(err)

	parentMsg := buildTestMessage(*senderChat)
	parentMsg.Text = "Parent message"
	parentMsg.ChatMessage.Text = "Parent message"

	parentResponse, err := sender.SendChatMessage(context.Background(), parentMsg)
	s.Require().NoError(err)
	s.Require().Len(parentResponse.Messages(), 1)
	parentID := parentResponse.Messages()[0].ID

	_, err = WaitOnMessengerResponse(receiver, func(response *MessengerResponse) bool {
		for _, msg := range response.Messages() {
			if msg.ID == parentID {
				return true
			}
		}

		return false
	}, "parent message not received")
	s.Require().NoError(err)

	_, err = receiver.MarkAllRead(context.Background(), receiverChat.ID)
	s.Require().NoError(err)
	receiverParentChat, ok := receiver.allChats.Load(receiverChat.ID)
	s.Require().True(ok)
	s.Require().Equal(uint(0), receiverParentChat.UnviewedMessagesCount)
	s.Require().Equal(uint(0), receiverParentChat.UnviewedMentionsCount)

	_, err = sender.CreateThread(senderChat.ID, parentID)
	s.Require().NoError(err)

	threadID := parentID
	threadReply := buildTestMessage(*senderChat)
	threadReply.Text = "Reply from thread"
	threadReply.ChatMessage.Text = "Reply from thread"
	threadReply.ChatMessage.ThreadId = &threadID
	threadReply.ResponseTo = parentID
	threadReply.Mentioned = true

	threadResponse, err := sender.SendChatMessage(context.Background(), threadReply)
	s.Require().NoError(err)

	var threadReplyID string
	for _, msg := range threadResponse.Messages() {
		if msg.Text == "Reply from thread" && msg.GetThreadId() == parentID {
			threadReplyID = msg.ID
			break
		}
	}
	s.Require().NotEmpty(threadReplyID)

	receiverResponse, err := WaitOnMessengerResponse(receiver, func(response *MessengerResponse) bool {
		for _, msg := range response.Messages() {
			if msg.ID == threadReplyID {
				s.Require().Equal(parentID, msg.GetThreadId())
				return true
			}
		}

		return false
	}, "thread reply not received")
	s.Require().NoError(err)

	receiverParentChat, ok = receiver.allChats.Load(receiverChat.ID)
	s.Require().True(ok)
	s.Require().Equal(uint(0), receiverParentChat.UnviewedMessagesCount)
	s.Require().Equal(uint(0), receiverParentChat.UnviewedMentionsCount)

	var responseChat *Chat
	for _, chat := range receiverResponse.Chats() {
		if chat.ID == receiverChat.ID {
			responseChat = chat
			break
		}
	}
	s.Require().NotNil(responseChat)
	s.Require().Equal(uint(0), responseChat.UnviewedMessagesCount)
	s.Require().Equal(uint(0), responseChat.UnviewedMentionsCount)

	var responseThread *Thread
	for _, thread := range receiverResponse.Threads() {
		if thread.ThreadID == parentID {
			responseThread = thread
			break
		}
	}
	s.Require().NotNil(responseThread)

	thread, err := receiver.persistence.ThreadByID(receiverChat.ID, parentID)
	s.Require().NoError(err)
	s.Require().Equal(parentID, thread.ThreadID)
}

func (s *MessengerThreadsSuite) TestStartThreadFromNewMessageIsReceivedWithThreadMetadata() {
	receiver := s.m
	sender := s.newMessenger()
	receiver.featureFlags.Threads = true
	sender.featureFlags.Threads = true
	chatID := "start-thread-from-new-message-one-to-one-chat"

	receiverChat := CreateOneToOneChat(chatID, &sender.identity.PublicKey, receiver.getTimesource())
	s.Require().NoError(receiver.SaveChat(receiverChat))
	_, err := receiver.Join(receiverChat)
	s.Require().NoError(err)

	senderChat := CreateOneToOneChat(chatID, &receiver.identity.PublicKey, sender.getTimesource())
	s.Require().NoError(sender.SaveChat(senderChat))
	_, err = sender.Join(senderChat)
	s.Require().NoError(err)

	message := buildTestMessage(*senderChat)
	message.Text = "First message in the new thread"
	message.ChatMessage.Text = message.Text

	senderResponse, err := sender.StartThreadFromNewMessage(context.Background(), &requests.StartThreadFromNewMessage{
		Message:    message,
		ThreadName: "New thread",
	})
	s.Require().NoError(err)
	s.Require().Len(senderResponse.Threads(), 1)

	threadID := senderResponse.Threads()[0].ThreadID
	_, err = WaitOnMessengerResponse(receiver, func(response *MessengerResponse) bool {
		var receivedRoot, receivedReply bool
		for _, receivedMessage := range response.Messages() {
			switch receivedMessage.ID {
			case threadID:
				receivedRoot = true
			case message.ID:
				receivedReply = true
			}
		}
		return receivedRoot && receivedReply
	}, "thread root and first reply not received")
	s.Require().NoError(err)

	thread, err := receiver.persistence.ThreadByID(receiverChat.ID, threadID)
	s.Require().NoError(err)
	s.Require().Equal("New thread", thread.Name)

	receivedReply, err := receiver.MessageByID(message.ID)
	s.Require().NoError(err)
	s.Require().Equal(threadID, receivedReply.GetThreadId())
	s.Require().Equal(threadID, receivedReply.ResponseTo)

	threadMessages, cursor, err := receiver.MessageByChatID(receiverChat.ID, threadID, "", 10)
	s.Require().NoError(err)
	s.Require().Empty(cursor)
	s.Require().Len(threadMessages, 1)
	s.Require().Equal(message.ID, threadMessages[0].ID)
}

func (s *MessengerThreadsSuite) TestMarkThreadReadClearsOnlyThreadMessages() {
	chat := CreateOneToOneChat("test-user", &s.m.identity.PublicKey, s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat))

	parentMsg := buildTestMessage(*chat)
	parentMsg.ID = "parent-id"
	parentMsg.Text = "Parent"
	parentMsg.ChatMessage.Text = "Parent"
	parentMsg.Seen = false
	s.Require().NoError(s.m.SaveMessages([]*common.Message{parentMsg}))

	_, err := s.m.CreateThread(chat.ID, "parent-id")
	s.Require().NoError(err)

	threadID := "parent-id"
	unreadReply := buildTestMessage(*chat)
	unreadReply.ID = "reply-unread"
	unreadReply.Text = "Unread reply"
	unreadReply.ChatMessage.Text = "Unread reply"
	unreadReply.ChatMessage.ThreadId = &threadID
	unreadReply.ResponseTo = "parent-id"
	unreadReply.Mentioned = true
	unreadReply.Seen = false

	olderUnreadReply := buildTestMessage(*chat)
	olderUnreadReply.ID = "reply-older-unread"
	olderUnreadReply.Text = "Older unread reply"
	olderUnreadReply.ChatMessage.Text = "Older unread reply"
	olderUnreadReply.ChatMessage.ThreadId = &threadID
	olderUnreadReply.ResponseTo = "parent-id"
	olderUnreadReply.Seen = false

	s.Require().NoError(s.m.SaveMessages([]*common.Message{olderUnreadReply, unreadReply}))

	thread, err := s.m.persistence.ThreadByID(chat.ID, threadID)
	s.Require().NoError(err)
	s.Require().Equal(uint(2), thread.UnviewedMessagesCount)
	s.Require().Equal(uint(2), thread.UnviewedMentionsCount)

	response, err := s.m.MarkThreadRead(context.Background(), chat.ID, threadID)
	s.Require().NoError(err)
	s.Require().Len(response.Threads(), 1)
	s.Require().Equal(uint(0), response.Threads()[0].UnviewedMessagesCount)
	s.Require().Equal(uint(0), response.Threads()[0].UnviewedMentionsCount)

	parentAfter, err := s.m.MessageByID("parent-id")
	s.Require().NoError(err)
	s.Require().False(parentAfter.Seen)

	replyAfter, err := s.m.MessageByID("reply-unread")
	s.Require().NoError(err)
	s.Require().True(replyAfter.Seen)

	olderReplyAfter, err := s.m.MessageByID("reply-older-unread")
	s.Require().NoError(err)
	s.Require().True(olderReplyAfter.Seen)

	thread, err = s.m.persistence.ThreadByID(chat.ID, threadID)
	s.Require().NoError(err)
	s.Require().Equal(olderUnreadReply.Clock, thread.ReadMessagesAtClockValue)
	s.Require().Equal(uint(0), thread.UnviewedMessagesCount)
	s.Require().Equal(uint(0), thread.UnviewedMentionsCount)
}

func (s *MessengerThreadsSuite) TestThreadReadWatermarkMarksOutOfOrderReplySeen() {
	chat := CreatePublicChat("thread-read-watermark", s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat))

	threadID := "parent-id"
	s.Require().NoError(s.m.persistence.UpsertThread(threadID, chat.ID, threadID, ""))
	_, _, err := s.m.persistence.MarkThreadRead(chat.ID, threadID, 10)
	s.Require().NoError(err)

	senderKey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	contact, err := contacts.BuildContactFromPublicKey(&senderKey.PublicKey)
	s.Require().NoError(err)

	state := s.m.buildMessageState()
	state.CurrentMessageState = &CurrentMessageState{
		MessageID:        "delayed-thread-reply",
		WhisperTimestamp: 10,
		Contact:          contact,
		PublicKey:        &senderKey.PublicKey,
	}

	err = s.m.HandleChatMessage(context.Background(), state, &protobuf.ChatMessage{
		ChatId:      chat.ID,
		Clock:       10,
		Timestamp:   10,
		Text:        "delayed reply",
		MessageType: protobuf.MessageType_PUBLIC_GROUP,
		ContentType: protobuf.ChatMessage_TEXT_PLAIN,
		ThreadId:    &threadID,
	}, nil, false)
	s.Require().NoError(err)
	s.Require().Len(state.Response.Messages(), 1)
	s.Require().True(state.Response.Messages()[0].Seen)
}

func (s *MessengerThreadsSuite) TestSyncThreadReadCreatesPlaceholderWithWatermark() {
	chat := CreatePublicChat("thread-read-sync", s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat))

	threadID := "parent-id"
	state := s.m.buildMessageState()
	err := s.m.HandleSyncThreadMessagesRead(context.Background(), state, &protobuf.SyncThreadMessagesRead{
		ChatId:   chat.ID,
		ThreadId: threadID,
		Clock:    10,
	}, nil)
	s.Require().NoError(err)
	s.Require().Len(state.Response.Threads(), 1)

	thread, err := s.m.persistence.ThreadByID(chat.ID, threadID)
	s.Require().NoError(err)
	s.Require().Equal(uint64(10), thread.ReadMessagesAtClockValue)
	s.Require().Equal("", thread.Name)

	s.Require().NoError(s.m.persistence.UpsertThread(threadID, chat.ID, threadID, "Parent"))
	thread, err = s.m.persistence.ThreadByID(chat.ID, threadID)
	s.Require().NoError(err)
	s.Require().Equal("Parent", thread.Name)
	s.Require().Equal(uint64(10), thread.ReadMessagesAtClockValue)

	state = s.m.buildMessageState()
	err = s.m.HandleSyncThreadMessagesRead(context.Background(), state, &protobuf.SyncThreadMessagesRead{
		ChatId:   chat.ID,
		ThreadId: threadID,
		Clock:    9,
	}, nil)
	s.Require().NoError(err)
	s.Require().Empty(state.Response.Threads())

	thread, err = s.m.persistence.ThreadByID(chat.ID, threadID)
	s.Require().NoError(err)
	s.Require().Equal(uint64(10), thread.ReadMessagesAtClockValue)
}

func (s *MessengerThreadsSuite) TestCreateThreadValidatesEmptyParams() {
	chat := CreateOneToOneChat("test-user", &s.m.identity.PublicKey, s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat))

	// Missing chatID
	_, err := s.m.CreateThread("", "parent-id")
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "chatID and parentMessageID are required")

	// Missing parentMessageID
	_, err = s.m.CreateThread(chat.ID, "")
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "chatID and parentMessageID are required")
}

func (s *MessengerThreadsSuite) TestCreateThreadWithEmptyParentNameBackfills() {
	chat := CreateOneToOneChat("test-user", &s.m.identity.PublicKey, s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat))

	// Create thread for non-existent parent first (would fail in CreateThread API)
	// But test that persistence layer handles empty names correctly
	err := s.m.persistence.UpsertThread("future-parent", chat.ID, "future-parent", "")
	s.Require().NoError(err)

	// Retrieve thread
	thread, err := s.m.persistence.ThreadByID(chat.ID, "future-parent")
	s.Require().NoError(err)
	s.Require().Equal("", thread.Name)

	// Now save the parent message
	parentMsg := buildTestMessage(*chat)
	parentMsg.ID = "future-parent"
	parentMsg.Text = "Parent message arrives"
	parentMsg.ChatMessage.Text = "Parent message arrives"
	s.Require().NoError(s.m.SaveMessages([]*common.Message{parentMsg}))

	// Thread name should be updated
	thread, err = s.m.persistence.ThreadByID(chat.ID, "future-parent")
	s.Require().NoError(err)
	s.Require().Equal("Parent message arrives", thread.Name)
}

func (s *MessengerThreadsSuite) TestThreadMessagesAreReceivedAndListedWhenThreadsDisabled() {
	receiver := s.m
	receiver.featureFlags.Threads = false

	sender := s.newMessenger()
	sender.featureFlags.Threads = true

	receiverChat := CreateOneToOneChat("threads-disabled-one-to-one-chat", &sender.identity.PublicKey, receiver.getTimesource())
	s.Require().NoError(receiver.SaveChat(receiverChat))
	_, err := receiver.Join(receiverChat)
	s.Require().NoError(err)

	senderChat := CreateOneToOneChat("threads-disabled-one-to-one-chat", &receiver.identity.PublicKey, sender.getTimesource())
	s.Require().NoError(sender.SaveChat(senderChat))
	_, err = sender.Join(senderChat)
	s.Require().NoError(err)

	parentMsg := buildTestMessage(*senderChat)
	parentMsg.Text = "Parent message"
	parentMsg.ChatMessage.Text = "Parent message"

	parentResponse, err := sender.SendChatMessage(context.Background(), parentMsg)
	s.Require().NoError(err)
	s.Require().Len(parentResponse.Messages(), 1)
	parentID := parentResponse.Messages()[0].ID

	_, err = WaitOnMessengerResponse(receiver, func(response *MessengerResponse) bool {
		for _, msg := range response.Messages() {
			if msg.ID == parentID {
				return true
			}
		}

		return false
	}, "parent message not received")
	s.Require().NoError(err)

	_, err = sender.CreateThread(senderChat.ID, parentID)
	s.Require().NoError(err)

	threadID := parentID
	threadReply := buildTestMessage(*senderChat)
	threadReply.Text = "Reply from thread"
	threadReply.ChatMessage.Text = "Reply from thread"
	threadReply.ChatMessage.ThreadId = &threadID
	threadReply.ResponseTo = parentID

	threadResponse, err := sender.SendChatMessage(context.Background(), threadReply)
	s.Require().NoError(err)

	var threadReplyID string
	for _, msg := range threadResponse.Messages() {
		if msg.Text == "Reply from thread" && msg.GetThreadId() == parentID {
			threadReplyID = msg.ID
			break
		}
	}
	s.Require().NotEmpty(threadReplyID)

	_, err = WaitOnMessengerResponse(receiver, func(response *MessengerResponse) bool {
		for _, msg := range response.Messages() {
			if msg.ID == threadReplyID {
				s.Require().Equal(parentID, msg.GetThreadId())
				return true
			}
		}

		return false
	}, "thread reply not received")
	s.Require().NoError(err)

	messages, cursor, err := receiver.MessageByChatID(receiverChat.ID, "", "", 10)
	s.Require().NoError(err)
	s.Require().Empty(cursor)

	var foundThreadReply bool
	for _, msg := range messages {
		if msg.ID == threadReplyID {
			foundThreadReply = true
			s.Require().Equal(parentID, msg.GetThreadId())
			s.Require().Equal("Reply from thread", msg.Text)
		}
	}

	s.Require().True(foundThreadReply)
}
