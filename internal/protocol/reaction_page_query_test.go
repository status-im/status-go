package protocol

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/protocol/protobuf"
)

func TestReactionsByMessageIDsMatchesMessagePages(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	p := newSQLitePersistence(db)
	threadID := "thread"
	for i := 0; i < 8; i++ {
		id := fmt.Sprintf("message-%d", i)
		message := &common.Message{ID: id, LocalChatID: testPublicChatID, From: testPK,
			ChatMessage: &protobuf.ChatMessage{Text: id, Clock: uint64(i + 1)}}
		if i >= 4 {
			message.ThreadId = &threadID
		}
		require.NoError(t, p.SaveMessages([]*common.Message{message}))
		for _, retracted := range []bool{false, true} {
			require.NoError(t, p.SaveEmojiReaction(&EmojiReaction{
				From: fmt.Sprint(retracted), LocalChatID: testPublicChatID,
				EmojiReaction: &protobuf.EmojiReaction{MessageId: id, ChatId: testPublicChatID,
					Clock: uint64(i + 1), Emoji: "1f600", Retracted: retracted},
			}))
		}
	}
	for _, threadsEnabled := range []bool{false, true} {
		for _, thread := range []string{"", threadID} {
			for _, limit := range []int{1, 2, 20} {
				cursor := ""
				for {
					messages, nextCursor, err := p.MessageByChatID(testPublicChatID, thread, cursor, limit, threadsEnabled)
					require.NoError(t, err)
					ids := make([]string, 0, len(messages))
					for _, message := range messages {
						ids = append(ids, message.ID)
					}
					before, err := p.EmojiReactionsByChatID(testPublicChatID, thread, cursor, limit, threadsEnabled)
					require.NoError(t, err)
					after, err := p.emojiReactionsByChatIDsMessageIDs([]string{testPublicChatID}, ids)
					require.NoError(t, err)
					require.ElementsMatch(t, before, after)
					if nextCursor == "" {
						break
					}
					cursor = nextCursor
				}
			}
		}
	}
	_, err = db.Exec("UPDATE user_messages SET hide = 1 WHERE id = 'message-0'")
	require.NoError(t, err)
	require.NoError(t, p.SaveEmojiReaction(&EmojiReaction{From: "wrong-chat", LocalChatID: "other-chat",
		EmojiReaction: &protobuf.EmojiReaction{MessageId: "message-1", ChatId: "other-chat", Emoji: "1f600"}}))
	require.NoError(t, p.SaveEmojiReaction(&EmojiReaction{From: "orphan", LocalChatID: testPublicChatID,
		EmojiReaction: &protobuf.EmojiReaction{MessageId: "missing", ChatId: testPublicChatID, Emoji: "1f600"}}))
	reactions, err := p.emojiReactionsByChatIDsMessageIDs([]string{testPublicChatID},
		[]string{"message-0", "message-1", "message-1", "missing"})
	require.NoError(t, err)
	require.Len(t, reactions, 1)
	require.Equal(t, "message-1", reactions[0].MessageId)
	for _, ids := range [][]string{nil, {}} {
		reactions, err = p.emojiReactionsByChatIDsMessageIDs([]string{testPublicChatID}, ids)
		require.NoError(t, err)
		require.Empty(t, reactions)
	}
}

func TestReactionsByMessageIDsKeepsReactionLimit(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	p := newSQLitePersistence(db)
	require.NoError(t, p.SaveMessages([]*common.Message{{ID: "parent", LocalChatID: testPublicChatID,
		ChatMessage: &protobuf.ChatMessage{Text: "Parent", Clock: 1}}}))
	var reactions []*EmojiReaction
	for i := 0; i < 1005; i++ {
		reactions = append(reactions, &EmojiReaction{From: fmt.Sprint(i), LocalChatID: testPublicChatID,
			EmojiReaction: &protobuf.EmojiReaction{MessageId: "parent", ChatId: testPublicChatID, Emoji: "1f600"}})
	}
	require.NoError(t, p.SaveEmojiReactions(reactions))
	result, err := p.emojiReactionsByChatIDsMessageIDs([]string{testPublicChatID}, []string{"parent"})
	require.NoError(t, err)
	require.Len(t, result, 1000)
}
