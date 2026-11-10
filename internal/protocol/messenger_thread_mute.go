package protocol

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/status-im/status-go/internal/protocol/requests"
)

// Revisions outlive individual Messenger instances within the embedded backend.
// They order preference events in the UI; snapshots intentionally carry revision zero.
var threadMuteRevision atomic.Uint64

func addThreadMuteUpdate(response *MessengerResponse, thread *Thread) {
	thread.MuteRevision = threadMuteRevision.Add(1)
	response.AddThread(thread)
}

// muteThread shares the chat muting API and intervals, but stores the preference on the thread.
func (m *Messenger) muteThread(request *requests.MuteChat) (time.Time, error) {
	if err := m.validateThreadMute(request.ChatID, request.ThreadID); err != nil {
		return time.Time{}, err
	}
	switch request.MutedType {
	case MuteFor15Min, MuteFor1Hr, MuteFor8Hr, MuteFor24Hr, MuteFor1Week, MuteTillUnmuted, MuteTill1Min:
	default:
		return time.Time{}, errors.New("mute-thread: invalid interval")
	}
	deadline := muteDeadline(request.MutedType)
	response, err := m.updateThreadMute(request.ChatID, request.ThreadID, true, deadline)
	if err != nil {
		return time.Time{}, err
	}
	m.signalSession.SendNewMessages(response)
	return deadline, nil
}

func (m *Messenger) UnmuteThread(chatID, threadID string) error {
	if err := m.validateThreadMute(chatID, threadID); err != nil {
		return err
	}
	response, err := m.updateThreadMute(chatID, threadID, false, time.Time{})
	if err != nil {
		return err
	}
	m.signalSession.SendNewMessages(response)
	return nil
}

func (m *Messenger) updateThreadMute(chatID, threadID string, muted bool, deadline time.Time) (*MessengerResponse, error) {
	m.threadMuteMutex.Lock()
	defer m.threadMuteMutex.Unlock()
	var till interface{} = 0
	if muted {
		till = deadline.Format(time.RFC3339)
	}
	thread, err := m.persistence.setThreadMuted(chatID, threadID, muted, till)
	if err != nil {
		return nil, err
	}
	response := &MessengerResponse{}
	addThreadMuteUpdate(response, thread)
	// Publish after releasing the lock: client callbacks must never run under it.
	return response, nil
}

func (m *Messenger) validateThreadMute(chatID, threadID string) error {
	if !m.featureFlags.Threads {
		return ErrThreadFeatureDisabled
	}
	if chatID == "" || threadID == "" {
		return errors.New("mute-thread: invalid params")
	}
	chat, ok := m.allChats.Load(chatID)
	if !ok {
		return ErrChatNotFoundError
	}
	if !chat.SupportsThreads() {
		return ErrThreadsNotSupportedForChatType
	}
	return nil
}

// Called by the existing chat expiration watcher; state changes use the same response signal.
func (m *Messenger) unmuteExpiredThreads(now time.Time, response *MessengerResponse) error {
	m.threadMuteMutex.Lock()
	defer m.threadMuteMutex.Unlock()
	threads, err := m.persistence.unmuteExpiredThreads(context.Background(), now)
	if err != nil {
		return err
	}
	for _, thread := range threads {
		addThreadMuteUpdate(response, thread)
	}
	return nil
}

func (m *Messenger) isThreadMuted(chatID, threadID string) (bool, error) {
	if threadID == "" || !m.featureFlags.Threads {
		return false, nil
	}
	return m.persistence.IsThreadMuted(chatID, threadID)
}
