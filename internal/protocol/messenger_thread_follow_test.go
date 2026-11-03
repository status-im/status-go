package protocol

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/protocol/contacts"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	"github.com/status-im/status-go/internal/protocol/requests"
)

func TestThreadFollowPersistence(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	defer db.Close()
	p := newSQLitePersistence(db)

	const ownID = "0xown"
	followed := func(threadID string) *Thread {
		thread, err := p.ThreadByID(testPublicChatID, threadID)
		require.NoError(t, err)
		return thread
	}

	require.NoError(t, p.UpsertThread("thread", testPublicChatID, "thread", "Thread"))
	require.False(t, followed("thread").Followed)

	// Explicit choices are ordered by clock.
	updated, err := p.SetThreadFollowed(testPublicChatID, "thread", true, 10)
	require.NoError(t, err)
	require.True(t, updated)
	updated, err = p.SetThreadFollowed(testPublicChatID, "thread", false, 9)
	require.NoError(t, err)
	require.False(t, updated)
	require.True(t, followed("thread").Followed)
	updated, err = p.SetThreadFollowed(testPublicChatID, "thread", false, 11)
	require.NoError(t, err)
	require.True(t, updated)
	require.False(t, followed("thread").Followed)
	require.EqualValues(t, 11, followed("thread").FollowedClock)

	// Automatic follows never override an explicit choice.
	require.NoError(t, p.SetThreadCreator("thread", testPublicChatID, ownID))
	updated, err = p.AutoFollowOwnedThread(testPublicChatID, "thread", ownID)
	require.NoError(t, err)
	require.False(t, updated)
	require.False(t, followed("thread").Followed)

	// Threads created by the user are auto-followed.
	require.NoError(t, p.UpsertThread("created", testPublicChatID, "created", "Created"))
	require.NoError(t, p.SetThreadCreator("created", testPublicChatID, ownID))
	updated, err = p.AutoFollowOwnedThread(testPublicChatID, "created", ownID)
	require.NoError(t, err)
	require.True(t, updated)
	require.True(t, followed("created").Followed)
	require.Zero(t, followed("created").FollowedClock)

	// Threads started on the user's messages are auto-followed, others are not.
	ownParent := &common.Message{ID: "own-parent", LocalChatID: testPublicChatID, From: ownID, ChatMessage: &protobuf.ChatMessage{Text: "own", Clock: 1}}
	otherParent := &common.Message{ID: "other-parent", LocalChatID: testPublicChatID, From: "0xother", ChatMessage: &protobuf.ChatMessage{Text: "other", Clock: 2}}
	require.NoError(t, p.SaveMessages([]*common.Message{ownParent, otherParent}))
	require.NoError(t, p.UpsertThread(ownParent.ID, testPublicChatID, ownParent.ID, "own"))
	require.NoError(t, p.SetThreadCreator(ownParent.ID, testPublicChatID, "0xother"))
	require.NoError(t, p.UpsertThread(otherParent.ID, testPublicChatID, otherParent.ID, "other"))
	require.NoError(t, p.SetThreadCreator(otherParent.ID, testPublicChatID, "0xother"))
	updated, err = p.AutoFollowOwnedThread(testPublicChatID, ownParent.ID, ownID)
	require.NoError(t, err)
	require.True(t, updated)
	updated, err = p.AutoFollowOwnedThread(testPublicChatID, otherParent.ID, ownID)
	require.NoError(t, err)
	require.False(t, updated)
	require.False(t, followed(otherParent.ID).Followed)

	// Deleted threads are not auto-followed.
	require.NoError(t, p.UpsertThread("deleted", testPublicChatID, "deleted", "Deleted"))
	require.NoError(t, p.SetThreadCreator("deleted", testPublicChatID, ownID))
	_, err = p.SoftDeleteThread(testPublicChatID, "deleted", 5, ownID, 5)
	require.NoError(t, err)
	updated, err = p.AutoFollowOwnedThread(testPublicChatID, "deleted", ownID)
	require.NoError(t, err)
	require.False(t, updated)
}

func TestBackedUpThreadsRestoreFollowState(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	defer db.Close()
	p := newSQLitePersistence(db)

	threads := []*protobuf.BackedUpThread{
		{ThreadId: "auto", ChatId: testPublicChatID, ParentMessageId: "auto", Name: "Auto", LastActivityAt: 1000, Followed: true},
		{ThreadId: "explicit", ChatId: testPublicChatID, ParentMessageId: "explicit", Name: "Explicit", LastActivityAt: 1000, Followed: true, FollowedClock: 10},
		{ThreadId: "unfollowed", ChatId: testPublicChatID, ParentMessageId: "unfollowed", Name: "Unfollowed", LastActivityAt: 1000, FollowedClock: 20},
	}
	require.NoError(t, p.SaveBackedUpThreads(threads))
	exported, err := p.AllThreadsForBackup()
	require.NoError(t, err)
	require.Equal(t, threads, exported)

	// Older explicit choices and automatic follows do not override newer choices.
	require.NoError(t, p.SaveBackedUpThreads([]*protobuf.BackedUpThread{
		{ThreadId: "explicit", ChatId: testPublicChatID, ParentMessageId: "explicit", Followed: false, FollowedClock: 9},
		{ThreadId: "unfollowed", ChatId: testPublicChatID, ParentMessageId: "unfollowed", Followed: true},
	}))
	thread, err := p.ThreadByID(testPublicChatID, "explicit")
	require.NoError(t, err)
	require.True(t, thread.Followed)
	thread, err = p.ThreadByID(testPublicChatID, "unfollowed")
	require.NoError(t, err)
	require.False(t, thread.Followed)

	// A newer explicit choice wins.
	require.NoError(t, p.SaveBackedUpThreads([]*protobuf.BackedUpThread{
		{ThreadId: "auto", ChatId: testPublicChatID, ParentMessageId: "auto", Followed: false, FollowedClock: 5},
	}))
	thread, err = p.ThreadByID(testPublicChatID, "auto")
	require.NoError(t, err)
	require.False(t, thread.Followed)
	require.EqualValues(t, 5, thread.FollowedClock)
}

func (s *MessengerThreadsSuite) TestFollowAndUnfollowThread() {
	chat := CreateOneToOneChat("thread-follow", &s.m.identity.PublicKey, s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat))

	parent := buildTestMessage(*chat)
	parent.ID = "follow-parent"
	parent.From = "0xother"
	s.Require().NoError(s.m.SaveMessages([]*common.Message{parent}))

	response, err := s.m.CreateThread(chat.ID, parent.ID)
	s.Require().NoError(err)
	s.Require().Len(response.Threads(), 1)
	s.Require().True(response.Threads()[0].Followed, "creators follow their threads")

	response, err = s.m.UnfollowThread(context.Background(), &requests.UnfollowThread{ChatID: chat.ID, ThreadID: parent.ID})
	s.Require().NoError(err)
	s.Require().Len(response.Threads(), 1)
	s.Require().False(response.Threads()[0].Followed)
	threads, err := s.m.ThreadsByChatID(chat.ID)
	s.Require().NoError(err)
	s.Require().Len(threads, 1)
	s.Require().False(threads[0].Followed)

	// Archived threads can be followed.
	_, err = s.m.persistence.db.Exec(`UPDATE threads SET last_activity_at = 1 WHERE chat_id = ? AND thread_id = ?`, chat.ID, parent.ID)
	s.Require().NoError(err)
	response, err = s.m.FollowThread(context.Background(), &requests.FollowThread{ChatID: chat.ID, ThreadID: parent.ID})
	s.Require().NoError(err)
	s.Require().Len(response.Threads(), 1)
	s.Require().True(response.Threads()[0].Followed)
	s.Require().True(response.Threads()[0].Archived)

	// A newer choice from a paired device must not make local actions stale.
	_, err = s.m.persistence.SetThreadFollowed(chat.ID, parent.ID, true, 1<<62)
	s.Require().NoError(err)
	response, err = s.m.UnfollowThread(context.Background(), &requests.UnfollowThread{ChatID: chat.ID, ThreadID: parent.ID})
	s.Require().NoError(err)
	s.Require().False(response.Threads()[0].Followed)

	_, err = s.m.FollowThread(context.Background(), &requests.FollowThread{ChatID: chat.ID})
	s.Require().ErrorIs(err, requests.ErrFollowThreadInvalidRequest)
	_, err = s.m.FollowThread(context.Background(), &requests.FollowThread{ChatID: chat.ID, ThreadID: "missing"})
	s.Require().ErrorIs(err, common.ErrRecordNotFound)
	_, err = s.m.FollowThread(context.Background(), &requests.FollowThread{ChatID: "missing", ThreadID: parent.ID})
	s.Require().ErrorIs(err, ErrChatNotFoundError)

	_, err = s.m.persistence.SoftDeleteThread(chat.ID, parent.ID, 1<<62, s.m.myHexIdentity(), 1)
	s.Require().NoError(err)
	_, err = s.m.FollowThread(context.Background(), &requests.FollowThread{ChatID: chat.ID, ThreadID: parent.ID})
	s.Require().ErrorIs(err, ErrThreadDeleted)

	s.m.featureFlags.Threads = false
	_, err = s.m.FollowThread(context.Background(), &requests.FollowThread{ChatID: chat.ID, ThreadID: parent.ID})
	s.Require().ErrorIs(err, ErrThreadFeatureDisabled)
}

func (s *MessengerThreadsSuite) TestSendingThreadReplyFollowsThread() {
	chat := s.createJoinedOneToOneThreadChat()

	parent := buildTestMessage(*chat)
	parent.ID = "reply-follow-parent"
	parent.From = chat.ID
	s.Require().NoError(s.m.SaveMessages([]*common.Message{parent}))
	s.Require().NoError(s.m.persistence.UpsertThread(parent.ID, chat.ID, parent.ID, "Parent"))
	s.Require().NoError(s.m.persistence.SetThreadCreator(parent.ID, chat.ID, chat.ID))

	_, err := s.m.UnfollowThread(context.Background(), &requests.UnfollowThread{ChatID: chat.ID, ThreadID: parent.ID})
	s.Require().NoError(err)

	threadID := parent.ID
	reply := buildTestMessage(*chat)
	reply.ChatMessage.ThreadId = &threadID
	reply.ResponseTo = threadID
	response, err := s.m.SendChatMessage(context.Background(), reply)
	s.Require().NoError(err)
	s.Require().Len(response.Threads(), 1)
	s.Require().True(response.Threads()[0].Followed)
}

func (s *MessengerThreadsSuite) TestSendingThreadReplyAfterSyncedOrRestoredUnfollow() {
	chat := s.createJoinedOneToOneThreadChat()
	parent := buildTestMessage(*chat)
	parent.ID = "future-unfollow-parent"
	parent.From = chat.ID
	s.Require().NoError(s.m.SaveMessages([]*common.Message{parent}))
	s.Require().NoError(s.m.persistence.UpsertThread(parent.ID, chat.ID, parent.ID, "Parent"))

	for _, source := range []string{"sync", "backup"} {
		s.Run(source, func() {
			clock, _ := chat.NextClockAndTimestamp(s.m.getTimesource())
			followedClock := clock + 1000
			switch source {
			case "sync":
				s.Require().NoError(s.m.HandleSyncThreadFollow(context.Background(), s.m.buildMessageState(),
					&protobuf.SyncThreadFollow{
						ChatId: chat.ID, ThreadId: parent.ID, Followed: false, Clock: followedClock,
					}, nil))
			case "backup":
				s.Require().NoError(s.m.persistence.SaveBackedUpThreads([]*protobuf.BackedUpThread{{
					ChatId: chat.ID, ThreadId: parent.ID, ParentMessageId: parent.ID,
					Followed: false, FollowedClock: followedClock,
				}}))
			}
			s.Require().Less(chat.LastClockValue, followedClock)
			thread, err := s.m.persistence.ThreadByID(chat.ID, parent.ID)
			s.Require().NoError(err)
			s.Require().False(thread.Followed)

			reply := buildTestMessage(*chat)
			reply.ThreadId = &parent.ID
			reply.ResponseTo = parent.ID
			response, err := s.m.SendChatMessage(context.Background(), reply)
			s.Require().NoError(err)
			var sentReply *common.Message
			for _, message := range response.Messages() {
				if message.ID == reply.ID {
					sentReply = message
				}
			}
			s.Require().NotNil(sentReply)
			s.Require().Greater(sentReply.Clock, followedClock)
			s.Require().Len(response.Threads(), 1)
			s.Require().True(response.Threads()[0].Followed)
			s.Require().Equal(sentReply.Clock, response.Threads()[0].FollowedClock)
		})
	}
}

func (s *MessengerThreadsSuite) TestDelayedParentRetriesThreadAutoFollow() {
	creator, err := crypto.GenerateKey()
	s.Require().NoError(err)
	creatorID := contacts.ContactIDFromPublicKey(&creator.PublicKey)
	chat := CreateOneToOneChat("delayed-parent-follow", &creator.PublicKey, s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat))

	for _, test := range []struct {
		name       string
		ownParent  bool
		unfollowed bool
		deleted    bool
	}{
		{name: "own-parent", ownParent: true},
		{name: "other-parent"},
		{name: "explicit-unfollow", ownParent: true, unfollowed: true},
		{name: "deleted-thread", ownParent: true, deleted: true},
	} {
		s.Run(test.name, func() {
			state := &ReceivedMessageState{
				Response: &MessengerResponse{},
				CurrentMessageState: &CurrentMessageState{
					PublicKey: &creator.PublicKey, WhisperTimestamp: 10,
				},
			}
			s.Require().NoError(s.m.HandleThreadMetadata(context.Background(), state, &protobuf.ThreadMetadata{
				Clock: 10, ChatId: chat.ID, ThreadId: test.name, Name: "Thread", CreatorId: creatorID,
				MessageType: protobuf.MessageType_ONE_TO_ONE,
			}, nil))
			thread, err := s.m.persistence.ThreadByID(chat.ID, test.name)
			s.Require().NoError(err)
			s.Require().False(thread.Followed)
			if test.unfollowed {
				_, err = s.m.persistence.SetThreadFollowed(chat.ID, test.name, false, 20)
				s.Require().NoError(err)
			}
			if test.deleted {
				_, err = s.m.persistence.SoftDeleteThread(chat.ID, test.name, 20, creatorID, 20)
				s.Require().NoError(err)
			}

			parent := buildTestMessage(*chat)
			parent.ID = test.name
			parent.From = creatorID
			if test.ownParent {
				parent.From = s.m.myHexIdentity()
			}
			s.Require().NoError(s.m.SaveMessages([]*common.Message{parent}))
			response := &MessengerResponse{}
			s.Require().NoError(s.m.updateThreadFollowsFromMessages([]*common.Message{parent}, response))
			expectedFollowed := test.ownParent && !test.unfollowed && !test.deleted
			thread, err = s.m.persistence.ThreadByID(chat.ID, test.name)
			s.Require().NoError(err)
			s.Require().Equal(expectedFollowed, thread.Followed)
			if expectedFollowed {
				s.Require().Len(response.Threads(), 1)
				s.Require().Equal(test.name, response.Threads()[0].ThreadID)
				s.Require().True(response.Threads()[0].Followed)
				s.Require().Zero(response.Threads()[0].FollowedClock)
			} else {
				s.Require().Empty(response.Threads())
			}
			if test.unfollowed {
				s.Require().EqualValues(20, thread.FollowedClock)
			}
		})
	}
}

func (s *MessengerThreadsSuite) TestThreadOnOwnMessageIsFollowedOnlyByParentAuthor() {
	receiver := s.m
	sender := s.newMessenger()
	sender.featureFlags.Threads = true

	chatID := "thread-follow-one-to-one"
	receiverChat := CreateOneToOneChat(chatID, &sender.identity.PublicKey, receiver.getTimesource())
	s.Require().NoError(receiver.SaveChat(receiverChat))
	_, err := receiver.Join(receiverChat)
	s.Require().NoError(err)
	senderChat := CreateOneToOneChat(chatID, &receiver.identity.PublicKey, sender.getTimesource())
	s.Require().NoError(sender.SaveChat(senderChat))
	_, err = sender.Join(senderChat)
	s.Require().NoError(err)

	sendAndWait := func(from *Messenger, to *Messenger, chat *Chat, text string, threadID string) string {
		message := buildTestMessage(*chat)
		message.Text = text
		message.ChatMessage.Text = text
		if threadID != "" {
			message.ChatMessage.ThreadId = &threadID
			message.ResponseTo = threadID
		}
		response, err := from.SendChatMessage(context.Background(), message)
		s.Require().NoError(err)
		id := response.Messages()[0].ID
		_, err = WaitOnMessengerResponse(to, func(r *MessengerResponse) bool {
			for _, m := range r.Messages() {
				if m.ID == id {
					return true
				}
			}
			return false
		}, "message not received")
		s.Require().NoError(err)
		return id
	}
	followedBy := func(m *Messenger, chatID, threadID string) bool {
		thread, err := m.persistence.ThreadByID(chatID, threadID)
		s.Require().NoError(err)
		return thread.Followed
	}

	// The sender starts a thread on the receiver's message: both follow it.
	parentID := sendAndWait(receiver, sender, receiverChat, "receiver parent", "")
	_, err = sender.CreateThread(senderChat.ID, parentID)
	s.Require().NoError(err)
	_, err = WaitOnMessengerResponse(receiver, func(r *MessengerResponse) bool {
		for _, thread := range r.Threads() {
			if thread.ThreadID == parentID && thread.Followed {
				return true
			}
		}
		return false
	}, "receiver did not follow thread started on its message")
	s.Require().NoError(err)
	s.Require().True(followedBy(sender, senderChat.ID, parentID))

	// The sender starts a thread on its own message: the receiver does not follow it.
	otherParentID := sendAndWait(sender, receiver, senderChat, "sender parent", "")
	_, err = sender.CreateThread(senderChat.ID, otherParentID)
	s.Require().NoError(err)
	sendAndWait(sender, receiver, senderChat, "reply in sender thread", otherParentID)
	s.Require().False(followedBy(receiver, receiverChat.ID, otherParentID))

	// Replies from others keep an explicit unfollow; the user's own reply follows again.
	_, err = receiver.UnfollowThread(context.Background(), &requests.UnfollowThread{ChatID: receiverChat.ID, ThreadID: parentID})
	s.Require().NoError(err)
	sendAndWait(sender, receiver, senderChat, "reply from sender", parentID)
	s.Require().False(followedBy(receiver, receiverChat.ID, parentID))
	sendAndWait(receiver, sender, receiverChat, "reply from receiver", otherParentID)
	s.Require().True(followedBy(receiver, receiverChat.ID, otherParentID))

	// Following never changes the parent chat's unread counts.
	_, err = receiver.MarkAllRead(context.Background(), receiverChat.ID)
	s.Require().NoError(err)
	sendAndWait(sender, receiver, senderChat, "another reply", otherParentID)
	chat, ok := receiver.allChats.Load(receiverChat.ID)
	s.Require().True(ok)
	s.Require().Zero(chat.UnviewedMessagesCount)
	s.Require().Zero(chat.UnviewedMentionsCount)
}

func (s *MessengerThreadsSuite) TestSyncThreadFollow() {
	chat := CreatePublicChat("thread-follow-sync", s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat))
	threadID := "synced-thread"

	handle := func(followed bool, clock uint64) *MessengerResponse {
		state := s.m.buildMessageState()
		err := s.m.HandleSyncThreadFollow(context.Background(), state, &protobuf.SyncThreadFollow{
			ChatId: chat.ID, ThreadId: threadID, Followed: followed, Clock: clock,
		}, nil)
		s.Require().NoError(err)
		return state.Response
	}

	// The choice is kept on a placeholder until the thread arrives.
	response := handle(false, 10)
	s.Require().Len(response.Threads(), 1)
	s.Require().False(response.Threads()[0].Followed)

	// Creation metadata of the user's own thread must not override the synced unfollow.
	s.Require().NoError(s.m.persistence.UpsertThread(threadID, chat.ID, threadID, "Synced"))
	s.Require().NoError(s.m.persistence.SetThreadCreator(threadID, chat.ID, s.m.myHexIdentity()))
	followed, err := s.m.persistence.AutoFollowOwnedThread(chat.ID, threadID, s.m.myHexIdentity())
	s.Require().NoError(err)
	s.Require().False(followed)

	s.Require().Empty(handle(true, 9).Threads(), "stale sync is ignored")
	response = handle(true, 11)
	s.Require().Len(response.Threads(), 1)
	s.Require().True(response.Threads()[0].Followed)

	err = s.m.HandleSyncThreadFollow(context.Background(), s.m.buildMessageState(), &protobuf.SyncThreadFollow{
		ChatId: "missing", ThreadId: threadID, Followed: true, Clock: 12,
	}, nil)
	s.Require().ErrorIs(err, ErrChatNotFound)
}

func (s *MessengerThreadsSuite) TestThreadCreationMetadataAutoFollowsThreadOnOwnMessage() {
	creator, err := crypto.GenerateKey()
	s.Require().NoError(err)
	creatorID := contacts.ContactIDFromPublicKey(&creator.PublicKey)
	chat := CreateOneToOneChat("metadata-follow", &creator.PublicKey, s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat))

	ownParent := buildTestMessage(*chat)
	ownParent.ID = "own-parent"
	ownParent.From = s.m.myHexIdentity()
	otherParent := buildTestMessage(*chat)
	otherParent.ID = "other-parent"
	otherParent.From = creatorID
	s.Require().NoError(s.m.SaveMessages([]*common.Message{ownParent, otherParent}))

	handle := func(threadID string) *MessengerResponse {
		state := &ReceivedMessageState{
			Response: &MessengerResponse{},
			CurrentMessageState: &CurrentMessageState{
				PublicKey:        &creator.PublicKey,
				WhisperTimestamp: 10,
			},
		}
		err := s.m.HandleThreadMetadata(context.Background(), state, &protobuf.ThreadMetadata{
			Clock: 10, ChatId: chat.ID, ThreadId: threadID, Name: "Thread", CreatorId: creatorID,
			MessageType: protobuf.MessageType_ONE_TO_ONE,
		}, nil)
		s.Require().NoError(err)
		return state.Response
	}

	response := handle(ownParent.ID)
	s.Require().Len(response.Threads(), 1)
	s.Require().True(response.Threads()[0].Followed)

	response = handle(otherParent.ID)
	s.Require().Len(response.Threads(), 1)
	s.Require().False(response.Threads()[0].Followed)
}
