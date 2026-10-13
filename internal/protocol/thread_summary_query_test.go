package protocol

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/protocol/protobuf"
)

func TestThreadSummaryUnreadCounts(t *testing.T) {
	for _, chatType := range []ChatType{ChatTypeCommunityChat, ChatTypeOneToOne} {
		t.Run(fmt.Sprint(chatType), func(t *testing.T) {
			db, err := openTestDB()
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, db.Close()) })
			p := newSQLitePersistence(db)
			require.NoError(t, p.SaveChat(Chat{ID: testPK, ChatType: chatType}))
			for _, id := range []string{"parent", "empty-parent", "outside-page"} {
				require.NoError(t, p.SaveMessages([]*common.Message{{
					ID: id, LocalChatID: testPK, From: "creator",
					ChatMessage: &protobuf.ChatMessage{Text: "Parent", Clock: 1},
				}}))
				require.NoError(t, p.UpsertThread(id, testPK, id, id))
			}
			threadID := "parent"
			for i, state := range []string{"plain", "mentioned", "replied", "seen", "hidden", "deleted", "deleted-for-me"} {
				message := &common.Message{
					ID: state, LocalChatID: testPK, From: "author",
					Seen: state == "seen", Mentioned: state == "mentioned", Replied: state == "replied",
					Deleted: state == "deleted", DeletedForMe: state == "deleted-for-me",
					ChatMessage: &protobuf.ChatMessage{Text: state, ThreadId: &threadID, Clock: uint64(i + 2)},
				}
				require.NoError(t, p.SaveMessages([]*common.Message{message}))
			}
			_, err = db.Exec("UPDATE user_messages SET hide = 1 WHERE id = 'hidden'")
			require.NoError(t, err)
			threads, err := p.ThreadsWithSummariesByParentMessageIDs(testPK, []string{"parent", "empty-parent"}, 0)
			require.NoError(t, err)
			require.Len(t, threads, 2)
			for _, thread := range threads {
				if thread.ParentMessageID == "empty-parent" {
					require.Zero(t, thread.UnviewedMessagesCount)
					require.Zero(t, thread.UnviewedMentionsCount)
					continue
				}
				require.Equal(t, "parent", thread.ParentMessageID)
				require.Equal(t, uint(3), thread.UnviewedMessagesCount)
				expectedMentions := uint(2)
				if chatType == ChatTypeOneToOne {
					expectedMentions = 3
				}
				require.Equal(t, expectedMentions, thread.UnviewedMentionsCount)
				reference, err := p.ThreadByID(testPK, thread.ThreadID)
				require.NoError(t, err)
				require.Equal(t, reference.UnviewedMessagesCount, thread.UnviewedMessagesCount)
				require.Equal(t, reference.UnviewedMentionsCount, thread.UnviewedMentionsCount)
			}
		})
	}
}

func TestThreadSummaryLatestMessageOrdering(t *testing.T) {
	for _, scenario := range []struct {
		name         string
		parentClock  uint64
		replyClock   uint64
		replyID      string
		expectedText string
	}{
		{"numeric_clock", 9, 10, "reply", "Reply"},
		{"parent_latest", 10, 9, "reply", "Parent"},
		{"equal_clock_reply_wins", 10, 10, "z-reply", "Reply"},
		{"equal_clock_parent_wins", 10, 10, "0-reply", "Parent"},
		{"large_clock", 9223372036854775806, 9223372036854775807, "reply", "Reply"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			db, err := openTestDB()
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, db.Close()) })
			p := newSQLitePersistence(db)
			require.NoError(t, p.SaveChat(Chat{ID: testPublicChatID, ChatType: ChatTypeCommunityChat}))
			threadID := "thread"
			messages := []*common.Message{
				{ID: "a-parent", LocalChatID: testPublicChatID, From: "creator",
					ChatMessage: &protobuf.ChatMessage{Text: "Parent", Clock: scenario.parentClock, Timestamp: 100}},
				{ID: scenario.replyID, LocalChatID: testPublicChatID, From: "participant",
					ChatMessage: &protobuf.ChatMessage{Text: "Reply", Clock: scenario.replyClock, Timestamp: 200, ThreadId: &threadID}},
				{ID: "deleted-reply", LocalChatID: testPublicChatID, From: "deleted", Deleted: true,
					ChatMessage: &protobuf.ChatMessage{Text: "Deleted", Clock: scenario.replyClock + 1, ThreadId: &threadID}},
			}
			// SQLite stores clock values as signed 64-bit integers.
			if scenario.replyClock == 9223372036854775807 {
				messages = messages[:2]
			}
			require.NoError(t, p.SaveMessages(messages))
			require.NoError(t, p.UpsertThread(threadID, testPublicChatID, "a-parent", "Thread"))
			for _, previewLimit := range []int{0, 1, 6} {
				threads, err := p.ThreadsWithSummariesByParentMessageIDs(testPublicChatID, []string{"a-parent"}, previewLimit)
				require.NoError(t, err)
				require.Len(t, threads, 1)
				thread := threads[0]
				require.Equal(t, uint(2), thread.MessagesCount)
				require.Equal(t, uint(2), thread.ParticipantsCount)
				require.Equal(t, scenario.expectedText, thread.LastMessage.Text)
				if scenario.expectedText == "Parent" {
					require.Equal(t, "creator", thread.LastMessage.From)
					require.Equal(t, uint64(100), thread.LastMessage.Timestamp)
				} else {
					require.Equal(t, "participant", thread.LastMessage.From)
					require.Equal(t, uint64(200), thread.LastMessage.Timestamp)
				}
			}
		})
	}
}
