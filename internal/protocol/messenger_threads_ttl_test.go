package protocol

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/protocol/migrations"
	"github.com/status-im/status-go/internal/protocol/protobuf"
)

type MessengerThreadsTTLSuite struct {
	MessengerBaseTestSuite
}

func TestMessengerThreadsTTLSuite(t *testing.T) {
	suite.Run(t, new(MessengerThreadsTTLSuite))
}

func (s *MessengerThreadsTTLSuite) SetupTest() {
	s.MessengerBaseTestSuite.SetupTest()
	s.m.featureFlags.Threads = true
}

func (s *MessengerThreadsTTLSuite) ttl() uint64 {
	return uint64(threadInactivityTTL.Milliseconds())
}

func (s *MessengerThreadsTTLSuite) setThreadActivity(chatID, threadID string, at uint64) {
	_, err := s.m.persistence.db.Exec(`UPDATE threads SET last_activity_at = ? WHERE chat_id = ? AND thread_id = ?`, at, chatID, threadID)
	s.Require().NoError(err)
}

func (s *MessengerThreadsTTLSuite) createThread(chat *Chat, parentID string) {
	parent := buildTestMessage(*chat)
	parent.ID = parentID
	parent.Text = "Parent " + parentID
	parent.ChatMessage.Text = parent.Text
	s.Require().NoError(s.m.SaveMessages([]*common.Message{parent}))
	_, err := s.m.CreateThread(chat.ID, parentID)
	s.Require().NoError(err)
}

func (s *MessengerThreadsTTLSuite) newChat(id string) *Chat {
	chat := CreateOneToOneChat(id, &s.m.identity.PublicKey, s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat))
	return chat
}

func findThread(threads []*Thread, threadID string) *Thread {
	for _, thread := range threads {
		if thread.ThreadID == threadID {
			return thread
		}
	}
	return nil
}

func (s *MessengerThreadsTTLSuite) TestNewThreadIsActive() {
	chat := s.newChat("ttl-new")
	before := s.m.getTimesource().GetCurrentTime()

	parent := buildTestMessage(*chat)
	parent.ID = "parent"
	s.Require().NoError(s.m.SaveMessages([]*common.Message{parent}))
	response, err := s.m.CreateThread(chat.ID, "parent")
	s.Require().NoError(err)
	after := s.m.getTimesource().GetCurrentTime()

	s.Require().Len(response.Threads(), 1)
	thread := response.Threads()[0]
	s.Require().GreaterOrEqual(thread.LastActivityAt, before)
	s.Require().LessOrEqual(thread.LastActivityAt, after)
	s.Require().False(thread.Archived)
}

func (s *MessengerThreadsTTLSuite) TestInactiveThreadIsArchivedOnEveryReadPath() {
	chat := s.newChat("ttl-archived")
	s.createThread(chat, "old")
	s.createThread(chat, "fresh")
	now := s.m.getTimesource().GetCurrentTime()
	s.setThreadActivity(chat.ID, "old", now-s.ttl()-1000)

	byChat, err := s.m.ThreadsByChatID(chat.ID)
	s.Require().NoError(err)
	s.Require().True(findThread(byChat, "old").Archived)
	s.Require().False(findThread(byChat, "fresh").Archived)

	byChats, err := s.m.ThreadsByChatIDs([]string{chat.ID})
	s.Require().NoError(err)
	s.Require().True(findThread(byChats, "old").Archived)
	s.Require().False(findThread(byChats, "fresh").Archived)

	summaries, err := s.m.ThreadSummariesByParentMessageIDs(chat.ID, []string{"old", "fresh"}, defaultThreadParticipantPreviewLimit)
	s.Require().NoError(err)
	s.Require().True(findThread(summaries, "old").Archived)
	s.Require().False(findThread(summaries, "fresh").Archived)

	thread, err := s.m.persistence.ThreadByID(chat.ID, "old")
	s.Require().NoError(err)
	s.Require().True(thread.Archived)

	thread, err = s.m.persistence.ThreadWithSummaryByID(chat.ID, "old")
	s.Require().NoError(err)
	s.Require().True(thread.Archived)
	s.Require().Equal(now-s.ttl()-1000, thread.LastActivityAt)
}

func (s *MessengerThreadsTTLSuite) TestThreadWithoutActivityIsArchivedFromCreationTime() {
	chat := s.newChat("ttl-no-activity")
	createdAt := s.m.getTimesource().GetCurrentTime()
	now := createdAt
	originalClock := s.m.persistence.threadClock
	s.m.persistence.threadClock = func() uint64 { return now }
	defer func() { s.m.persistence.threadClock = originalClock }()

	s.createThread(chat, "parent")

	for _, elapsed := range []uint64{0, s.ttl(), s.ttl() + 1} {
		now = createdAt + elapsed
		// Repeated metadata must not restart the inactivity period.
		s.Require().NoError(s.m.persistence.UpsertThread("parent", chat.ID, "parent", ""))

		byChat, err := s.m.ThreadsByChatID(chat.ID)
		s.Require().NoError(err)
		byChats, err := s.m.ThreadsByChatIDs([]string{chat.ID})
		s.Require().NoError(err)
		summaries, err := s.m.ThreadSummariesByParentMessageIDs(chat.ID, []string{"parent"}, defaultThreadParticipantPreviewLimit)
		s.Require().NoError(err)
		thread, err := s.m.persistence.ThreadByID(chat.ID, "parent")
		s.Require().NoError(err)
		summary, err := s.m.persistence.ThreadWithSummaryByID(chat.ID, "parent")
		s.Require().NoError(err)

		for _, result := range []*Thread{findThread(byChat, "parent"), findThread(byChats, "parent"), findThread(summaries, "parent"), thread, summary} {
			s.Require().NotNil(result)
			s.Require().Equal(createdAt, result.LastActivityAt)
			s.Require().Equal(elapsed > s.ttl(), result.Archived)
		}
	}

	response, err := s.m.sweepArchivedThreads(createdAt+s.ttl(), now)
	s.Require().NoError(err)
	s.Require().Len(response.Threads(), 1)
	s.Require().Equal("parent", response.Threads()[0].ThreadID)
	s.Require().True(response.Threads()[0].Archived)
}

func (s *MessengerThreadsTTLSuite) TestSentReplyUnarchivesThread() {
	receiver := s.newMessenger()
	chat := CreateOneToOneChat("ttl-send", &receiver.identity.PublicKey, s.m.getTimesource())
	s.Require().NoError(s.m.SaveChat(chat))
	_, err := s.m.Join(chat)
	s.Require().NoError(err)
	s.createThread(chat, "parent")
	s.setThreadActivity(chat.ID, "parent", 1)

	threadID := "parent"
	reply := buildTestMessage(*chat)
	reply.Text = "Reply"
	reply.ChatMessage.Text = "Reply"
	reply.ChatMessage.ThreadId = &threadID
	response, err := s.m.SendChatMessage(context.Background(), reply)
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 1)

	thread := findThread(response.Threads(), threadID)
	s.Require().NotNil(thread)
	s.Require().False(thread.Archived)
	s.Require().Greater(thread.LastActivityAt, uint64(1))
}

func (s *MessengerThreadsTTLSuite) TestOnlyNewMessagesCountAsActivity() {
	chat := s.newChat("ttl-touch")
	s.createThread(chat, "parent")
	s.setThreadActivity(chat.ID, "parent", 1)

	threadID := "parent"
	reply := buildTestMessage(*chat)
	reply.ID = "reply"
	reply.Timestamp = s.m.getTimesource().GetCurrentTime()
	reply.ChatMessage.ThreadId = &threadID
	s.Require().NoError(s.m.SaveMessages([]*common.Message{reply}))

	isNew := func(message *common.Message) bool { return message.New }
	s.m.touchThreadActivity([]*common.Message{reply}, isNew)
	thread, err := s.m.persistence.ThreadByID(chat.ID, threadID)
	s.Require().NoError(err)
	s.Require().True(thread.Archived)

	reply.New = true
	s.m.touchThreadActivity([]*common.Message{reply, reply}, isNew)
	thread, err = s.m.persistence.ThreadByID(chat.ID, threadID)
	s.Require().NoError(err)
	s.Require().False(thread.Archived)

	s.m.featureFlags.Threads = false
	s.setThreadActivity(chat.ID, threadID, 1)
	s.m.touchThreadActivity([]*common.Message{reply}, isNew)
	thread, err = s.m.persistence.ThreadByID(chat.ID, threadID)
	s.Require().NoError(err)
	s.Require().Equal(uint64(1), thread.LastActivityAt)
}

func (s *MessengerThreadsTTLSuite) TestReceivedReplyUnarchivesThread() {
	receiver := s.m
	sender := s.newMessenger()
	sender.featureFlags.Threads = true

	receiverChat := CreateOneToOneChat("ttl-receive", &sender.identity.PublicKey, receiver.getTimesource())
	s.Require().NoError(receiver.SaveChat(receiverChat))
	_, err := receiver.Join(receiverChat)
	s.Require().NoError(err)

	senderChat := CreateOneToOneChat("ttl-receive", &receiver.identity.PublicKey, sender.getTimesource())
	s.Require().NoError(sender.SaveChat(senderChat))
	_, err = sender.Join(senderChat)
	s.Require().NoError(err)

	parent := buildTestMessage(*senderChat)
	parent.Text = "Parent"
	parent.ChatMessage.Text = "Parent"
	parentResponse, err := sender.SendChatMessage(context.Background(), parent)
	s.Require().NoError(err)
	threadID := parentResponse.Messages()[0].ID
	_, err = sender.CreateThread(senderChat.ID, threadID)
	s.Require().NoError(err)

	sendReply := func(text string) {
		reply := buildTestMessage(*senderChat)
		reply.Text = text
		reply.ChatMessage.Text = text
		reply.ChatMessage.ThreadId = &threadID
		reply.ResponseTo = threadID
		_, err := sender.SendChatMessage(context.Background(), reply)
		s.Require().NoError(err)
	}
	waitForThread := func(text string) *Thread {
		var thread *Thread
		_, err := WaitOnMessengerResponse(receiver, func(response *MessengerResponse) bool {
			for _, msg := range response.Messages() {
				if msg.Text == text {
					thread = findThread(response.Threads(), threadID)
					return thread != nil
				}
			}
			return false
		}, "thread reply not received")
		s.Require().NoError(err)
		return thread
	}

	sendReply("First reply")
	s.Require().False(waitForThread("First reply").Archived)

	s.setThreadActivity(receiverChat.ID, threadID, 1)
	thread, err := receiver.persistence.ThreadByID(receiverChat.ID, threadID)
	s.Require().NoError(err)
	s.Require().True(thread.Archived)

	sendReply("Second reply")
	s.Require().False(waitForThread("Second reply").Archived)
}

func (s *MessengerThreadsTTLSuite) TestSweepReportsOnlyNewlyArchivedThreads() {
	chat := s.newChat("ttl-sweep")
	s.createThread(chat, "crossed")
	s.createThread(chat, "already-archived")
	s.createThread(chat, "active")
	s.createThread(chat, "no-activity")

	now := s.m.getTimesource().GetCurrentTime()
	lastSweepAt := now - 10*60*1000
	s.setThreadActivity(chat.ID, "crossed", now-s.ttl()-60*1000)
	s.setThreadActivity(chat.ID, "already-archived", lastSweepAt-s.ttl()-1)
	s.setThreadActivity(chat.ID, "no-activity", 0)

	response, err := s.m.sweepArchivedThreads(lastSweepAt, now)
	s.Require().NoError(err)
	s.Require().Len(response.Threads(), 1)
	thread := response.Threads()[0]
	s.Require().Equal("crossed", thread.ThreadID)
	s.Require().True(thread.Archived)
	s.Require().NotNil(thread.LastMessage)

	response, err = s.m.sweepArchivedThreads(now, now+1000)
	s.Require().NoError(err)
	s.Require().True(response.IsEmpty())

	response, err = s.m.sweepArchivedThreads(now, now)
	s.Require().NoError(err)
	s.Require().True(response.IsEmpty())
}

func (s *MessengerThreadsTTLSuite) threadReply(chat *Chat, id, threadID string, timestamp uint64) *common.Message {
	reply := buildTestMessage(*chat)
	reply.ID = id
	reply.Timestamp = timestamp
	reply.New = true
	reply.ChatMessage.ThreadId = &threadID
	return reply
}

func (s *MessengerThreadsTTLSuite) TestOldMessageDoesNotUnarchiveThread() {
	chat := s.newChat("ttl-old-message")
	s.createThread(chat, "parent")
	now := s.m.getTimesource().GetCurrentTime()
	day := uint64(24 * 60 * 60 * 1000)
	s.setThreadActivity(chat.ID, "parent", now-10*day)
	isNew := func(message *common.Message) bool { return message.New }

	old := s.threadReply(chat, "old", "parent", now-9*day)
	s.m.touchThreadActivity([]*common.Message{old}, isNew)
	thread, err := s.m.persistence.ThreadByID(chat.ID, "parent")
	s.Require().NoError(err)
	s.Require().True(thread.Archived)
	s.Require().Equal(now-9*day, thread.LastActivityAt)

	// The latest message of a batch wins, whatever its position.
	recent := s.threadReply(chat, "recent", "parent", now-day)
	older := s.threadReply(chat, "older", "parent", now-8*day)
	s.m.touchThreadActivity([]*common.Message{recent, older}, isNew)
	thread, err = s.m.persistence.ThreadByID(chat.ID, "parent")
	s.Require().NoError(err)
	s.Require().False(thread.Archived)
	s.Require().Equal(now-day, thread.LastActivityAt)
}

func (s *MessengerThreadsTTLSuite) TestFutureMessageTimestampIsCappedAtNow() {
	chat := s.newChat("ttl-future")
	s.createThread(chat, "parent")
	s.setThreadActivity(chat.ID, "parent", 1)

	before := s.m.getTimesource().GetCurrentTime()
	future := s.threadReply(chat, "future", "parent", before+365*24*60*60*1000)
	s.m.touchThreadActivity([]*common.Message{future}, func(*common.Message) bool { return true })
	after := s.m.getTimesource().GetCurrentTime()

	thread, err := s.m.persistence.ThreadByID(chat.ID, "parent")
	s.Require().NoError(err)
	s.Require().GreaterOrEqual(thread.LastActivityAt, before)
	s.Require().LessOrEqual(thread.LastActivityAt, after)
}

func (s *MessengerThreadsTTLSuite) TestThreadCreatedFromOldMessageIsArchived() {
	chat := s.newChat("ttl-old-create")
	now := s.m.getTimesource().GetCurrentTime()
	day := uint64(24 * 60 * 60 * 1000)

	old := s.threadReply(chat, "old-reply", "old-thread", now-9*day)
	old.ThreadMetadataCreationAuthorized = true
	recent := s.threadReply(chat, "recent-reply", "recent-thread", now-day)
	recent.ThreadMetadataCreationAuthorized = true
	s.Require().NoError(s.m.SaveMessages([]*common.Message{old, recent}))

	thread, err := s.m.persistence.ThreadByID(chat.ID, "old-thread")
	s.Require().NoError(err)
	s.Require().Equal(now-9*day, thread.LastActivityAt)
	s.Require().True(thread.Archived)

	thread, err = s.m.persistence.ThreadByID(chat.ID, "recent-thread")
	s.Require().NoError(err)
	s.Require().Equal(now-day, thread.LastActivityAt)
	s.Require().False(thread.Archived)

	// Received thread metadata and synced read state also use the message time.
	s.Require().NoError(s.m.persistence.UpsertThreadAt("metadata-thread", chat.ID, "metadata-thread", "", now-9*day))
	s.Require().NoError(s.m.persistence.UpsertPendingCreatorThreadAt("read-thread", chat.ID, "read-thread", "", now-9*day))
	for _, id := range []string{"metadata-thread", "read-thread"} {
		thread, err = s.m.persistence.ThreadByID(chat.ID, id)
		s.Require().NoError(err)
		s.Require().True(thread.Archived, id)
	}
}

func (s *MessengerThreadsTTLSuite) TestTouchThreadActivityNeverMovesBackwards() {
	chat := s.newChat("ttl-monotonic")
	s.createThread(chat, "parent")
	s.setThreadActivity(chat.ID, "parent", 2000)

	s.Require().NoError(s.m.persistence.TouchThreadActivity(chat.ID, "parent", 1000))
	thread, err := s.m.persistence.ThreadByID(chat.ID, "parent")
	s.Require().NoError(err)
	s.Require().Equal(uint64(2000), thread.LastActivityAt)

	s.Require().NoError(s.m.persistence.TouchThreadActivity(chat.ID, "parent", 3000))
	thread, err = s.m.persistence.ThreadByID(chat.ID, "parent")
	s.Require().NoError(err)
	s.Require().Equal(uint64(3000), thread.LastActivityAt)
}

func (s *MessengerThreadsTTLSuite) TestThreadsInactiveBetweenBounds() {
	chat := s.newChat("ttl-window")
	for _, id := range []string{"zero", "below", "from", "inside", "to"} {
		s.createThread(chat, id)
	}
	s.setThreadActivity(chat.ID, "zero", 0)
	s.setThreadActivity(chat.ID, "below", 99)
	s.setThreadActivity(chat.ID, "from", 100)
	s.setThreadActivity(chat.ID, "inside", 150)
	s.setThreadActivity(chat.ID, "to", 200)

	identities, err := s.m.persistence.ThreadsInactiveBetween(100, 200)
	s.Require().NoError(err)
	s.Require().ElementsMatch([]threadIdentity{
		{chatID: chat.ID, threadID: "from"},
		{chatID: chat.ID, threadID: "inside"},
	}, identities)

	identities, err = s.m.persistence.ThreadsInactiveBetween(0, 100)
	s.Require().NoError(err)
	s.Require().ElementsMatch([]threadIdentity{
		{chatID: chat.ID, threadID: "zero"},
		{chatID: chat.ID, threadID: "below"},
	}, identities)
}

func (s *MessengerThreadsTTLSuite) TestRestoredThreadActivityComesFromMessages() {
	chat := s.newChat("ttl-restore")

	parent := buildTestMessage(*chat)
	parent.ID = "with-messages"
	parent.Timestamp = 1000
	threadID := "with-messages"
	reply := buildTestMessage(*chat)
	reply.ID = "reply"
	reply.Timestamp = 5000
	reply.ChatMessage.ThreadId = &threadID
	s.Require().NoError(s.m.SaveMessages([]*common.Message{parent, reply}))

	before := s.m.getTimesource().GetCurrentTime()
	s.Require().NoError(s.m.persistence.SaveBackedUpThreads([]*protobuf.BackedUpThread{
		{ThreadId: threadID, ChatId: chat.ID, ParentMessageId: threadID, Name: "Restored"},
		{ThreadId: "without-messages", ChatId: chat.ID, ParentMessageId: "without-messages", Name: "Empty"},
	}))

	thread, err := s.m.persistence.ThreadByID(chat.ID, threadID)
	s.Require().NoError(err)
	s.Require().Equal(uint64(5000), thread.LastActivityAt)
	s.Require().True(thread.Archived)

	thread, err = s.m.persistence.ThreadByID(chat.ID, "without-messages")
	s.Require().NoError(err)
	s.Require().GreaterOrEqual(thread.LastActivityAt, before)
	s.Require().False(thread.Archived)

	// Restoring an existing thread keeps its activity.
	s.setThreadActivity(chat.ID, "without-messages", 42)
	s.Require().NoError(s.m.persistence.SaveBackedUpThreads([]*protobuf.BackedUpThread{
		{ThreadId: "without-messages", ChatId: chat.ID, ParentMessageId: "without-messages", Name: "Empty"},
	}))
	thread, err = s.m.persistence.ThreadByID(chat.ID, "without-messages")
	s.Require().NoError(err)
	s.Require().Equal(uint64(42), thread.LastActivityAt)
}

func TestIsThreadArchived(t *testing.T) {
	ttl := uint64(threadInactivityTTL.Milliseconds())
	now := ttl * 3
	if !isThreadArchived(0, now) {
		t.Fatal("zero activity time must not exempt a thread from archiving")
	}
	if !isThreadArchived(now-ttl-1, now) {
		t.Fatal("thread older than the TTL must be archived")
	}
	if isThreadArchived(now-ttl, now) {
		t.Fatal("thread exactly at the TTL must not be archived")
	}
	if threadArchiveCutoff(ttl-1) != 0 {
		t.Fatal("cutoff must not underflow")
	}
	if isThreadArchived(0, ttl) {
		t.Fatal("thread exactly at the TTL must not be archived even at epoch zero")
	}
}

func TestThreadActivityMigration(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE threads (thread_id TEXT PRIMARY KEY, chat_id TEXT, parent_message_id TEXT);
		CREATE TABLE user_messages (
			id TEXT, local_chat_id TEXT, thread_id TEXT, timestamp INTEGER,
			deleted BOOLEAN DEFAULT FALSE, deleted_for_me BOOLEAN DEFAULT FALSE
		);
		INSERT INTO threads VALUES
			('empty', 'chat', 'missing'),
			('zero', 'chat', 'zero-parent'),
			('deleted', 'chat', 'deleted-parent'),
			('future-parent', 'chat', 'future-parent'),
			('future-reply', 'chat', 'reply-parent'),
			('active', 'chat', 'parent');
		INSERT INTO user_messages (id, local_chat_id, thread_id, timestamp, deleted) VALUES
			('zero-parent', 'chat', NULL, 0, FALSE),
			('deleted-parent', 'chat', NULL, 1000, TRUE),
			('parent', 'chat', NULL, 1000, FALSE),
			('reply', 'chat', 'active', 2000, FALSE);
	`)
	require.NoError(t, err)
	future := time.Now().Add(365 * 24 * time.Hour).UnixMilli()
	_, err = db.Exec(`INSERT INTO user_messages (id, local_chat_id, thread_id, timestamp) VALUES
		('future-parent', 'chat', NULL, ?),
		('reply-parent', 'chat', NULL, 1000),
		('future-message', 'chat', 'future-reply', ?)`, future, future)
	require.NoError(t, err)

	migration, err := migrations.Asset("1793045254_add_thread_last_activity.up.sql")
	require.NoError(t, err)
	before := uint64(time.Now().Unix()) * 1000
	_, err = db.Exec(string(migration))
	require.NoError(t, err)
	after := uint64(time.Now().UnixMilli())

	for _, id := range []string{"empty", "zero", "deleted", "future-parent", "future-reply", "active"} {
		var activityAt uint64
		require.NoError(t, db.QueryRow(`SELECT last_activity_at FROM threads WHERE thread_id = ?`, id).Scan(&activityAt))
		if id == "active" {
			require.Equal(t, uint64(2000), activityAt)
			continue
		}
		require.GreaterOrEqual(t, activityAt, before)
		require.LessOrEqual(t, activityAt, after)
		require.False(t, isThreadArchived(activityAt, activityAt+uint64(threadInactivityTTL.Milliseconds())))
		require.True(t, isThreadArchived(activityAt, activityAt+uint64(threadInactivityTTL.Milliseconds())+1))
		require.True(t, isThreadArchived(activityAt, after+uint64(threadInactivityTTL.Milliseconds())+1))
	}
}
