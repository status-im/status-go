package protocol

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/protocol/protobuf"
)

// Exercise receipt and reload using two independent test identities.
func (s *MessengerActivityCenterMessageSuite) TestThreadMentionAndReplyNotifications() {
	for _, scope := range []string{"group", "community"} {
		s.Run(scope, func() {
			receiver := s.m
			sender := s.newMessenger(bobPassword, []string{bobAddress})
			receiver.featureFlags.Threads = true
			sender.featureFlags.Threads = true

			var chat *Chat
			if scope == "community" {
				community, channel := s.createCommunity(sender)
				chat = channel
				s.advertiseCommunityTo(community, sender, receiver)
				s.joinCommunity(community, sender, receiver)
			} else {
				s.Require().NoError(makeMutualContact(sender, receiver.IdentityPublicKey()))
				s.Require().NoError(makeMutualContact(receiver, sender.IdentityPublicKey()))
				response, err := sender.CreateGroupChatWithMembers(context.Background(), "Activity thread test", []string{receiver.myHexIdentity()})
				s.Require().NoError(err)
				chat = response.Chats()[0]
				_, err = WaitOnMessengerResponse(receiver, func(r *MessengerResponse) bool {
					for _, c := range r.Chats() {
						if c.ID == chat.ID && c.Active {
							return true
						}
					}
					return false
				}, "group invitation not received")
				s.Require().NoError(err)
			}

			root := buildTestMessage(*chat)
			root.Text = "Release planning"
			response, err := receiver.SendChatMessage(context.Background(), root)
			s.Require().NoError(err)
			rootID := response.Messages()[0].ID
			_, err = WaitOnMessengerResponse(sender, func(r *MessengerResponse) bool {
				for _, message := range r.Messages() {
					if message.ID == rootID {
						return true
					}
				}
				return false
			}, "thread root not received")
			s.Require().NoError(err)
			_, err = sender.CreateThread(chat.ID, rootID)
			s.Require().NoError(err)

			for _, kind := range []ActivityCenterType{ActivityCenterNotificationTypeMention, ActivityCenterNotificationTypeReply} {
				message := buildTestMessage(*chat)
				message.Text = "Thread reply"
				message.ThreadId = &rootID
				message.ResponseTo = rootID
				if kind == ActivityCenterNotificationTypeMention {
					message.Text = "Hello @" + receiver.myHexIdentity()
				}
				_, err = sender.SendChatMessage(context.Background(), message)
				s.Require().NoError(err)
				received, err := WaitOnMessengerResponse(receiver, func(r *MessengerResponse) bool {
					for _, notification := range r.ActivityCenterNotifications() {
						if notification.Message != nil && notification.Message.ID == message.ID {
							return true
						}
					}
					return false
				}, "thread notification not received")
				s.Require().NoError(err)
				var notification *ActivityCenterNotification
				for _, n := range received.ActivityCenterNotifications() {
					if n.Message != nil && n.Message.ID == message.ID {
						notification = n
					}
				}
				s.Require().NotNil(notification)
				s.Require().Equal(kind, notification.Type)
				s.Require().Equal(rootID, notification.Message.GetThreadId())
				s.Require().Equal("Release planning", notification.ThreadName)

				individual, err := receiver.ActivityCenterNotification(notification.ID)
				s.Require().NoError(err)
				s.Require().NotNil(individual)
				s.Require().Equal(rootID, individual.Message.GetThreadId())
				s.Require().Equal(notification.ThreadName, individual.ThreadName)

				page, err := receiver.ActivityCenterNotifications(ActivityCenterNotificationsRequest{Limit: 100, ReadType: ActivityCenterQueryParamsReadAll})
				s.Require().NoError(err)
				var reloaded *ActivityCenterNotification
				for _, n := range page.Notifications {
					if n.ID.String() == notification.ID.String() {
						reloaded = n
					}
				}
				s.Require().NotNil(reloaded)
				s.Require().Equal(notification.ThreadName, reloaded.ThreadName)
				ids := []types.HexBytes{notification.ID}
				read, err := receiver.MarkActivityCenterNotificationsRead(context.Background(), ids, 0, false)
				s.Require().NoError(err)
				s.Require().Len(read.ActivityCenterNotifications(), 1)
				s.Require().Equal(notification.ThreadName, read.ActivityCenterNotifications()[0].ThreadName)
				unread, err := receiver.MarkActivityCenterNotificationsUnread(context.Background(), ids, receiver.GetCurrentTimeInMillis()+1, false)
				s.Require().NoError(err)
				s.Require().Len(unread.ActivityCenterNotifications(), 1)
				s.Require().Equal(notification.ThreadName, unread.ActivityCenterNotifications()[0].ThreadName)

				readFromChat, err := receiver.MarkMessagesRead(chat.ID, []string{message.ID})
				s.Require().NoError(err)
				s.Require().Len(readFromChat.ActivityCenterNotifications(), 1)
				s.Require().True(readFromChat.ActivityCenterNotifications()[0].Read)
				s.Require().Equal(notification.ThreadName, readFromChat.ActivityCenterNotifications()[0].ThreadName)

				unreadFromChat, err := receiver.MarkMessageAsUnread(chat.ID, message.ID)
				s.Require().NoError(err)
				var unreadNotification *ActivityCenterNotification
				for _, n := range unreadFromChat.ActivityCenterNotifications() {
					if n.ID.String() == notification.ID.String() {
						unreadNotification = n
					}
				}
				s.Require().NotNil(unreadNotification)
				s.Require().False(unreadNotification.Read)
				s.Require().Equal(notification.ThreadName, unreadNotification.ThreadName)

				_, _, seenFromChat, err := receiver.MarkMessagesSeen(chat.ID, []string{message.ID})
				s.Require().NoError(err)
				s.Require().Len(seenFromChat, 1)
				s.Require().True(seenFromChat[0].Read)
				s.Require().Equal(notification.ThreadName, seenFromChat[0].ThreadName)
			}
		})
	}
}

// Verify thread name resolution respects notification type, chat scope, and the feature flag.
func TestActivityCenterThreadNames(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	p := newSQLitePersistence(db)
	require.NoError(t, p.SaveChat(Chat{ID: testPK, ChatType: ChatTypeOneToOne}))
	require.NoError(t, p.SaveMessages([]*common.Message{{ID: "parent", LocalChatID: testPK, From: "creator", ChatMessage: &protobuf.ChatMessage{Text: "Release planning", Clock: 1}}}))
	require.NoError(t, p.UpsertThread("parent", testPK, "parent", "Release planning"))
	m := &Messenger{persistence: p, logger: zap.NewNop()}
	m.featureFlags.Threads = true
	threadID := "parent"
	message := &common.Message{ChatMessage: &protobuf.ChatMessage{ThreadId: &threadID}}
	mention := &ActivityCenterNotification{ChatID: testPK, Type: ActivityCenterNotificationTypeMention, Message: message}
	reply := &ActivityCenterNotification{ChatID: testPK, Type: ActivityCenterNotificationTypeReply, Message: message}
	unrelated := &ActivityCenterNotification{ChatID: testPK, Type: ActivityCenterNotificationTypeContactRequest, Message: message}
	otherChat := &ActivityCenterNotification{ChatID: "other-chat", Type: ActivityCenterNotificationTypeMention, Message: message}
	mainChat := &ActivityCenterNotification{ChatID: testPK, Type: ActivityCenterNotificationTypeMention,
		Message: &common.Message{ChatMessage: &protobuf.ChatMessage{}}, ReplyMessage: message}
	m.populateActivityCenterThreadNames([]*ActivityCenterNotification{nil, {}, mention, reply, unrelated, otherChat, mainChat})
	require.Equal(t, "Release planning", mention.ThreadName)
	require.Equal(t, "Release planning", reply.ThreadName)
	require.Empty(t, unrelated.ThreadName)
	require.Empty(t, otherChat.ThreadName)
	require.Empty(t, mainChat.ThreadName)

	t.Run("missing thread metadata", func(t *testing.T) {
		missingThreadID := "missing-thread"
		notification := &ActivityCenterNotification{
			ChatID: testPK,
			Type:   ActivityCenterNotificationTypeMention,
			Message: &common.Message{ChatMessage: &protobuf.ChatMessage{
				ThreadId: &missingThreadID,
			}},
		}
		m.populateActivityCenterThreadNames([]*ActivityCenterNotification{notification, reply})
		require.Empty(t, notification.ThreadName)
		require.Equal(t, missingThreadID, notification.Message.GetThreadId())
		require.Equal(t, "Release planning", reply.ThreadName)
		encoded, err := json.Marshal(notification)
		require.NoError(t, err)
		require.NotContains(t, string(encoded), `"threadName"`)
	})

	t.Run("empty thread name", func(t *testing.T) {
		updated, err := p.UpdateThreadMetadata(testPK, threadID, "", 1)
		require.NoError(t, err)
		require.True(t, updated)
		m.populateActivityCenterThreadNames([]*ActivityCenterNotification{mention, reply})
		require.Empty(t, mention.ThreadName)
		require.Empty(t, reply.ThreadName)
		require.Equal(t, threadID, mention.Message.GetThreadId())
	})

	t.Run("renamed thread", func(t *testing.T) {
		updated, err := p.UpdateThreadMetadata(testPK, threadID, "Updated release plan", 2)
		require.NoError(t, err)
		require.True(t, updated)
		m.populateActivityCenterThreadNames([]*ActivityCenterNotification{mention, reply})
		require.Equal(t, "Updated release plan", mention.ThreadName)
		require.Equal(t, "Updated release plan", reply.ThreadName)
	})

	m.featureFlags.Threads = false
	mention.ThreadName = ""
	m.populateActivityCenterThreadNames([]*ActivityCenterNotification{mention})
	require.Empty(t, mention.ThreadName)
	encoded, err := json.Marshal(mention)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), `"threadName"`)
}
