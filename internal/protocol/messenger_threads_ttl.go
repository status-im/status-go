package protocol

import (
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/panics"
	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/signal"
)

var (
	// threadInactivityTTL is how long a thread may go without new messages
	// before it is archived (hidden from the chat list).
	threadInactivityTTL = 7 * 24 * time.Hour
	// threadArchiveSweepInterval is how often threads that crossed the TTL
	// are reported to the client.
	threadArchiveSweepInterval = 10 * time.Minute
)

func (db sqlitePersistence) threadNow() uint64 {
	if db.threadClock != nil {
		return db.threadClock()
	}
	return uint64(time.Now().UnixMilli())
}

// threadActivityAt bounds an activity time: unknown (0) means now, and
// future times are capped at now so a skewed or malicious sender clock cannot
// keep a thread active.
func (db sqlitePersistence) threadActivityAt(at uint64) uint64 {
	now := db.threadNow()
	if at == 0 || at > now {
		return now
	}
	return at
}

// messageActivityTime is when a message was sent, so that old messages fetched
// later (e.g. from store nodes) do not revive an inactive thread.
func messageActivityTime(message *common.Message) uint64 {
	if message.Timestamp != 0 {
		return message.Timestamp
	}
	return message.WhisperTimestamp
}

// threadArchiveCutoff returns the activity time before which threads are archived.
func threadArchiveCutoff(now uint64) uint64 {
	ttl := uint64(threadInactivityTTL.Milliseconds())
	if now <= ttl {
		return 0
	}
	return now - ttl
}

func isThreadArchived(lastActivityAt, now uint64) bool {
	return lastActivityAt < threadArchiveCutoff(now)
}

func (db sqlitePersistence) applyThreadArchiveState(threads ...*Thread) {
	now := db.threadNow()
	for _, thread := range threads {
		if thread != nil {
			thread.Archived = isThreadArchived(thread.LastActivityAt, now)
		}
	}
}

// touchThreadActivity records new messages as thread activity at the time they
// were sent, which unarchives the thread only if the message itself is recent
// enough. Edits, deletions and reactions are not activity.
func (m *Messenger) touchThreadActivity(messages []*common.Message, isNew func(*common.Message) bool) {
	if !m.featureFlags.Threads {
		return
	}
	type threadRef struct{ chatID, threadID string }
	latest := make(map[threadRef]uint64)
	var order []threadRef
	for _, message := range messages {
		if message == nil || message.GetThreadId() == "" || message.LocalChatID == "" || !isNew(message) {
			continue
		}
		ref := threadRef{message.LocalChatID, message.GetThreadId()}
		at := messageActivityTime(message)
		previous, ok := latest[ref]
		if !ok {
			order = append(order, ref)
		}
		if !ok || at > previous {
			latest[ref] = at
		}
	}
	for _, ref := range order {
		if err := m.persistence.TouchThreadActivity(ref.chatID, ref.threadID, latest[ref]); err != nil {
			m.logger.Warn("failed to record thread activity",
				zap.String("chatID", ref.chatID), zap.String("threadID", ref.threadID), zap.Error(err))
		}
	}
}

// startThreadArchiveLoop periodically reports threads that became archived
// while the messenger was running. Threads already archived at startup are
// reported through the archived flag when clients load threads.
func (m *Messenger) startThreadArchiveLoop() {
	if !m.featureFlags.Threads {
		return
	}
	lastSweepAt := m.getTimesource().GetCurrentTime()

	m.shutdownWaitGroup.Add(1)
	go func() {
		defer panics.LogOnPanic()
		defer m.shutdownWaitGroup.Done()

		ticker := time.NewTicker(threadArchiveSweepInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if m.isPaused() {
					continue
				}
				now := m.getTimesource().GetCurrentTime()
				response, err := m.sweepArchivedThreads(lastSweepAt, now)
				if err != nil {
					m.logger.Warn("failed to sweep archived threads", zap.Error(err))
					continue
				}
				lastSweepAt = now
				if !response.IsEmpty() {
					signal.SendNewMessages(response)
				}
			case <-m.quit:
				return
			}
		}
	}()
}

// sweepArchivedThreads returns a response with the threads that crossed the
// inactivity TTL between the two given times.
func (m *Messenger) sweepArchivedThreads(lastSweepAt, now uint64) (*MessengerResponse, error) {
	response := &MessengerResponse{}
	if now <= lastSweepAt {
		return response, nil
	}
	identities, err := m.persistence.ThreadsInactiveBetween(threadArchiveCutoff(lastSweepAt), threadArchiveCutoff(now))
	if err != nil || len(identities) == 0 {
		return response, err
	}
	for start := 0; start < len(identities); start += threadSummaryCacheCapacity {
		end := start + threadSummaryCacheCapacity
		if end > len(identities) {
			end = len(identities)
		}
		threads, err := m.persistence.threadsWithSummariesByIDs(identities[start:end])
		if err != nil {
			return nil, err
		}
		response.AddThreads(threads)
	}
	return response, nil
}
