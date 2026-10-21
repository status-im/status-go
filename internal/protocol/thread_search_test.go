package protocol

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/protocol/protobuf"
)

func TestThreadSearch(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	// Exercise both existing search conditions with SQLite's case-sensitive LIKE enabled.
	_, err = db.Exec("PRAGMA case_sensitive_like = ON")
	require.NoError(t, err)
	p := newSQLitePersistence(db)
	threadID, siblingID := "thread", "sibling-thread"
	require.NoError(t, p.SaveMessages([]*common.Message{
		{ID: "root", LocalChatID: testPublicChatID, From: testPK, ChatMessage: &protobuf.ChatMessage{ChatId: testPublicChatID, Text: "Needle root", Clock: 1}},
		{ID: "reply", LocalChatID: testPublicChatID, From: testPK, ChatMessage: &protobuf.ChatMessage{ChatId: testPublicChatID, Text: "Needle reply", ThreadId: &threadID, Clock: 2}},
		{ID: "lowercase", LocalChatID: testPublicChatID, From: testPK, ChatMessage: &protobuf.ChatMessage{ChatId: testPublicChatID, Text: "needle reply", ThreadId: &threadID, Clock: 3}},
		{ID: "channel", LocalChatID: testPublicChatID, From: testPK, ChatMessage: &protobuf.ChatMessage{ChatId: testPublicChatID, Text: "Needle channel"}},
		{ID: "sibling", LocalChatID: testPublicChatID, From: testPK, ChatMessage: &protobuf.ChatMessage{ChatId: testPublicChatID, Text: "Needle sibling", ThreadId: &siblingID}},
		{ID: "other-chat", LocalChatID: "other-chat", From: testPK, ChatMessage: &protobuf.ChatMessage{ChatId: testPublicChatID, Text: "Needle other chat", ThreadId: &threadID}},
		{ID: "hidden", LocalChatID: testPublicChatID, From: testPK, ChatMessage: &protobuf.ChatMessage{ChatId: testPublicChatID, Text: "Needle hidden", ThreadId: &threadID}},
	}))
	require.NoError(t, p.UpsertThread(threadID, testPublicChatID, "root", "Thread"))
	_, err = db.Exec("UPDATE user_messages SET hide = 1 WHERE id = ?", "hidden")
	require.NoError(t, err)

	for _, tc := range []struct {
		name      string
		term      string
		sensitive bool
		expected  []string
	}{
		{"insensitive", "needle", false, []string{"lowercase", "reply", "root"}},
		{"sensitive", "Needle", true, []string{"reply", "root"}},
		{"root only", "root", false, []string{"root"}},
		{"no matches", "absent", false, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			messages, err := p.AllMessagesFromThreadWhichMatchTerm(testPublicChatID, threadID, tc.term, tc.sensitive)
			require.NoError(t, err)
			var ids []string
			for _, message := range messages {
				ids = append(ids, message.ID)
			}
			require.Equal(t, tc.expected, ids)
		})
	}
	for _, args := range [][3]string{{testPublicChatID, threadID, ""}, {testPublicChatID, "", "needle"}, {"", threadID, "needle"}} {
		_, err := p.AllMessagesFromThreadWhichMatchTerm(args[0], args[1], args[2], false)
		require.Error(t, err)
	}
	messages, err := p.AllMessageByChatIDWhichMatchTerm(testPublicChatID, "needle", false)
	require.NoError(t, err)
	require.Len(t, messages, 5)

	// Invalid metadata must never pull a root from another chat into this scope.
	require.NoError(t, p.UpsertThread("cross-chat-root", testPublicChatID, "other-chat", "Invalid root"))
	messages, err = p.AllMessagesFromThreadWhichMatchTerm(testPublicChatID, "cross-chat-root", "needle", false)
	require.NoError(t, err)
	require.Empty(t, messages)

	// The public messenger path must validate the thread's parent before searching.
	messenger := &Messenger{persistence: p, allChats: &chatMap{}}
	messenger.allChats.Store(testPublicChatID, &Chat{ID: testPublicChatID})
	messages, err = messenger.AllMessagesFromThreadWhichMatchTerm(testPublicChatID, threadID, "needle", false)
	require.NoError(t, err)
	require.Len(t, messages, 3)
	_, err = messenger.AllMessagesFromThreadWhichMatchTerm(testPublicChatID, "missing-thread", "needle", false)
	require.Error(t, err)
	_, err = messenger.AllMessagesFromThreadWhichMatchTerm("other-chat", threadID, "needle", false)
	require.Error(t, err)
}

func TestThreadSearchAcrossChatsAndCommunities(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	_, err = db.Exec("PRAGMA case_sensitive_like = ON")
	require.NoError(t, err)
	p := newSQLitePersistence(db)
	require.NoError(t, p.SaveChats([]*Chat{
		{ID: "personal", ChatType: ChatTypeOneToOne},
		{ID: "channel-a", ChatType: ChatTypeCommunityChat, CommunityID: "community"},
		{ID: "channel-b", ChatType: ChatTypeCommunityChat, CommunityID: "community"},
		{ID: "other-channel", ChatType: ChatTypeCommunityChat, CommunityID: "other-community"},
	}))

	var messages []*common.Message
	for _, chatID := range []string{"personal", "channel-a", "channel-b", "other-channel"} {
		rootID := chatID + "-root"
		for i, kind := range []string{"root", "reply", "lowercase", "hidden", "unmatched"} {
			body := &protobuf.ChatMessage{ChatId: chatID, Text: "Needle " + kind, Clock: uint64(i + 1)}
			if kind != "root" {
				body.ThreadId = &rootID
			}
			if kind == "lowercase" {
				body.Text = "needle reply"
			} else if kind == "unmatched" {
				body.Text = "unrelated text"
			}
			messages = append(messages, &common.Message{
				ID: chatID + "-" + kind, LocalChatID: chatID, From: testPK, ChatMessage: body,
			})
		}
		require.NoError(t, p.UpsertThread(rootID, chatID, rootID, "Thread"))
	}
	require.NoError(t, p.SaveMessages(messages))
	_, err = db.Exec("UPDATE user_messages SET hide = 1 WHERE id LIKE '%-hidden'")
	require.NoError(t, err)

	for _, scope := range []struct {
		name         string
		communities  []string
		chats        []string
		matchedChats []string
	}{
		{"everywhere", []string{"community", "other-community"}, []string{"personal"}, []string{"personal", "channel-a", "channel-b", "other-channel"}},
		{"full community", []string{"community"}, nil, []string{"channel-a", "channel-b"}},
	} {
		t.Run(scope.name, func(t *testing.T) {
			for _, sensitive := range []bool{false, true} {
				t.Run(fmt.Sprintf("case-sensitive=%t", sensitive), func(t *testing.T) {
					matches, err := p.AllMessagesFromChatsAndCommunitiesWhichMatchTerm(scope.communities, scope.chats, "Needle", sensitive)
					require.NoError(t, err)
					var expected, actual []string
					for _, chatID := range scope.matchedChats {
						expected = append(expected, chatID+"-root", chatID+"-reply")
						if !sensitive {
							expected = append(expected, chatID+"-lowercase")
						}
					}
					for _, message := range matches {
						actual = append(actual, message.ID)
						if message.ID != message.LocalChatID+"-root" {
							require.Equal(t, message.LocalChatID+"-root", message.GetThreadId())
						}
					}
					require.ElementsMatch(t, expected, actual)
				})
			}
		})
	}
}

func BenchmarkThreadSearch(b *testing.B) {
	for _, parentSize := range []int{1000, 10000} {
		b.Run(fmt.Sprintf("parent=%d", parentSize), func(b *testing.B) {
			db, err := openTestDB()
			require.NoError(b, err)
			b.Cleanup(func() { require.NoError(b, db.Close()) })
			p := newSQLitePersistence(db)
			threadID := "thread"
			messages := make([]*common.Message, parentSize)
			for i := range messages {
				body := &protobuf.ChatMessage{ChatId: testPublicChatID, Text: "needle", Clock: uint64(i)}
				if i < 10 {
					body.ThreadId = &threadID
				}
				messages[i] = &common.Message{ID: fmt.Sprintf("message-%d", i), LocalChatID: testPublicChatID, From: testPK, ChatMessage: body}
			}
			require.NoError(b, p.SaveMessages(messages))
			require.NoError(b, p.UpsertThread(threadID, testPublicChatID, "message-10", "Thread"))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				matches, err := p.AllMessagesFromThreadWhichMatchTerm(testPublicChatID, threadID, "needle", false)
				require.NoError(b, err)
				require.Len(b, matches, 11)
			}
		})
	}
}
