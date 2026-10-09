package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/db/appdatabase"
	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	protocolsqlite "github.com/status-im/status-go/internal/protocol/sqlite"
	"github.com/status-im/status-go/internal/testutils"
)

func seedThreadSummaryCache(t *testing.T, p *sqlitePersistence) {
	t.Helper()
	require.NoError(t, p.SaveChat(Chat{ID: testPublicChatID, ChatType: ChatTypeCommunityChat}))
	threadID := "thread"
	require.NoError(t, p.SaveMessages([]*common.Message{
		{ID: "parent", LocalChatID: testPublicChatID, From: "creator",
			ChatMessage: &protobuf.ChatMessage{Text: "Parent", Clock: 1}},
		{ID: "reply", LocalChatID: testPublicChatID, From: "author",
			ChatMessage: &protobuf.ChatMessage{Text: "Reply", Clock: 2, ThreadId: &threadID}},
	}))
	require.NoError(t, p.UpsertThread(threadID, testPublicChatID, "parent", "Thread"))
}

func assertCachedThreadSummaries(t *testing.T, p *sqlitePersistence, previewLimit int) []*Thread {
	t.Helper()
	actual, err := p.ThreadsWithSummariesByParentMessageIDs(testPublicChatID, []string{"parent"}, previewLimit)
	require.NoError(t, err)
	uncached := *p
	uncached.threadSummaryCache = nil
	expected, err := uncached.ThreadsWithSummariesByParentMessageIDs(testPublicChatID, []string{"parent"}, previewLimit)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
	return actual
}

func TestThreadSummaryCacheInvalidatesLocalWrites(t *testing.T) {
	for _, query := range []string{
		"UPDATE user_messages SET text = 'Edited' WHERE id = 'reply'",
		"UPDATE user_messages SET text = 'Edited parent', clock_value = 3 WHERE id = 'parent'",
		"UPDATE user_messages SET source = 'new-author' WHERE id = 'reply'",
		"UPDATE user_messages SET deleted = 1 WHERE id = 'reply'",
		"UPDATE user_messages SET deleted_for_me = 1 WHERE id = 'reply'",
		"UPDATE user_messages SET hide = 1, seen = 1 WHERE id = 'reply'",
		"UPDATE user_messages SET seen = 1 WHERE id = 'reply'",
		"UPDATE user_messages SET mentioned = 1 WHERE id = 'reply'",
		"UPDATE user_messages SET replied = 1 WHERE id = 'reply'",
		"DELETE FROM user_messages WHERE id = 'reply'",
		"DELETE FROM user_messages WHERE id = 'parent'",
		"UPDATE threads SET name = 'Renamed' WHERE thread_id = 'thread'",
		"DELETE FROM threads WHERE thread_id = 'thread'",
	} {
		t.Run(query, func(t *testing.T) {
			db, err := openTestDB()
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, db.Close()) })
			p := newSQLitePersistence(db)
			seedThreadSummaryCache(t, p)
			assertCachedThreadSummaries(t, p, 6)
			assertCachedThreadSummaries(t, p, 6)
			_, err = db.Exec(query)
			require.NoError(t, err)
			assertCachedThreadSummaries(t, p, 6)
		})
	}
}

func TestThreadSummaryCacheInsertionAndReadWatermark(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	p := newSQLitePersistence(db)
	seedThreadSummaryCache(t, p)
	assertCachedThreadSummaries(t, p, 6)
	threadID := "thread"
	require.NoError(t, p.SaveMessages([]*common.Message{{ID: "new-reply", LocalChatID: testPublicChatID, From: "another-author",
		ChatMessage: &protobuf.ChatMessage{Text: "New reply", Clock: 3, ThreadId: &threadID}}}))
	threads := assertCachedThreadSummaries(t, p, 6)
	require.Equal(t, uint(3), threads[0].MessagesCount)
	require.Equal(t, "New reply", threads[0].LastMessage.Text)
	_, _, err = p.MarkThreadRead(testPublicChatID, threadID, 3)
	require.NoError(t, err)
	threads = assertCachedThreadSummaries(t, p, 6)
	require.Zero(t, threads[0].UnviewedMessagesCount)
}

func TestThreadSummaryCacheCopiesAndPreviewLimits(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	p := newSQLitePersistence(db)
	seedThreadSummaryCache(t, p)
	threads := assertCachedThreadSummaries(t, p, 6)
	require.Len(t, p.threadSummaryCache.entries, 1)
	threads[0].LastMessage.Text = "Caller mutation"
	threads[0].ParticipantsPreviewIDs[0] = "Caller mutation"
	threads[0].MessagesCount = 99
	threads = assertCachedThreadSummaries(t, p, 6)
	require.Equal(t, "Reply", threads[0].LastMessage.Text)
	for _, limit := range []int{0, 1, 6, 1, 0, 6} {
		threads = assertCachedThreadSummaries(t, p, limit)
		require.Equal(t, uint(2), threads[0].ParticipantsCount)
		require.Len(t, threads[0].ParticipantsPreviewIDs, min(limit, 2))
	}
}

func TestThreadSummaryCacheExternalConnectionAndConcurrentReads(t *testing.T) {
	db, teardown, err := testutils.SetupTestSQLDB(appdatabase.DbInitializer{}, "thread-summary-cache")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, teardown()) })
	require.NoError(t, protocolsqlite.Migrate(db))
	db.SetMaxOpenConns(2)
	p := newSQLitePersistence(db)
	seedThreadSummaryCache(t, p)
	assertCachedThreadSummaries(t, p, 6)
	reader, err := db.Conn(context.Background())
	require.NoError(t, err)
	before, err := readThreadSummaryStamp(reader)
	require.NoError(t, err)
	// The reader is pinned, so this write uses a different pooled connection.
	_, err = db.Exec("UPDATE user_messages SET text = 'External edit' WHERE id = 'reply'")
	require.NoError(t, err)
	after, err := readThreadSummaryStamp(reader)
	require.NoError(t, err)
	require.Equal(t, before.connection, after.connection)
	require.Equal(t, before.totalChanges, after.totalChanges)
	require.NotEqual(t, before.dataVersion, after.dataVersion)
	require.False(t, p.threadSummaryCache.load(after, []*Thread{{ThreadID: "thread"}}, 6))
	require.NoError(t, reader.Close())
	threads := assertCachedThreadSummaries(t, p, 6)
	require.Equal(t, "External edit", threads[0].LastMessage.Text)
	var wg sync.WaitGroup
	errors := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				threads, err := p.ThreadsWithSummariesByParentMessageIDs(testPublicChatID, []string{"parent"}, 6)
				if err == nil && (len(threads) != 1 || threads[0].LastMessage.Text != "External edit") {
					err = fmt.Errorf("unexpected cached summary")
				}
				if err != nil {
					errors <- err
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}
}

func TestThreadSummaryCacheSchemaAndCapacity(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	p := newSQLitePersistence(db)
	seedThreadSummaryCache(t, p)
	assertCachedThreadSummaries(t, p, 6)
	conn, err := db.Conn(context.Background())
	require.NoError(t, err)
	before, err := readThreadSummaryStamp(conn)
	require.NoError(t, err)
	_, err = conn.ExecContext(context.Background(), "CREATE TABLE unrelated_cache_test (id TEXT)")
	require.NoError(t, err)
	after, err := readThreadSummaryStamp(conn)
	require.NoError(t, err)
	require.NotEqual(t, before.schemaVersion, after.schemaVersion)
	require.False(t, p.threadSummaryCache.load(after, []*Thread{{ThreadID: "thread"}}, 6))
	require.NoError(t, conn.Close())
	for i := 0; i < threadSummaryCacheCapacity+1; i++ {
		p.threadSummaryCache.store(after, []*Thread{{ThreadID: fmt.Sprint(i)}}, 6)
	}
	require.LessOrEqual(t, len(p.threadSummaryCache.entries), threadSummaryCacheCapacity)
}

func TestThreadSummaryParticipantsExcludeEmptySources(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	p := newSQLitePersistence(db)
	seedThreadSummaryCache(t, p)
	for _, query := range []string{
		"UPDATE user_messages SET source = '' WHERE id = 'parent'",
		"UPDATE user_messages SET source = '' WHERE id = 'reply'",
	} {
		_, err = db.Exec(query)
		require.NoError(t, err)
		for _, limit := range []int{0, 1, 6} {
			threads := assertCachedThreadSummaries(t, p, limit)
			if threads[0].LastMessage.From == "" {
				require.Zero(t, threads[0].ParticipantsCount)
			} else {
				require.Equal(t, uint(1), threads[0].ParticipantsCount)
			}
		}
	}
}

func TestMessagePageHasThreadHint(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	p := newSQLitePersistence(db)
	seedThreadSummaryCache(t, p)
	_, err = db.Exec("UPDATE user_messages SET source = ?", testPK)
	require.NoError(t, err)
	require.NoError(t, p.SaveMessages([]*common.Message{{ID: "plain", LocalChatID: testPublicChatID,
		ChatMessage: &protobuf.ChatMessage{Text: "No thread", Clock: 3}}}))
	for _, enabled := range []bool{false, true} {
		messages, _, err := p.MessageByChatID(testPublicChatID, "", "", 20, enabled)
		require.NoError(t, err)
		for _, message := range messages {
			if !enabled {
				require.Nil(t, message.HasThread)
				continue
			}
			require.NotNil(t, message.HasThread)
			require.Equal(t, message.ID == "parent", *message.HasThread)
			encoded, err := json.Marshal(message)
			require.NoError(t, err)
			var decoded map[string]interface{}
			require.NoError(t, json.Unmarshal(encoded, &decoded))
			require.Equal(t, *message.HasThread, decoded["hasThread"])
		}
	}
	threadMessages, _, err := p.MessageByChatID(testPublicChatID, "thread", "", 20, true)
	require.NoError(t, err)
	require.Nil(t, threadMessages[0].HasThread)
	_, err = db.Exec("DELETE FROM threads")
	require.NoError(t, err)
	messages, _, err := p.MessageByChatID(testPublicChatID, "", "", 20, true)
	require.NoError(t, err)
	for _, message := range messages {
		require.False(t, *message.HasThread)
	}
}
