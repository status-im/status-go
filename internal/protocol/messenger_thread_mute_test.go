package protocol

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/db/appdatabase"
	"github.com/status-im/status-go/internal/protocol/requests"
	protocolsqlite "github.com/status-im/status-go/internal/protocol/sqlite"
	"github.com/status-im/status-go/internal/testutils"
)

func threadMuteMessenger(t *testing.T) *Messenger {
	t.Helper()
	db, err := openTestDB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	p := newSQLitePersistence(db)
	seedThreadSummaryCache(t, p)
	require.NoError(t, p.UpsertThread("sibling", testPublicChatID, "other-parent", "Other thread"))
	m := &Messenger{persistence: p, allChats: &chatMap{}}
	m.featureFlags.Threads = true
	m.allChats.Store(testPublicChatID, &Chat{ID: testPublicChatID, ChatType: ChatTypeCommunityChat, Active: true})
	return m
}

func TestThreadMuteIsolationAndPersistence(t *testing.T) {
	m := threadMuteMessenger(t)
	request := &requests.MuteChat{ChatID: testPublicChatID, ThreadID: "thread", MutedType: MuteTillUnmuted}
	deadline, err := m.MuteChat(request)
	require.NoError(t, err)
	require.True(t, deadline.IsZero())
	sibling, err := m.persistence.ThreadByID(testPublicChatID, "sibling")
	require.NoError(t, err)
	require.False(t, sibling.Muted)
	chat, ok := m.allChats.Load(testPublicChatID)
	require.True(t, ok)
	require.False(t, chat.Muted)
	// Recreate persistence to ensure no preference depends on messenger memory.
	p := newSQLitePersistence(m.persistence.db)
	threads, err := p.ThreadsByChatID(testPublicChatID)
	require.NoError(t, err)
	require.Len(t, threads, 2)
	for _, thread := range threads {
		require.Equal(t, thread.ThreadID == "thread", thread.Muted)
	}
	threads, err = p.ThreadsByChatIDs([]string{testPublicChatID})
	require.NoError(t, err)
	for _, thread := range threads {
		require.Equal(t, thread.ThreadID == "thread", thread.Muted)
	}
	summaries, err := p.ThreadsWithSummariesByParentMessageIDs(testPublicChatID, []string{"parent"}, 6)
	require.NoError(t, err)
	require.True(t, summaries[0].Muted)
	require.True(t, summaries[0].MuteTill.IsZero())
	require.EqualValues(t, 2, summaries[0].MessagesCount)
	require.NoError(t, m.UnmuteThread(request.ChatID, request.ThreadID))
	thread, err := p.ThreadByID(request.ChatID, request.ThreadID)
	require.NoError(t, err)
	require.False(t, thread.Muted)
	require.Zero(t, thread.MuteTill.Unix())
	// Unmuting a thread must preserve an independently muted parent.
	chat.Muted = true
	require.NoError(t, m.UnmuteThread(request.ChatID, request.ThreadID))
	require.True(t, chat.Muted)
}

func TestThreadMuteIntervalsAndExpiry(t *testing.T) {
	m := threadMuteMessenger(t)
	for variation, duration := range map[requests.MutingVariation]time.Duration{
		MuteTill1Min: time.Minute, MuteFor15Min: 15 * time.Minute, MuteFor1Hr: time.Hour,
		MuteFor8Hr: 8 * time.Hour, MuteFor24Hr: 24 * time.Hour, MuteFor1Week: 7 * 24 * time.Hour,
	} {
		deadline, err := m.MuteChat(&requests.MuteChat{ChatID: testPublicChatID, ThreadID: "thread", MutedType: variation})
		require.NoError(t, err)
		require.WithinDuration(t, time.Now().Add(duration), deadline, 2*time.Second)
		thread, err := m.persistence.ThreadByID(testPublicChatID, "thread")
		require.NoError(t, err)
		require.True(t, thread.Muted)
		require.True(t, thread.MuteTill.Equal(deadline))
		data, err := json.Marshal(thread)
		require.NoError(t, err)
		var decoded struct {
			MuteTill time.Time `json:"muteTill"`
		}
		require.NoError(t, json.Unmarshal(data, &decoded))
		require.True(t, decoded.MuteTill.Equal(deadline))
	}
	now := time.Now()
	require.NoError(t, m.persistence.MuteThread(testPublicChatID, "thread", now.Add(-time.Minute)))
	require.NoError(t, m.persistence.MuteThread(testPublicChatID, "sibling", time.Time{}))
	response := &MessengerResponse{}
	require.NoError(t, m.unmuteExpiredThreads(now, response))
	require.Len(t, response.Threads(), 1)
	require.Equal(t, "thread", response.Threads()[0].ThreadID)
	require.False(t, response.Threads()[0].Muted)
	require.Zero(t, response.Threads()[0].MuteTill.Unix())
	muted, err := m.isThreadMuted(testPublicChatID, "thread")
	require.NoError(t, err)
	require.False(t, muted)
	sibling, err := m.persistence.ThreadByID(testPublicChatID, "sibling")
	require.NoError(t, err)
	require.True(t, sibling.Muted)
	require.True(t, sibling.MuteTill.IsZero())
	// Future deadlines must not be expired by the shared watcher.
	require.NoError(t, m.persistence.MuteThread(testPublicChatID, "thread", now.Add(time.Hour)))
	response = &MessengerResponse{}
	require.NoError(t, m.unmuteExpiredThreads(now, response))
	require.Empty(t, response.Threads())
}

func TestThreadMuteValidation(t *testing.T) {
	m := threadMuteMessenger(t)
	for _, request := range []*requests.MuteChat{
		{ThreadID: "thread", MutedType: MuteTillUnmuted},
		{ChatID: testPublicChatID, ThreadID: "thread", MutedType: -1},
		{ChatID: testPublicChatID, ThreadID: "thread", MutedType: 0},
		{ChatID: testPublicChatID, ThreadID: "thread", MutedType: 9},
		{ChatID: "wrong-chat", ThreadID: "thread", MutedType: MuteTillUnmuted},
		{ChatID: testPublicChatID, ThreadID: "missing", MutedType: MuteTillUnmuted},
	} {
		_, err := m.MuteChat(request)
		require.Error(t, err)
	}
	require.Error(t, m.UnmuteThread(testPublicChatID, ""))
	require.Error(t, m.UnmuteThread(testPublicChatID, "missing"))
	thread, err := m.persistence.ThreadByID(testPublicChatID, "thread")
	require.NoError(t, err)
	require.False(t, thread.Muted)
	m.featureFlags.Threads = false
	_, err = m.MuteChat(&requests.MuteChat{ChatID: testPublicChatID, ThreadID: "thread", MutedType: MuteTillUnmuted})
	require.ErrorIs(t, err, ErrThreadFeatureDisabled)
	require.ErrorIs(t, m.UnmuteThread(testPublicChatID, "thread"), ErrThreadFeatureDisabled)
}

func TestThreadMuteSuppressesOnlyThreadActivity(t *testing.T) {
	m := threadMuteMessenger(t)
	request := &requests.MuteChat{ChatID: testPublicChatID, ThreadID: "thread", MutedType: MuteTillUnmuted}
	_, err := m.MuteChat(request)
	require.NoError(t, err)
	for _, threadID := range []string{"thread", "sibling", ""} {
		muted, err := m.isThreadMuted(testPublicChatID, threadID)
		require.NoError(t, err)
		require.Equal(t, threadID == "thread", muted)
	}
	message, err := m.persistence.MessageByID("reply")
	require.NoError(t, err)
	message.New = true
	message.Mentioned = true
	state := &ReceivedMessageState{AllChats: m.allChats, Response: &MessengerResponse{}}
	require.NoError(t, state.addNewActivityCenterNotification(ecdsa.PublicKey{}, m, message, nil))
	require.Empty(t, state.Response.ActivityCenterNotifications())
	require.NoError(t, m.UnmuteThread(request.ChatID, request.ThreadID))
	muted, err := m.isThreadMuted(testPublicChatID, "thread")
	require.NoError(t, err)
	require.False(t, muted)
}

func TestThreadMuteMatchesChatStorage(t *testing.T) {
	m := threadMuteMessenger(t)
	assertSameState := func() {
		t.Helper()
		var chatMuted, threadMuted bool
		var chatTill, threadTill, chatType, threadType string
		require.NoError(t, m.persistence.db.QueryRow(`SELECT muted, CAST(muted_till AS TEXT), typeof(muted_till) FROM chats WHERE id = ?`, testPublicChatID).Scan(&chatMuted, &chatTill, &chatType))
		require.NoError(t, m.persistence.db.QueryRow(`SELECT muted, CAST(muted_till AS TEXT), typeof(muted_till) FROM threads WHERE chat_id = ? AND thread_id = ?`, testPublicChatID, "thread").Scan(&threadMuted, &threadTill, &threadType))
		require.Equal(t, chatMuted, threadMuted)
		require.Equal(t, chatTill, threadTill)
		require.Equal(t, chatType, threadType)
	}
	for _, deadline := range []time.Time{time.Now().Add(time.Hour).Truncate(time.Second), {}} {
		require.NoError(t, m.persistence.MuteChat(testPublicChatID, deadline))
		require.NoError(t, m.persistence.MuteThread(testPublicChatID, "thread", deadline))
		assertSameState()
	}
	require.NoError(t, m.persistence.UnmuteChat(testPublicChatID))
	require.NoError(t, m.persistence.UnmuteThread(testPublicChatID, "thread"))
	assertSameState()
	chat, err := m.persistence.Chat(testPublicChatID)
	require.NoError(t, err)
	thread, err := m.persistence.ThreadByID(testPublicChatID, "thread")
	require.NoError(t, err)
	require.True(t, chat.MuteTill.Equal(thread.MuteTill))
}

func TestThreadMuteConcurrentRenewalAndExpiration(t *testing.T) {
	db, teardown, err := testutils.SetupTestSQLDB(appdatabase.DbInitializer{}, "thread-mute-concurrency")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, teardown()) })
	require.NoError(t, protocolsqlite.Migrate(db))
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(2)
	p := newSQLitePersistence(db)
	seedThreadSummaryCache(t, p)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	now := time.Now().Truncate(time.Second)
	for i := 0; i < 30; i++ {
		require.NoError(t, p.MuteThread(testPublicChatID, "thread", now.Add(-time.Minute)))
		start := make(chan struct{})
		results := make(chan error, 2)
		var workers sync.WaitGroup
		workers.Add(2)
		go func() {
			defer workers.Done()
			<-start
			results <- p.MuteThread(testPublicChatID, "thread", now.Add(time.Hour))
		}()
		go func() {
			defer workers.Done()
			<-start
			_, err := p.unmuteExpiredThreads(ctx, now)
			results <- err
		}()
		close(start)
		workers.Wait()
		require.NoError(t, <-results)
		require.NoError(t, <-results)
		thread, err := p.ThreadByID(testPublicChatID, "thread")
		require.NoError(t, err)
		require.True(t, thread.Muted)
		require.True(t, thread.MuteTill.Equal(now.Add(time.Hour)))
	}
}

func TestThreadMuteWriteAndResponseRollbackTogether(t *testing.T) {
	for _, expire := range []bool{false, true} {
		t.Run(map[bool]string{false: "mute", true: "expiration"}[expire], func(t *testing.T) {
			m := threadMuteMessenger(t)
			p := m.persistence
			now := time.Now()
			if expire {
				require.NoError(t, p.MuteThread(testPublicChatID, "thread", now.Add(-time.Minute)))
			}
			// Force response metadata to become unavailable after the UPDATE. Both the
			// preference write and this trigger must roll back on the resulting read error.
			_, err := p.db.Exec(`CREATE TRIGGER remove_muted_thread AFTER UPDATE OF muted ON threads
    WHEN NEW.thread_id = 'thread' BEGIN DELETE FROM threads WHERE thread_id = NEW.thread_id; END`)
			require.NoError(t, err)
			if expire {
				_, err = p.unmuteExpiredThreads(context.Background(), now)
			} else {
				err = p.MuteThread(testPublicChatID, "thread", now.Add(time.Hour))
			}
			require.Error(t, err)
			thread, err := p.ThreadByID(testPublicChatID, "thread")
			require.NoError(t, err)
			require.Equal(t, expire, thread.Muted)
			// No transaction or RETURNING cursor may retain the sole pooled connection.
			p.db.SetMaxOpenConns(1)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			require.NoError(t, p.db.PingContext(ctx))
		})
	}
}

func TestThreadMuteDelayedExpirationHasOlderRevision(t *testing.T) {
	m := threadMuteMessenger(t)
	now := time.Now()
	require.NoError(t, m.persistence.MuteThread(testPublicChatID, "thread", now.Add(-time.Minute)))
	expired := &MessengerResponse{}
	require.NoError(t, m.unmuteExpiredThreads(now, expired))
	require.Len(t, expired.Threads(), 1)
	oldRevision := expired.Threads()[0].MuteRevision
	require.NotZero(t, oldRevision)
	// The watcher may deliver its response after a newer mutation was committed.
	// Preference revisions make that delayed response distinguishable in the UI.
	thread, err := m.persistence.setThreadMuted(testPublicChatID, "thread", true, now.Add(time.Hour).Format(time.RFC3339))
	require.NoError(t, err)
	renewed := &MessengerResponse{}
	addThreadMuteUpdate(renewed, thread)
	require.Greater(t, renewed.Threads()[0].MuteRevision, oldRevision)
	require.True(t, renewed.Threads()[0].Muted)
}

func TestThreadMuteConcurrentMessengerOperations(t *testing.T) {
	m := threadMuteMessenger(t)
	now := time.Now()
	start := make(chan struct{})
	results := make(chan error, 6)
	for worker := 0; worker < 6; worker++ {
		go func(worker int) {
			<-start
			for i := 0; i < 10; i++ {
				var err error
				switch worker % 3 {
				case 0:
					_, err = m.MuteChat(&requests.MuteChat{ChatID: testPublicChatID, ThreadID: "thread", MutedType: MuteFor1Hr})
				case 1:
					err = m.UnmuteThread(testPublicChatID, "thread")
				case 2:
					err = m.unmuteExpiredThreads(now, &MessengerResponse{})
				}
				if err != nil {
					results <- err
					return
				}
			}
			results <- nil
		}(worker)
	}
	close(start)
	for worker := 0; worker < 6; worker++ {
		require.NoError(t, <-results)
	}
	_, err := m.MuteChat(&requests.MuteChat{ChatID: testPublicChatID, ThreadID: "thread", MutedType: MuteTillUnmuted})
	require.NoError(t, err)
	require.NoError(t, m.unmuteExpiredThreads(now.Add(time.Hour), &MessengerResponse{}))
	muted, err := m.isThreadMuted(testPublicChatID, "thread")
	require.NoError(t, err)
	require.True(t, muted)
}
