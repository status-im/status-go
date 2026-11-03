package protocol

import (
	"context"
	"errors"

	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	"github.com/status-im/status-go/internal/protocol/requests"
)

// FollowThread shows the thread in the chat list. Archived threads can be followed.
func (m *Messenger) FollowThread(ctx context.Context, request *requests.FollowThread) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	return m.setThreadFollowed(ctx, request.ChatID, request.ThreadID, true)
}

// UnfollowThread hides the thread from the chat list.
func (m *Messenger) UnfollowThread(ctx context.Context, request *requests.UnfollowThread) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	return m.setThreadFollowed(ctx, request.ChatID, request.ThreadID, false)
}

func (m *Messenger) setThreadFollowed(ctx context.Context, chatID, threadID string, followed bool) (*MessengerResponse, error) {
	if !m.featureFlags.Threads {
		return nil, ErrThreadFeatureDisabled
	}
	chat, ok := m.allChats.Load(chatID)
	if !ok {
		return nil, ErrChatNotFoundError
	}
	thread, err := m.persistence.ThreadByID(chatID, threadID)
	if err != nil {
		return nil, err
	}
	if thread.Deleted {
		return nil, ErrThreadDeleted
	}

	clock, _ := chat.NextClockAndTimestamp(m.getTimesource())
	// A paired device may have stored a newer choice than our chat clock.
	if clock <= thread.FollowedClock {
		clock = thread.FollowedClock + 1
		chat.LastClockValue = clock
	}
	if _, err := m.persistence.SetThreadFollowed(chatID, threadID, followed, clock); err != nil {
		return nil, err
	}
	if err := m.syncThreadFollow(ctx, chatID, threadID, followed, clock, m.dispatchMessage); err != nil {
		return nil, err
	}

	thread, err = m.persistence.ThreadWithSummaryByID(chatID, threadID)
	if err != nil {
		return nil, err
	}
	response := &MessengerResponse{}
	response.AddThread(thread)
	return response, nil
}

func (m *Messenger) syncThreadFollow(ctx context.Context, chatID, threadID string, followed bool, clock uint64, rawMessageHandler RawMessageHandler) error {
	if !m.hasPairedDevices() {
		return nil
	}

	_, chat := m.getLastClockWithRelatedChat()

	encodedMessage, err := proto.Marshal(&protobuf.SyncThreadFollow{
		Clock:    clock,
		ChatId:   chatID,
		ThreadId: threadID,
		Followed: followed,
	})
	if err != nil {
		return err
	}

	_, err = rawMessageHandler(ctx, common.RawMessage{
		LocalChatID: chat.ID,
		Payload:     encodedMessage,
		MessageType: protobuf.ApplicationMetadataMessage_SYNC_THREAD_FOLLOW,
		ResendType:  common.ResendTypeDataSync,
	})
	return err
}

func (m *Messenger) HandleSyncThreadFollow(ctx context.Context, state *ReceivedMessageState, message *protobuf.SyncThreadFollow, statusMessage *common.StatusMessage) error {
	if message.ChatId == "" || message.ThreadId == "" || message.Clock == 0 {
		return errors.New("invalid thread follow sync message")
	}
	if _, ok := m.allChats.Load(message.ChatId); !ok {
		return ErrChatNotFound
	}

	_, err := m.persistence.ThreadByID(message.ChatId, message.ThreadId)
	if errors.Is(err, common.ErrRecordNotFound) {
		var syncedAt uint64
		if state.CurrentMessageState != nil {
			syncedAt = state.CurrentMessageState.WhisperTimestamp
		}
		// Keep the choice until the thread itself arrives.
		err = m.persistence.UpsertPendingCreatorThreadAt(message.ThreadId, message.ChatId, message.ThreadId, "", syncedAt)
	}
	if err != nil {
		return err
	}

	updated, err := m.persistence.SetThreadFollowed(message.ChatId, message.ThreadId, message.Followed, message.Clock)
	if err != nil || !updated {
		return err
	}

	thread, err := m.persistence.ThreadWithSummaryByID(message.ChatId, message.ThreadId)
	if err != nil {
		return err
	}
	state.Response.AddThread(thread)
	return nil
}

// updateThreadFollowsFromMessages follows threads the user replied in, and
// auto-follows threads started on the user's messages. Call it after the
// messages are saved and before threads are added to the response.
func (m *Messenger) updateThreadFollowsFromMessages(messages []*common.Message, response *MessengerResponse) error {
	if !m.featureFlags.Threads {
		return nil
	}
	ownID := m.myHexIdentity()
	for _, message := range messages {
		if message == nil || message.Deleted || message.DeletedForMe || message.LocalChatID == "" {
			continue
		}
		threadID := message.GetThreadId()
		if threadID == "" {
			if message.From != ownID {
				continue
			}
			threads, err := m.persistence.ThreadsWithSummariesByParentMessageIDs(
				message.LocalChatID, []string{message.ID}, defaultThreadParticipantPreviewLimit)
			if err != nil {
				return err
			}
			for _, thread := range threads {
				updated, err := m.persistence.AutoFollowOwnedThread(message.LocalChatID, thread.ThreadID, ownID)
				if err != nil {
					return err
				}
				if updated {
					thread.Followed = true
					response.AddThread(thread)
				}
			}
			continue
		}
		var err error
		if message.From == ownID {
			_, err = m.persistence.SetThreadFollowed(message.LocalChatID, threadID, true, message.Clock)
		} else {
			_, err = m.persistence.AutoFollowOwnedThread(message.LocalChatID, threadID, ownID)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
