package protocol

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/db/appdatabase"
	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	"github.com/status-im/status-go/internal/protocol/sqlite"
)

type threadQueryCountingConnector struct {
	driver  driver.Driver
	path    string
	queries *[]string
}

func (c threadQueryCountingConnector) Connect(context.Context) (driver.Conn, error) {
	conn, err := c.driver.Open(c.path)
	if err != nil {
		return nil, err
	}
	queryer, ok := conn.(driver.QueryerContext)
	if !ok {
		_ = conn.Close()
		return nil, fmt.Errorf("thread query counting requires a driver.QueryerContext")
	}
	return &threadQueryCountingConn{Conn: conn, queryer: queryer, queries: c.queries}, nil
}

func (c threadQueryCountingConnector) Driver() driver.Driver {
	return c.driver
}

type threadQueryCountingConn struct {
	driver.Conn
	queryer driver.QueryerContext
	queries *[]string
}

func (c *threadQueryCountingConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	*c.queries = append(*c.queries, query)
	return c.queryer.QueryContext(ctx, query, args)
}

func openThreadQueryCountingDB(t *testing.T) (*sql.DB, *[]string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "threads.db")
	db, err := appdatabase.InitializeDB(path, "password", 1)
	require.NoError(t, err)
	require.NoError(t, sqlite.Migrate(db))
	sqlDriver := db.Driver()
	require.NoError(t, db.Close())

	queries := make([]string, 0)
	db = sql.OpenDB(threadQueryCountingConnector{driver: sqlDriver, path: path, queries: &queries})
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return db, &queries
}

func TestThreadsByIDs(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	p := newSQLitePersistence(db)

	peer, err := crypto.GenerateKey()
	require.NoError(t, err)
	publicChat := CreatePublicChat("bulk-public-chat", &testTimeSource{})
	privateChat := CreateOneToOneChat("bulk-one-to-one", &peer.PublicKey, &testTimeSource{})
	require.NoError(t, p.SaveChats([]*Chat{publicChat, privateChat}))

	publicID, privateID := "public-thread", "private-thread"
	stored := []*protobuf.BackedUpThread{
		{ChatId: publicChat.ID, ThreadId: publicID, ParentMessageId: "public-parent", Name: "Public", ReadMessagesAtClockValue: 12},
		{ChatId: privateChat.ID, ThreadId: privateID, ParentMessageId: "private-parent", Name: "Private", ReadMessagesAtClockValue: 34},
		{ChatId: publicChat.ID, ThreadId: "unrelated", ParentMessageId: "absent-parent", Name: "Unrelated"},
	}
	require.NoError(t, p.SaveBackedUpThreads(stored))

	var messages []*common.Message
	for _, storedThread := range stored[:2] {
		threadID := storedThread.ThreadId
		messages = append(messages, &common.Message{
			ID: storedThread.ParentMessageId, LocalChatID: storedThread.ChatId, From: testPK, Seen: true,
			ChatMessage: &protobuf.ChatMessage{Text: storedThread.Name, Clock: 1, Timestamp: 1},
		})
		for i, kind := range []string{"unread", "mentioned", "replied", "seen", "hidden", "deleted", "deleted-for-me"} {
			message := &common.Message{
				ID: threadID + "-" + kind, LocalChatID: storedThread.ChatId, From: testPK,
				Seen: kind == "seen", Mentioned: kind == "mentioned", Replied: kind == "replied",
				Deleted: kind == "deleted", DeletedForMe: kind == "deleted-for-me",
				ChatMessage: &protobuf.ChatMessage{ThreadId: &threadID, Text: kind, Clock: uint64(i + 2), Timestamp: uint64(i + 2)},
			}
			messages = append(messages, message)
		}
	}
	require.NoError(t, p.SaveMessages(messages))
	require.NoError(t, p.SetHideOnMessage(publicID+"-hidden"))
	require.NoError(t, p.SetHideOnMessage(privateID+"-hidden"))

	identities := []threadIdentity{
		{chatID: privateChat.ID, threadID: privateID},
		{chatID: publicChat.ID, threadID: publicID},
	}
	threads, err := p.threadsByIDs(identities)
	require.NoError(t, err)
	require.Len(t, threads, 2)
	require.Equal(t, privateID, threads[0].ThreadID)
	require.Equal(t, publicID, threads[1].ThreadID)
	require.EqualValues(t, 3, threads[0].UnviewedMessagesCount)
	require.EqualValues(t, 3, threads[0].UnviewedMentionsCount)
	require.EqualValues(t, 3, threads[1].UnviewedMessagesCount)
	require.EqualValues(t, 2, threads[1].UnviewedMentionsCount)

	for i, identity := range identities {
		single, err := p.ThreadByID(identity.chatID, identity.threadID)
		require.NoError(t, err)
		require.Equal(t, single, threads[i])
	}
	require.NoError(t, p.populateThreadsSummaries(threads, defaultThreadParticipantPreviewLimit))
	for i, identity := range identities {
		single, err := p.ThreadWithSummaryByID(identity.chatID, identity.threadID)
		require.NoError(t, err)
		require.Equal(t, single, threads[i])
	}

	duplicates, err := p.threadsByIDs([]threadIdentity{identities[0], identities[0]})
	require.NoError(t, err)
	require.Len(t, duplicates, 2)
	require.Equal(t, duplicates[0], duplicates[1])
	empty, err := p.threadsByIDs(nil)
	require.NoError(t, err)
	require.Empty(t, empty)

	for _, missing := range []threadIdentity{
		{chatID: publicChat.ID, threadID: privateID},
		{chatID: privateChat.ID, threadID: publicID},
		{chatID: publicChat.ID, threadID: "missing"},
		{chatID: "missing", threadID: publicID},
		{},
	} {
		threads, err := p.threadsByIDs([]threadIdentity{identities[0], missing})
		require.ErrorIs(t, err, common.ErrRecordNotFound)
		require.Nil(t, threads)
	}
}

func TestBackedUpThreadResponseQueryCount(t *testing.T) {
	db, queries := openThreadQueryCountingDB(t)
	p := newSQLitePersistence(db)
	m := &Messenger{persistence: p, featureFlags: common.FeatureFlags{Threads: true}}

	stored := make([]*protobuf.BackedUpThread, 1000)
	for i := range stored {
		id := fmt.Sprintf("thread-%04d", i)
		stored[i] = &protobuf.BackedUpThread{
			ChatId: fmt.Sprintf("chat-%d", i%3), ThreadId: id, ParentMessageId: "parent-" + id, Name: id,
			ReadMessagesAtClockValue: uint64(i + 1),
		}
	}
	require.NoError(t, p.SaveBackedUpThreads(stored))

	for _, count := range []int{0, 1, 100, 101, 1000} {
		t.Run(fmt.Sprintf("threads-%d", count), func(t *testing.T) {
			input := []*protobuf.BackedUpThread{nil}
			for _, thread := range stored[:count] {
				incoming := &protobuf.BackedUpThread{ChatId: thread.ChatId, ThreadId: thread.ThreadId, Name: "Incoming name"}
				input = append(input, incoming, nil, incoming)
			}
			input = append(input, stored[:count]...)
			response := &MessengerResponse{}
			*queries = nil
			require.NoError(t, m.addBackedUpThreadsToResponse(response, input))
			batches := (count + 99) / 100
			require.Len(t, *queries, batches*2)
			for i := 0; i < batches; i++ {
				require.Contains(t, (*queries)[2*i], "WITH requested(chat_id, thread_id)")
				require.Contains(t, (*queries)[2*i+1], "WITH candidates AS")
			}
			require.Len(t, response.Threads(), count)
			expected := make(map[threadIdentity]*protobuf.BackedUpThread, count)
			for _, thread := range stored[:count] {
				expected[threadIdentity{chatID: thread.ChatId, threadID: thread.ThreadId}] = thread
			}
			for _, thread := range response.Threads() {
				storedThread, ok := expected[threadIdentity{chatID: thread.ChatID, threadID: thread.ThreadID}]
				require.True(t, ok)
				require.Equal(t, storedThread.Name, thread.Name)
				require.Equal(t, storedThread.ParentMessageId, thread.ParentMessageID)
				require.Equal(t, storedThread.ReadMessagesAtClockValue, thread.ReadMessagesAtClockValue)
			}
		})
	}

	t.Run("feature-disabled", func(t *testing.T) {
		m.featureFlags.Threads = false
		*queries = nil
		response := &MessengerResponse{}
		require.NoError(t, m.addBackedUpThreadsToResponse(response, stored))
		require.Empty(t, *queries)
		require.Empty(t, response.Threads())
		m.featureFlags.Threads = true
	})

	t.Run("missing-pair", func(t *testing.T) {
		*queries = nil
		response := &MessengerResponse{}
		require.ErrorIs(t, m.addBackedUpThreadsToResponse(response, []*protobuf.BackedUpThread{
			stored[0], {ChatId: "wrong-chat", ThreadId: stored[1].ThreadId},
		}), common.ErrRecordNotFound)
		require.Len(t, *queries, 1)
		require.Empty(t, response.Threads())
	})

	t.Run("database-error", func(t *testing.T) {
		require.NoError(t, db.Close())
		err := m.addBackedUpThreadsToResponse(&MessengerResponse{}, stored[:1])
		require.ErrorContains(t, err, "database is closed")
	})
}

func TestThreadMetadataQueryUsesThreadIndex(t *testing.T) {
	db, queries := openThreadQueryCountingDB(t)
	p := newSQLitePersistence(db)
	identities := make([]threadIdentity, 100)
	stored := make([]*protobuf.BackedUpThread, len(identities))
	for i := range identities {
		id := fmt.Sprintf("thread-%03d", i)
		identities[i] = threadIdentity{chatID: "chat", threadID: id}
		stored[i] = &protobuf.BackedUpThread{ChatId: "chat", ThreadId: id}
	}
	require.NoError(t, p.SaveBackedUpThreads(stored))
	*queries = nil
	_, err := p.threadsByIDs(identities)
	require.NoError(t, err)
	require.Len(t, *queries, 1)

	args := make([]interface{}, 0, len(identities)*2+1)
	for _, identity := range identities {
		args = append(args, identity.chatID, identity.threadID)
	}
	args = append(args, ChatTypeOneToOne)
	rows, err := db.Query("EXPLAIN QUERY PLAN "+(*queries)[0], args...)
	require.NoError(t, err)
	defer rows.Close()
	var indexedThreadLookup bool
	for rows.Next() {
		var id, parent, unused int
		var detail string
		require.NoError(t, rows.Scan(&id, &parent, &unused, &detail))
		require.NotContains(t, detail, "SCAN threads")
		if strings.Contains(detail, "SEARCH threads USING INDEX") {
			indexedThreadLookup = true
		}
	}
	require.NoError(t, rows.Err())
	require.True(t, indexedThreadLookup, "bulk metadata query must use indexed thread lookups")
}
