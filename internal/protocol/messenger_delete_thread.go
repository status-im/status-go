package protocol

import (
	"context"
	"crypto/ecdsa"
	"errors"

	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/protocol/contacts"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	"github.com/status-im/status-go/internal/protocol/requests"
)

var ErrThreadDeleted = errors.New("thread is deleted")
var ErrThreadDeletePermission = errors.New("only the thread creator or an admin can delete this thread")

func (m *Messenger) senderCanDeleteThread(chat *Chat, thread *Thread, signer *ecdsa.PublicKey) (bool, error) {
	// Deletion shares edit permissions, including community token masters.
	return m.senderCanEditThread(chat, thread, signer)
}

func (m *Messenger) DeleteThread(ctx context.Context, request *requests.DeleteThread) (*MessengerResponse, error) {
	if !m.featureFlags.Threads {
		return nil, ErrThreadFeatureDisabled
	}
	if err := request.Validate(); err != nil {
		return nil, err
	}
	chat, ok := m.allChats.Load(request.ChatID)
	if !ok {
		return nil, ErrChatNotFoundError
	}
	thread, err := m.persistence.ThreadByID(chat.ID, request.ThreadID)
	if err != nil {
		return nil, err
	}
	allowed, err := m.senderCanDeleteThread(chat, thread, &m.identity.PublicKey)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrThreadDeletePermission
	}
	clock, timestamp := chat.NextClockAndTimestamp(m.getTimesource())
	deletion := &DeleteThread{DeleteThread: &protobuf.DeleteThread{ChatId: chat.ID, ThreadId: thread.ThreadID, Clock: clock, Timestamp: timestamp}}
	encoded, err := m.encodeChatEntity(chat, deletion)
	if err != nil {
		return nil, err
	}
	// Dispatch first so a failed dispatch can be retried without hiding the thread.
	if _, err = m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID: chat.ID, Payload: encoded, MessageType: protobuf.ApplicationMetadataMessage_DELETE_THREAD,
		SkipGroupMessageWrap: true, ResendType: chat.DefaultResendType(),
	}); err != nil {
		return nil, err
	}
	if err = m.persistence.SoftDeleteThread(chat.ID, thread.ThreadID, clock, contacts.ContactIDFromPublicKey(&m.identity.PublicKey), timestamp); err != nil {
		return nil, err
	}
	thread, err = m.persistence.ThreadWithSummaryByID(chat.ID, thread.ThreadID)
	if err != nil {
		return nil, err
	}
	response := &MessengerResponse{}
	if err := m.addDeletedThreadToResponse(response, chat, thread); err != nil {
		return nil, err
	}
	return response, nil
}

func (m *Messenger) HandleDeleteThread(ctx context.Context, state *ReceivedMessageState, deletionProto *protobuf.DeleteThread, statusMessage *common.StatusMessage) error {
	if !m.featureFlags.Threads {
		return ErrThreadFeatureDisabled
	}
	if deletionProto == nil || deletionProto.ChatId == "" || deletionProto.ThreadId == "" || deletionProto.Clock == 0 {
		return requests.ErrDeleteThreadInvalidRequest
	}
	if err := validateClockValue(deletionProto.Clock, state.CurrentMessageState.WhisperTimestamp); err != nil {
		return err
	}
	deletion := &DeleteThread{DeleteThread: deletionProto, SigPubKey: state.CurrentMessageState.PublicKey}
	if deletion.SigPubKey == nil {
		return ErrThreadSenderRequired
	}
	chat, err := m.matchChatEntity(deletion, protobuf.ApplicationMetadataMessage_DELETE_THREAD)
	if err != nil {
		return err
	}
	if !chat.OneToOne() && chat.ID != deletion.ChatId {
		return errors.New("thread deletion chat does not match message chat")
	}
	thread, err := m.persistence.ThreadByID(chat.ID, deletion.ThreadId)
	if err != nil && !errors.Is(err, common.ErrRecordNotFound) {
		return err
	}
	if thread == nil {
		thread = &Thread{ThreadID: deletion.ThreadId, ChatID: chat.ID, CreatorPending: true}
	}
	allowed, err := m.senderCanDeleteThread(chat, thread, deletion.SigPubKey)
	if err != nil {
		return err
	}
	if !allowed {
		if thread.CreatorPending {
			return m.bufferPendingThreadOperation(chat, &ThreadMetadata{ThreadMetadata: &protobuf.ThreadMetadata{ThreadId: deletion.ThreadId, Clock: deletion.Clock}, SigPubKey: deletion.SigPubKey}, "", true, deletion.Timestamp)
		}
		return ErrThreadDeletePermission
	}
	// Admin deletion can be validated before creation arrives. Keep a placeholder
	// tombstone so subsequent replies are deleted immediately.
	if err = m.persistence.UpsertPendingCreatorThread(thread.ThreadID, chat.ID, thread.ThreadID, ""); err != nil {
		return err
	}
	if err = m.persistence.SoftDeleteThread(chat.ID, thread.ThreadID, deletion.Clock, contacts.ContactIDFromPublicKey(deletion.SigPubKey), deletion.Timestamp); err != nil {
		return err
	}
	if chat.LastClockValue < deletion.Clock {
		chat.LastClockValue = deletion.Clock
		if err = m.saveChat(chat); err != nil {
			return err
		}
		state.Response.AddChat(chat)
	}
	thread, err = m.persistence.ThreadWithSummaryByID(chat.ID, thread.ThreadID)
	if err != nil {
		return err
	}
	return m.addDeletedThreadToResponse(state.Response, chat, thread)
}

// A thread reply may also be the chat's cached last message. Replace that preview
// after deletion so neither the persisted chat nor its response exposes the reply.
func (m *Messenger) addDeletedThreadToResponse(response *MessengerResponse, chat *Chat, thread *Thread) error {
	if chat.LastMessage != nil && chat.LastMessage.GetThreadId() == thread.ThreadID {
		messages, err := m.persistence.LatestMessageByChatID(chat.ID)
		if err != nil {
			return err
		}
		chat.LastMessage = nil
		if len(messages) > 0 {
			chat.LastMessage = messages[0]
		}
		if err := m.saveChat(chat); err != nil {
			return err
		}
		response.AddChat(chat)
	}
	response.AddThread(thread)
	return nil
}

// Called only after the parent message deletion has passed its own author/admin
// permission checks. A parent author need not also be the thread creator.
func (m *Messenger) deleteThreadForDeletedParent(chat *Chat, message *common.Message, clock uint64, actor string) error {
	if message.GetThreadId() != "" {
		return nil
	}
	thread, err := m.persistence.ThreadByID(chat.ID, message.ID)
	if errors.Is(err, common.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return m.persistence.SoftDeleteThread(chat.ID, thread.ThreadID, clock, actor, m.getTimesource().GetCurrentTime())
}
