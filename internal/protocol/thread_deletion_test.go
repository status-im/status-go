package protocol

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/protocol/contacts"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	"github.com/status-im/status-go/internal/protocol/requests"
)

func TestThreadDeletionPersistsAndCannotResurrect(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	defer db.Close()
	p := newSQLitePersistence(db)
	chat := CreatePublicChat("deletion-test", &testTimeSource{})
	require.NoError(t, p.SaveChats([]*Chat{chat}))
	threadID := "root"
	root := buildTestMessage(*chat)
	root.ID = threadID
	reply := buildTestMessage(*chat)
	reply.ID = "reply"
	reply.ThreadId = &threadID
	require.NoError(t, p.SaveMessages([]*common.Message{root, reply}))
	require.NoError(t, p.UpsertThread(threadID, chat.ID, root.ID, "Thread"))
	notification := &ActivityCenterNotification{ID: []byte{1}, Type: ActivityCenterNotificationTypeMention, ChatID: chat.ID, Message: reply, Timestamp: 1, UpdatedAt: 1}
	_, err = p.SaveActivityCenterNotification(notification, false)
	require.NoError(t, err)
	// Force a reply update to fail and verify the thread tombstone rolls back too.
	_, err = db.Exec(`CREATE TRIGGER fail_thread_delete BEFORE UPDATE OF deleted ON user_messages BEGIN SELECT RAISE(ABORT, 'forced failure'); END`)
	require.NoError(t, err)
	require.Error(t, p.SoftDeleteThread(chat.ID, threadID, 20, "admin", 18))
	before, err := p.ThreadByID(chat.ID, threadID)
	require.NoError(t, err)
	require.False(t, before.Deleted)
	_, err = db.Exec(`DROP TRIGGER fail_thread_delete`)
	require.NoError(t, err)
	require.NoError(t, p.SoftDeleteThread(chat.ID, threadID, 20, "admin", 18))
	storedNotification, err := p.GetActivityCenterNotificationByID(notification.ID)
	require.NoError(t, err)
	require.True(t, storedNotification.Deleted)
	require.True(t, storedNotification.Read)
	require.NoError(t, p.SoftDeleteThread(chat.ID, threadID, 10, "older-admin"))
	storedRoot, err := p.MessageByID(root.ID)
	require.NoError(t, err)
	require.False(t, storedRoot.Deleted)
	storedReply, err := p.MessageByID(reply.ID)
	require.NoError(t, err)
	require.True(t, storedReply.Deleted)
	require.Equal(t, "admin", storedReply.DeletedBy)
	// A restore or edit may rewrite a reply with deleted=false.
	reply.Deleted = false
	require.NoError(t, p.SaveMessages([]*common.Message{reply}))
	late := buildTestMessage(*chat)
	late.ID = "late"
	late.ThreadId = &threadID
	require.NoError(t, p.SaveMessages([]*common.Message{late}))
	require.True(t, late.Deleted)
	// Exercise the database guard independently of SaveMessages.
	_, err = db.Exec(`UPDATE user_messages SET deleted = FALSE WHERE id = ?`, late.ID)
	require.NoError(t, err)
	storedLate, err := p.MessageByID(late.ID)
	require.NoError(t, err)
	require.True(t, storedLate.Deleted)
	thread, err := p.ThreadWithSummaryByID(chat.ID, threadID)
	require.NoError(t, err)
	require.True(t, thread.Deleted)
	require.EqualValues(t, 20, thread.DeletedClock)
	require.EqualValues(t, 18, thread.DeletedAt)
	require.Equal(t, "admin", thread.DeletedBy)
	listed, err := p.ThreadsByChatID(chat.ID)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.True(t, listed[0].Deleted)
	cards, err := p.ThreadsWithSummariesByParentMessageIDs(chat.ID, []string{threadID}, 6)
	require.NoError(t, err)
	require.Len(t, cards, 1)
	require.True(t, cards[0].Deleted)
	require.Zero(t, cards[0].MessagesCount)
	require.Zero(t, thread.MessagesCount)
	require.Zero(t, thread.UnviewedMessagesCount)
	require.Nil(t, thread.LastMessage)
	updated, err := p.UpdateThreadMetadata(chat.ID, threadID, "Resurrected", 100)
	require.NoError(t, err)
	require.False(t, updated)
	backup, err := p.AllThreadsForBackup()
	require.NoError(t, err)
	restoredDB, err := openTestDB()
	require.NoError(t, err)
	defer restoredDB.Close()
	restored := newSQLitePersistence(restoredDB)
	require.NoError(t, restored.SaveBackedUpThreads(backup))
	require.NoError(t, restored.SaveBackedUpThreads([]*protobuf.BackedUpThread{{ThreadId: threadID, ChatId: chat.ID, Name: "Old backup"}}))
	restoredThread, err := restored.ThreadByID(chat.ID, threadID)
	require.NoError(t, err)
	require.True(t, restoredThread.Deleted)
	require.EqualValues(t, 20, restoredThread.DeletedClock)
	require.EqualValues(t, 18, restoredThread.DeletedAt)
	// Tombstones must survive message restoration after thread restoration.
	late.ID = "restored-reply"
	late.Deleted = false
	require.NoError(t, restored.SaveMessages([]*common.Message{late}))
	require.True(t, late.Deleted)
	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM user_messages`).Scan(&count))
	require.Equal(t, 3, count)
}

func (s *MessengerThreadsSuite) TestDeleteThreadKeepsParentAndBlocksWrites() {
	chat := CreateOneToOneChat("self", &s.m.identity.PublicKey, s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat))
	root := buildTestMessage(*chat)
	root.ID = "delete-root"
	s.Require().NoError(s.m.SaveMessages([]*common.Message{root}))
	_, err := s.m.CreateThread(chat.ID, root.ID)
	s.Require().NoError(err)
	reply := buildTestMessage(*chat)
	reply.ID = "delete-reply"
	reply.ThreadId = &root.ID
	s.Require().NoError(s.m.SaveMessages([]*common.Message{reply}))
	response, err := s.m.DeleteThread(context.Background(), &requests.DeleteThread{ChatID: chat.ID, ThreadID: root.ID})
	s.Require().NoError(err)
	s.Require().Len(response.Threads(), 1)
	s.Require().True(response.Threads()[0].Deleted)
	parent, err := s.m.MessageByID(root.ID)
	s.Require().NoError(err)
	s.Require().False(parent.Deleted)
	_, err = s.m.SendChatMessage(context.Background(), reply)
	s.Require().ErrorIs(err, ErrThreadDeleted)
	name := "new"
	_, err = s.m.EditThread(context.Background(), &requests.EditThread{ChatID: chat.ID, ThreadID: root.ID, Name: &name})
	s.Require().ErrorIs(err, ErrThreadDeleted)
	_, err = s.m.CreateThread(chat.ID, root.ID)
	s.Require().ErrorIs(err, ErrThreadDeleted)
	_, err = s.m.DeleteThread(context.Background(), &requests.DeleteThread{ChatID: chat.ID, ThreadID: root.ID})
	s.Require().NoError(err)
}

func (s *MessengerThreadsSuite) TestDeleteThreadRejectsNonCreator() {
	peer, err := crypto.GenerateKey()
	s.Require().NoError(err)
	chat := CreateOneToOneChat("peer", &peer.PublicKey, s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat))
	s.Require().NoError(s.m.persistence.UpsertThread("peer-root", chat.ID, "peer-root", "Thread"))
	s.Require().NoError(s.m.persistence.SetThreadCreator("peer-root", chat.ID, contacts.ContactIDFromPublicKey(&peer.PublicKey)))
	_, err = s.m.DeleteThread(context.Background(), &requests.DeleteThread{ChatID: chat.ID, ThreadID: "peer-root"})
	s.Require().ErrorIs(err, ErrThreadDeletePermission)
}

func (s *MessengerThreadsSuite) TestThreadDeletionBeforeCreationIsAuthenticatedAndTerminal() {
	creator, chat, metadata := s.newThreadMetadataHandler()
	other, err := crypto.GenerateKey()
	s.Require().NoError(err)
	handleDelete := func(signer *ecdsa.PublicKey, clock uint64) error {
		return s.m.HandleDeleteThread(context.Background(), &ReceivedMessageState{
			Response: &MessengerResponse{}, CurrentMessageState: &CurrentMessageState{PublicKey: signer, WhisperTimestamp: clock},
		}, &protobuf.DeleteThread{ChatId: chat.ID, ThreadId: "reordered", Clock: clock, MessageType: protobuf.MessageType_ONE_TO_ONE}, nil)
	}
	s.Require().NoError(handleDelete(&creator.PublicKey, 20))
	pending, err := s.m.persistence.PendingThreadMetadata(chat.ID, "reordered")
	s.Require().NoError(err)
	s.Require().Len(pending, 1)
	s.Require().True(pending[0].Deleted)
	// A subsequent rename from the same signer must not replace a pending delete.
	_, err = metadata(&protobuf.ThreadMetadata{ThreadId: "reordered", Clock: 30, Name: "Later rename"})
	s.Require().NoError(err)
	response, err := metadata(&protobuf.ThreadMetadata{ThreadId: "reordered", Clock: 10, Name: "Created", CreatorId: contacts.ContactIDFromPublicKey(&creator.PublicKey)})
	s.Require().NoError(err)
	s.Require().Len(response.Threads(), 1)
	s.Require().True(response.Threads()[0].Deleted)
	// Retried creation and renames cannot resurrect the tombstone.
	_, err = metadata(&protobuf.ThreadMetadata{ThreadId: "reordered", Clock: 40, Name: "Retry", CreatorId: contacts.ContactIDFromPublicKey(&creator.PublicKey)})
	s.Require().NoError(err)
	_, err = metadata(&protobuf.ThreadMetadata{ThreadId: "reordered", Clock: 50, Name: "Renamed"})
	s.Require().NoError(err)
	thread, err := s.m.persistence.ThreadByID(chat.ID, "reordered")
	s.Require().NoError(err)
	s.Require().True(thread.Deleted)
	// No claimed actor in the payload can substitute for an authenticated signer.
	s.Require().Error(handleDelete(nil, 60))
	// A one-to-one event from another signer maps to that signer's chat, never this thread.
	_ = handleDelete(&other.PublicKey, 60)
	thread, err = s.m.persistence.ThreadByID(chat.ID, "reordered")
	s.Require().NoError(err)
	s.Require().Equal(contacts.ContactIDFromPublicKey(&creator.PublicKey), thread.DeletedBy)
}

func (s *MessengerThreadsSuite) TestThreadDeletionFromNonCreatorIsRejectedOnReceive() {
	creator, chat, metadata := s.newThreadMetadataHandler()
	_, err := metadata(&protobuf.ThreadMetadata{ThreadId: "protected", Clock: 10, Name: "Created", CreatorId: contacts.ContactIDFromPublicKey(&creator.PublicKey)})
	s.Require().NoError(err)
	// A paired installation signed by us is still not the thread creator.
	err = s.m.HandleDeleteThread(context.Background(), &ReceivedMessageState{
		Response: &MessengerResponse{}, CurrentMessageState: &CurrentMessageState{PublicKey: &s.m.identity.PublicKey, WhisperTimestamp: 20},
	}, &protobuf.DeleteThread{ChatId: chat.ID, ThreadId: "protected", Clock: 20, MessageType: protobuf.MessageType_ONE_TO_ONE}, nil)
	s.Require().ErrorIs(err, ErrThreadDeletePermission)
	thread, err := s.m.persistence.ThreadByID(chat.ID, "protected")
	s.Require().NoError(err)
	s.Require().False(thread.Deleted)
}

func (s *MessengerThreadsSuite) TestDeleteThreadAdminCanDeleteAnotherCreatorsThread() {
	peer, err := crypto.GenerateKey()
	s.Require().NoError(err)
	response, err := s.m.CreateGroupChatWithMembers(context.Background(), "delete group", nil)
	s.Require().NoError(err)
	chat := response.Chats()[0]
	s.Require().NoError(s.m.persistence.UpsertThread("member-root", chat.ID, "member-root", "Member thread"))
	s.Require().NoError(s.m.persistence.SetThreadCreator("member-root", chat.ID, contacts.ContactIDFromPublicKey(&peer.PublicKey)))
	response, err = s.m.DeleteThread(context.Background(), &requests.DeleteThread{ChatID: chat.ID, ThreadID: "member-root"})
	s.Require().NoError(err)
	s.Require().True(response.Threads()[0].Deleted)
}

func (s *MessengerThreadsSuite) TestThreadDeletionPropagatesAndLateRepliesStayDeleted() {
	receiver := s.m
	sender := s.newMessenger()
	sender.featureFlags.Threads = true
	receiverChat := CreateOneToOneChat("sender", &sender.identity.PublicKey, receiver.getTimesource())
	s.Require().NoError(receiver.SaveChat(receiverChat))
	_, err := receiver.Join(receiverChat)
	s.Require().NoError(err)
	senderChat := CreateOneToOneChat("receiver", &receiver.identity.PublicKey, sender.getTimesource())
	s.Require().NoError(sender.SaveChat(senderChat))
	_, err = sender.Join(senderChat)
	s.Require().NoError(err)
	reply := buildTestMessage(*senderChat)
	reply.Text = "first thread reply"
	reply.ChatMessage.Text = reply.Text
	created, err := sender.StartThreadFromNewMessage(context.Background(), &requests.StartThreadFromNewMessage{ThreadName: "Delete test", Message: reply})
	s.Require().NoError(err)
	threadID := created.Threads()[0].ThreadID
	_, err = WaitOnMessengerResponse(receiver, func(response *MessengerResponse) bool {
		thread, err := receiver.persistence.ThreadByID(receiverChat.ID, threadID)
		_, replyErr := receiver.persistence.MessageByID(reply.ID)
		return err == nil && thread.CreatorID != "" && replyErr == nil
	}, "thread not received")
	s.Require().NoError(err)

	// Model a peer that has not yet received the deletion (or an older client).
	// Delete only on the receiver; the sender is still allowed to send a reply.
	s.Require().NoError(receiver.persistence.SoftDeleteThread(receiverChat.ID, threadID, 1, sender.selfContact.ID))
	late := buildTestMessage(*senderChat)
	late.Text = "late reply"
	late.ChatMessage.Text = late.Text
	late.ThreadId = &threadID
	_, err = sender.SendChatMessage(context.Background(), late)
	s.Require().NoError(err)
	response, err := WaitOnMessengerResponse(receiver, func(response *MessengerResponse) bool {
		stored, err := receiver.persistence.MessageByID(late.ID)
		return err == nil && stored.Deleted
	}, "late reply was not retained as deleted")
	s.Require().NoError(err)
	for _, message := range response.Messages() {
		s.Require().NotEqual(late.ID, message.ID)
	}
	s.Require().Empty(response.Notifications())

	// Send the authenticated deletion over the actual transport to a peer that
	// already has a tombstone. This must still emit a deleted thread update.
	_, err = sender.DeleteThread(context.Background(), &requests.DeleteThread{ChatID: senderChat.ID, ThreadID: threadID})
	s.Require().NoError(err)
	_, err = WaitOnMessengerResponse(receiver, func(response *MessengerResponse) bool {
		for _, thread := range response.Threads() {
			if thread.ThreadID == threadID && thread.Deleted {
				return true
			}
		}
		return false
	}, "thread deletion not received")
	s.Require().NoError(err)
	root, err := receiver.persistence.MessageByID(threadID)
	s.Require().NoError(err)
	s.Require().False(root.Deleted)
	s.Require().Equal(threadID, senderChat.LastMessage.ID)
	s.Require().Equal(threadID, receiverChat.LastMessage.ID)
	s.Require().False(receiverChat.LastMessage.Deleted)
	storedReply, err := receiver.persistence.MessageByID(reply.ID)
	s.Require().NoError(err)
	s.Require().True(storedReply.Deleted)
}

func (s *MessengerThreadsSuite) TestDeleteThreadAllowsCommunityTokenMaster() {
	response, err := s.m.CreateCommunity(&requests.CreateCommunity{
		Membership: protobuf.CommunityPermissions_AUTO_ACCEPT,
		Name:       "Deletion permissions", Color: "#ffffff", Description: "Thread deletion permissions",
	}, false)
	s.Require().NoError(err)
	community := response.Communities()[0]
	response, err = s.m.CreateCommunityChat(community.ID(), &protobuf.CommunityChat{
		Permissions: &protobuf.CommunityPermissions{Access: protobuf.CommunityPermissions_AUTO_ACCEPT},
		Identity:    &protobuf.ChatIdentity{DisplayName: "Threads"},
	})
	s.Require().NoError(err)
	chat := response.Chats()[0]
	community, err = s.m.communitiesManager.GetByIDString(chat.CommunityID)
	s.Require().NoError(err)
	member, err := crypto.GenerateKey()
	s.Require().NoError(err)
	_, err = community.AddMember(&member.PublicKey, []protobuf.CommunityMember_Roles{protobuf.CommunityMember_ROLE_TOKEN_MASTER}, 0)
	s.Require().NoError(err)
	s.Require().NoError(s.m.communitiesManager.SaveCommunity(community))
	thread := &Thread{ThreadID: "another-members-thread", ChatID: chat.ID, CreatorID: "another-member"}
	allowed, err := s.m.senderCanDeleteThread(chat, thread, &member.PublicKey)
	s.Require().NoError(err)
	s.Require().True(allowed)
	// Ordinary members still cannot delete someone else's thread.
	_, err = community.SetRoleToMember(&member.PublicKey, protobuf.CommunityMember_ROLE_NONE)
	s.Require().NoError(err)
	s.Require().NoError(s.m.communitiesManager.SaveCommunity(community))
	allowed, err = s.m.senderCanDeleteThread(chat, thread, &member.PublicKey)
	s.Require().NoError(err)
	s.Require().False(allowed)
}

func TestThreadDeletionCascadesFromParentInEitherDeliveryOrder(t *testing.T) {
	for _, parentFirst := range []bool{false, true} {
		t.Run(fmt.Sprint("parent-first-", parentFirst), func(t *testing.T) {
			db, err := openTestDB()
			require.NoError(t, err)
			defer db.Close()
			p := newSQLitePersistence(db)
			chat := CreatePublicChat("parent-cascade", &testTimeSource{})
			require.NoError(t, p.SaveChats([]*Chat{chat}))
			parent := buildTestMessage(*chat)
			parent.ID = "root"
			parent.From = "parent-author"
			reply := buildTestMessage(*chat)
			reply.ID = "reply"
			reply.ThreadId = &parent.ID
			require.NoError(t, p.SaveMessages([]*common.Message{parent}))
			if !parentFirst {
				require.NoError(t, p.UpsertThread(parent.ID, chat.ID, parent.ID, "Thread"))
				require.NoError(t, p.SetThreadCreator(parent.ID, chat.ID, "different-creator"))
				require.NoError(t, p.SaveMessages([]*common.Message{reply}))
			}
			// A local-only hide must not delete the shared thread.
			parent.DeletedForMe = true
			require.NoError(t, p.SaveMessages([]*common.Message{parent}))
			if !parentFirst {
				thread, err := p.ThreadByID(chat.ID, parent.ID)
				require.NoError(t, err)
				require.False(t, thread.Deleted)
			}
			parent.Deleted = true
			parent.DeletedBy = "admin"
			require.NoError(t, p.SaveMessages([]*common.Message{parent}))
			if parentFirst {
				require.NoError(t, p.SaveMessages([]*common.Message{reply}))
				require.True(t, reply.Deleted)
				require.NoError(t, p.UpsertThread(parent.ID, chat.ID, parent.ID, "Late metadata"))
			}
			thread, err := p.ThreadByID(chat.ID, parent.ID)
			require.NoError(t, err)
			require.True(t, thread.Deleted)
			require.Equal(t, "admin", thread.DeletedBy)
			storedReply, err := p.MessageByID(reply.ID)
			require.NoError(t, err)
			require.True(t, storedReply.Deleted)
			late := buildTestMessage(*chat)
			late.ID = "late"
			late.ThreadId = &parent.ID
			require.NoError(t, p.SaveMessages([]*common.Message{late}))
			require.True(t, late.Deleted)
			var count int
			require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM user_messages`).Scan(&count))
			require.Equal(t, 3, count)
		})
	}
}

func (s *MessengerThreadsSuite) TestDeleteParentCascadesWithoutThreadCreatorPermission() {
	chat := CreateOneToOneChat("self", &s.m.identity.PublicKey, s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat))
	parent := buildTestMessage(*chat)
	parent.ID = "root"
	parent.From = s.m.myHexIdentity()
	reply := buildTestMessage(*chat)
	reply.ID = "reply"
	reply.ThreadId = &parent.ID
	s.Require().NoError(s.m.SaveMessages([]*common.Message{parent, reply}))
	s.Require().NoError(s.m.persistence.UpsertThread(parent.ID, chat.ID, parent.ID, "Thread"))
	s.Require().NoError(s.m.persistence.SetThreadCreator(parent.ID, chat.ID, "another-creator"))
	response, err := s.m.DeleteMessageAndSend(context.Background(), parent.ID)
	s.Require().NoError(err)
	s.Require().Len(response.Threads(), 1)
	s.Require().True(response.Threads()[0].Deleted)
	stored, err := s.m.persistence.MessageByID(reply.ID)
	s.Require().NoError(err)
	s.Require().True(stored.Deleted)
}

func (s *MessengerThreadsSuite) TestDeleteParentCascadesOnPeer() {
	sender := s.m
	receiver := s.newMessenger()
	receiver.featureFlags.Threads = true
	senderChat := CreateOneToOneChat("receiver", &receiver.identity.PublicKey, sender.getTimesource())
	receiverChat := CreateOneToOneChat("sender", &sender.identity.PublicKey, receiver.getTimesource())
	s.Require().NoError(sender.SaveChat(senderChat))
	_, err := sender.Join(senderChat)
	s.Require().NoError(err)
	s.Require().NoError(receiver.SaveChat(receiverChat))
	_, err = receiver.Join(receiverChat)
	s.Require().NoError(err)
	parent := buildTestMessage(*senderChat)
	_, err = sender.SendChatMessage(context.Background(), parent)
	s.Require().NoError(err)
	_, err = WaitOnMessengerResponse(receiver, func(response *MessengerResponse) bool {
		for _, msg := range response.Messages() {
			if msg.ID == parent.ID {
				return true
			}
		}
		return false
	}, "parent not received")
	s.Require().NoError(err)
	// The peer creates a thread from our message; we still own its parent.
	_, err = receiver.CreateThread(receiverChat.ID, parent.ID)
	s.Require().NoError(err)
	reply := buildTestMessage(*receiverChat)
	reply.ThreadId = &parent.ID
	_, err = receiver.SendChatMessage(context.Background(), reply)
	s.Require().NoError(err)
	_, err = WaitOnMessengerResponse(sender, func(response *MessengerResponse) bool {
		thread, err := sender.persistence.ThreadByID(senderChat.ID, parent.ID)
		_, replyErr := sender.persistence.MessageByID(reply.ID)
		return err == nil && thread.CreatorID != "" && replyErr == nil
	}, "thread and reply not received")
	s.Require().NoError(err)
	response, err := sender.DeleteMessageAndSend(context.Background(), parent.ID)
	s.Require().NoError(err)
	s.Require().Len(response.Threads(), 1)
	s.Require().True(response.Threads()[0].Deleted)
	_, err = WaitOnMessengerResponse(receiver, func(response *MessengerResponse) bool {
		for _, thread := range response.Threads() {
			if thread.ThreadID == parent.ID && thread.Deleted {
				return true
			}
		}
		return false
	}, "cascaded deletion not received")
	s.Require().NoError(err)
	for _, messenger := range []*Messenger{sender, receiver} {
		storedParent, err := messenger.persistence.MessageByID(parent.ID)
		s.Require().NoError(err)
		s.Require().True(storedParent.Deleted)
		storedReply, err := messenger.persistence.MessageByID(reply.ID)
		s.Require().NoError(err)
		s.Require().True(storedReply.Deleted)
	}
}
