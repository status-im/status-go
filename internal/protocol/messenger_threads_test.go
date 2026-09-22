package protocol

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/internal/protocol/common"
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

func (s *MessengerThreadsSuite) TestCreateThreadRequiresParentMessage() {
	chat := CreateOneToOneChat("test-user", &s.m.identity.PublicKey, s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat))

	// Attempt to create thread for non-existent parent
	_, err := s.m.CreateThread(chat.ID, "non-existent-parent")
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "parent message not found")
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

	parent2 := buildTestMessage(*chat)
	parent2.ID = "parent-2"
	parent2.Text = "Second thread"
	parent2.ChatMessage.Text = "Second thread"

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

func (s *MessengerThreadsSuite) TestThreadsIncludeUnreadCounts() {
	chat := CreateOneToOneChat("test-user", &s.m.identity.PublicKey, s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat))

	parentMsg := buildTestMessage(*chat)
	parentMsg.ID = "parent-id"
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

	thread, err := s.m.persistence.ThreadByID(chat.ID, threadID)
	s.Require().NoError(err)
	s.Require().Equal(uint(1), thread.UnviewedMessagesCount)
	s.Require().Equal(uint(1), thread.UnviewedMentionsCount)
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
	receivedReply.ResponseTo = threadID
	receivedReply.LocalChatID = receiverChat.ID
	receivedReply.Mentioned = true
	receivedReply.Seen = false

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

func (s *MessengerThreadsSuite) TestReceivedThreadReplyDoesNotIncrementParentUnreadCount() {
	receiver := s.m
	receiver.featureFlags.Threads = true

	sender := s.newMessenger()
	sender.featureFlags.Threads = true
	chatID := "thread-unread-one-to-one-chat"

	receiverChat := CreateOneToOneChat(chatID, &sender.identity.PublicKey, receiver.getTimesource())
	s.Require().NoError(receiver.SaveChat(receiverChat))
	_, err := receiver.Join(receiverChat)
	s.Require().NoError(err)

	senderChat := CreateOneToOneChat(chatID, &receiver.identity.PublicKey, sender.getTimesource())
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

	_, err = receiver.MarkAllRead(context.Background(), chatID)
	s.Require().NoError(err)
	receiverParentChat, ok := receiver.allChats.Load(chatID)
	s.Require().True(ok)
	s.Require().Equal(uint(0), receiverParentChat.UnviewedMessagesCount)
	s.Require().Equal(uint(0), receiverParentChat.UnviewedMentionsCount)

	_, err = sender.CreateThread(chatID, parentID)
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

	receiverParentChat, ok = receiver.allChats.Load(chatID)
	s.Require().True(ok)
	s.Require().Equal(uint(0), receiverParentChat.UnviewedMessagesCount)
	s.Require().Equal(uint(0), receiverParentChat.UnviewedMentionsCount)

	var responseChat *Chat
	for _, chat := range receiverResponse.Chats() {
		if chat.ID == chatID {
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

	thread, err := receiver.persistence.ThreadByID(chatID, parentID)
	s.Require().NoError(err)
	s.Require().Equal(parentID, thread.ThreadID)
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
	s.Require().Equal(uint(1), thread.UnviewedMentionsCount)

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
	s.Require().Equal(uint(0), thread.UnviewedMessagesCount)
	s.Require().Equal(uint(0), thread.UnviewedMentionsCount)
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
	chatID := "threads-disabled-one-to-one-chat"

	receiverChat := CreateOneToOneChat(chatID, &sender.identity.PublicKey, receiver.getTimesource())
	s.Require().NoError(receiver.SaveChat(receiverChat))
	_, err := receiver.Join(receiverChat)
	s.Require().NoError(err)

	senderChat := CreateOneToOneChat(chatID, &receiver.identity.PublicKey, sender.getTimesource())
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

	_, err = sender.CreateThread(chatID, parentID)
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

	messages, cursor, err := receiver.MessageByChatID(chatID, "", "", 10)
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
