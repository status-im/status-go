package protocol

import (
	"bytes"
	"database/sql"
	"io/ioutil"
	"math"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/sqlite"
)

func TestTableUserMessagesAllFieldsCount(t *testing.T) {
	db := sqlitePersistence{}
	expected := len(strings.Split(db.tableUserMessagesAllFields(), ","))
	require.Equal(t, expected, db.tableUserMessagesAllFieldsCount())
}

func TestSaveMessages(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	for i := 0; i < 10; i++ {
		id := strconv.Itoa(i)
		err := insertMinimalMessage(p, id)
		require.NoError(t, err)

		m, err := p.MessageByID(id)
		require.NoError(t, err)
		require.EqualValues(t, id, m.ID)
	}
}

func TestMessagesByIDs(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	var ids []string
	for i := 0; i < 10; i++ {
		id := strconv.Itoa(i)
		err := insertMinimalMessage(p, id)
		require.NoError(t, err)
		ids = append(ids, id)

	}
	m, err := p.MessagesByIDs(ids)
	require.NoError(t, err)
	require.Len(t, m, 10)
}

func TestMessagesByIDs_WithDiscordMessagesPayload(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	var ids []string
	for i := 0; i < 10; i++ {
		id := strconv.Itoa(i)
		err := insertMinimalMessage(p, id)
		require.NoError(t, err)
		err = insertMinimalDiscordMessage(p, id, id)
		require.NoError(t, err)
		ids = append(ids, id)
	}

	m, err := p.MessagesByIDs(ids)
	require.NoError(t, err)
	require.Len(t, m, 10)

	for _, _m := range m {
		require.NotNil(t, _m.GetDiscordMessage())
	}
}

func TestMessageByID(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)
	id := "1"

	err = insertMinimalMessage(p, id)
	require.NoError(t, err)

	m, err := p.MessageByID(id)
	require.NoError(t, err)
	require.EqualValues(t, id, m.ID)
}

func TestMessageByID_WithDiscordMessagePayload(t *testing.T) {

	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)
	id := "1"
	discordMessageID := "2"

	err = insertMinimalDiscordMessage(p, id, discordMessageID)
	require.NoError(t, err)

	m, err := p.MessageByID(id)
	require.NoError(t, err)
	require.EqualValues(t, id, m.ID)
	require.NotNil(t, m.GetDiscordMessage())
	require.EqualValues(t, discordMessageID, m.GetDiscordMessage().Id)
	require.EqualValues(t, "2", m.GetDiscordMessage().Author.Id)
}

func TestMessageByID_WithDiscordMessageAttachmentPayload(t *testing.T) {

	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)
	id := "1"
	discordMessageID := "2"

	err = insertDiscordMessageWithAttachments(p, id, discordMessageID)
	require.NoError(t, err)

	m, err := p.MessageByID(id)
	require.NoError(t, err)
	require.EqualValues(t, id, m.ID)

	dm := m.GetDiscordMessage()
	require.NotNil(t, dm)
	require.EqualValues(t, discordMessageID, dm.Id)

	require.NotNil(t, dm.Attachments)
	require.Len(t, dm.Attachments, 2)
}

func TestMessagesExist(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	err = insertMinimalMessage(p, "1")
	require.NoError(t, err)

	result, err := p.MessagesExist([]string{"1"})
	require.NoError(t, err)

	require.True(t, result["1"])

	err = insertMinimalMessage(p, "2")
	require.NoError(t, err)

	result, err = p.MessagesExist([]string{"1", "2", "3"})
	require.NoError(t, err)

	require.True(t, result["1"])
	require.True(t, result["2"])
	require.False(t, result["3"])
}

func TestMessageByChatID(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)
	chatID := testPublicChatID
	count := 1000
	pageSize := 50

	var messages []*common.Message
	for i := 0; i < count; i++ {
		messages = append(messages, &common.Message{
			ID:          strconv.Itoa(i),
			LocalChatID: chatID,
			ChatMessage: protobuf.ChatMessage{
				Clock: uint64(i),
			},
			From: testPK,
		})

		// Add some other chats.
		if count%5 == 0 {
			messages = append(messages, &common.Message{
				ID:          strconv.Itoa(count + i),
				LocalChatID: "other-chat",
				ChatMessage: protobuf.ChatMessage{
					Clock: uint64(i),
				},

				From: testPK,
			})
		}
	}

	// Add some out-of-order message. Add more than page size.
	outOfOrderCount := pageSize + 1
	allCount := count + outOfOrderCount
	for i := 0; i < pageSize+1; i++ {
		messages = append(messages, &common.Message{
			ID:          strconv.Itoa(count*2 + i),
			LocalChatID: chatID,
			ChatMessage: protobuf.ChatMessage{
				Clock: uint64(i),
			},

			From: testPK,
		})
	}

	err = p.SaveMessages(messages)
	require.NoError(t, err)

	var (
		result []*common.Message
		cursor string
		iter   int
	)
	for {
		var (
			items []*common.Message
			err   error
		)

		items, cursor, err = p.MessageByChatID(chatID, cursor, pageSize)
		require.NoError(t, err)
		result = append(result, items...)

		iter++
		if len(cursor) == 0 || iter > count {
			break
		}
	}
	require.Equal(t, "", cursor) // for loop should exit because of cursor being empty
	require.EqualValues(t, math.Ceil(float64(allCount)/float64(pageSize)), iter)
	require.Equal(t, len(result), allCount)
	require.True(
		t,
		// Verify descending order.
		sort.SliceIsSorted(result, func(i, j int) bool {
			return result[i].Clock > result[j].Clock
		}),
	)
}

func TestFirstUnseenMessageIDByChatID(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	messageID, err := p.FirstUnseenMessageID(testPublicChatID)
	require.NoError(t, err)
	require.Equal(t, "", messageID)

	err = p.SaveMessages([]*common.Message{
		{
			ID:          "1",
			LocalChatID: testPublicChatID,
			ChatMessage: protobuf.ChatMessage{
				Clock: 1,
				Text:  "some-text"},
			From: testPK,
			Seen: true,
		},
		{
			ID:          "2",
			LocalChatID: testPublicChatID,
			ChatMessage: protobuf.ChatMessage{
				Clock: 2,
				Text:  "some-text"},
			From: testPK,
			Seen: false,
		},
		{
			ID:          "3",
			LocalChatID: testPublicChatID,
			ChatMessage: protobuf.ChatMessage{
				Clock: 3,
				Text:  "some-text"},
			From: testPK,
			Seen: false,
		},
	})
	require.NoError(t, err)

	messageID, err = p.FirstUnseenMessageID(testPublicChatID)
	require.NoError(t, err)
	require.Equal(t, "2", messageID)
}

func TestLatestMessageByChatID(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	var ids []string
	for i := 0; i < 10; i++ {
		id := strconv.Itoa(i)
		err := insertMinimalMessage(p, id)
		require.NoError(t, err)
		ids = append(ids, id)
	}

	id := strconv.Itoa(10)
	err = insertMinimalDeletedMessage(p, id)
	require.NoError(t, err)
	ids = append(ids, id)

	id = strconv.Itoa(11)
	err = insertMinimalDeletedForMeMessage(p, id)
	require.NoError(t, err)
	ids = append(ids, id)

	m, err := p.LatestMessageByChatID(testPublicChatID)
	require.NoError(t, err)
	require.Equal(t, m[0].ID, ids[9])
}

func TestOldestMessageWhisperTimestampByChatID(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)
	chatID := testPublicChatID

	_, hasMessage, err := p.OldestMessageWhisperTimestampByChatID(chatID)
	require.NoError(t, err)
	require.False(t, hasMessage)

	var messages []*common.Message
	for i := 0; i < 10; i++ {
		messages = append(messages, &common.Message{
			ID:          strconv.Itoa(i),
			LocalChatID: chatID,
			ChatMessage: protobuf.ChatMessage{
				Clock: uint64(i),
			},
			WhisperTimestamp: uint64(i + 10),
			From:             testPK,
		})
	}

	err = p.SaveMessages(messages)
	require.NoError(t, err)

	timestamp, hasMessage, err := p.OldestMessageWhisperTimestampByChatID(chatID)
	require.NoError(t, err)
	require.True(t, hasMessage)
	require.Equal(t, uint64(10), timestamp)
}

func TestPinMessageByChatID(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := sqlitePersistence{db: db}
	chatID := "chat-with-pinned-messages"
	messagesCount := 1000
	pageSize := 5
	pinnedMessagesCount := 0

	var messages []*common.Message
	var pinMessages []*common.PinMessage
	for i := 0; i < messagesCount; i++ {
		messages = append(messages, &common.Message{
			ID:          strconv.Itoa(i),
			LocalChatID: chatID,
			ChatMessage: protobuf.ChatMessage{
				Clock: uint64(i),
			},
			From: testPK,
		})

		// Pin this message
		if i%100 == 0 {
			from := testPK
			if i == 100 {
				from = "them"
			}

			pinMessage := &common.PinMessage{
				ID:          strconv.Itoa(i),
				LocalChatID: chatID,
				From:        from,
			}

			pinMessage.MessageId = strconv.Itoa(i)
			pinMessage.Clock = 111
			pinMessage.Pinned = true
			pinMessages = append(pinMessages, pinMessage)
			pinnedMessagesCount++

			if i%200 == 0 {
				// unpin a message
				unpinMessage := &common.PinMessage{
					ID:          strconv.Itoa(i),
					LocalChatID: chatID,
					From:        testPK,
				}
				pinMessage.MessageId = strconv.Itoa(i)
				unpinMessage.Clock = 333
				unpinMessage.Pinned = false
				pinMessages = append(pinMessages, unpinMessage)
				pinnedMessagesCount--

				// pinned before the unpin
				pinMessage2 := &common.PinMessage{
					ID:          strconv.Itoa(i),
					LocalChatID: chatID,
					From:        testPK,
				}
				pinMessage2.MessageId = strconv.Itoa(i)
				pinMessage2.Clock = 222
				pinMessage2.Pinned = true
				pinMessages = append(pinMessages, pinMessage2)
			}
		}

		// Add some other chats.
		if i%5 == 0 {
			messages = append(messages, &common.Message{
				ID:          strconv.Itoa(messagesCount + i),
				LocalChatID: "chat-without-pinned-messages",
				ChatMessage: protobuf.ChatMessage{
					Clock: uint64(i),
				},

				From: testPK,
			})
		}
	}

	err = p.SaveMessages(messages)
	require.NoError(t, err)

	err = p.SavePinMessages(pinMessages)
	require.NoError(t, err)

	var (
		result []*common.PinnedMessage
		cursor string
		iter   int
	)
	for {
		var (
			items []*common.PinnedMessage
			err   error
		)

		items, cursor, err = p.PinnedMessageByChatID(chatID, cursor, pageSize)
		require.NoError(t, err)
		result = append(result, items...)

		iter++
		if len(cursor) == 0 || iter > messagesCount {
			break
		}
	}

	require.Equal(t, "", cursor) // for loop should exit because of cursor being empty
	require.EqualValues(t, pinnedMessagesCount, len(result))
	require.EqualValues(t, math.Ceil(float64(pinnedMessagesCount)/float64(pageSize)), iter)
	require.True(
		t,
		// Verify descending order.
		sort.SliceIsSorted(result, func(i, j int) bool {
			return result[i].Message.Clock > result[j].Message.Clock
		}),
	)

	require.Equal(t, "them", result[len(result)-1].PinnedBy)
	for i := 0; i < len(result)-1; i++ {
		require.Equal(t, testPK, result[i].PinnedBy)
	}
}

func TestMessageReplies(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)
	chatID := testPublicChatID
	message1 := &common.Message{
		ID:          "id-1",
		LocalChatID: chatID,
		ChatMessage: protobuf.ChatMessage{
			Text:  "content-1",
			Clock: uint64(1),
		},
		From: "1",
	}
	message2 := &common.Message{
		ID:          "id-2",
		LocalChatID: chatID,
		ChatMessage: protobuf.ChatMessage{
			Text:       "content-2",
			Clock:      uint64(2),
			ResponseTo: "id-1",
		},

		From: "2",
	}

	message3 := &common.Message{
		ID:          "id-3",
		LocalChatID: chatID,
		ChatMessage: protobuf.ChatMessage{
			Text:       "content-3",
			Clock:      uint64(3),
			ResponseTo: "non-existing",
		},
		From: "3",
	}

	// Message that is deleted
	message4 := &common.Message{
		ID:          "id-4",
		LocalChatID: chatID,
		Deleted:     true,
		ChatMessage: protobuf.ChatMessage{
			Text:  "content-4",
			Clock: uint64(4),
		},
		From: "2",
	}

	// Message replied to a deleted message. It will not have QuotedMessage info
	message5 := &common.Message{
		ID:          "id-5",
		LocalChatID: chatID,
		ChatMessage: protobuf.ChatMessage{
			Text:       "content-4",
			Clock:      uint64(5),
			ResponseTo: "id-4",
		},
		From: "3",
	}

	// messages := []*common.Message{message1, message2, message3}
	messages := []*common.Message{message1, message2, message3, message4, message5}

	err = p.SaveMessages(messages)
	require.NoError(t, err)

	retrievedMessages, _, err := p.MessageByChatID(chatID, "", 10)
	require.NoError(t, err)

	require.Equal(t, "non-existing", retrievedMessages[2].ResponseTo)
	require.Nil(t, retrievedMessages[2].QuotedMessage)

	require.Equal(t, "id-1", retrievedMessages[3].ResponseTo)
	require.Equal(t, &common.QuotedMessage{ID: "id-1", From: "1", Text: "content-1"}, retrievedMessages[3].QuotedMessage)

	require.Equal(t, "", retrievedMessages[4].ResponseTo)
	require.Nil(t, retrievedMessages[4].QuotedMessage)

	// We have a ResponseTo, but no QuotedMessage only gives the ID and Deleted
	require.Equal(t, "id-4", retrievedMessages[0].ResponseTo)
	require.Equal(t, &common.QuotedMessage{ID: "id-4", Deleted: true}, retrievedMessages[0].QuotedMessage)
}

func TestMessageByChatIDWithTheSameClocks(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)
	chatID := testPublicChatID
	clockValues := []uint64{10, 10, 9, 9, 9, 11, 12, 11, 100000, 6, 4, 5, 5, 5, 5}
	count := len(clockValues)
	pageSize := 2

	var messages []*common.Message

	for i, clock := range clockValues {
		messages = append(messages, &common.Message{
			ID:          strconv.Itoa(i),
			LocalChatID: chatID,
			ChatMessage: protobuf.ChatMessage{
				Clock: clock,
			},
			From: testPK,
		})
	}

	err = p.SaveMessages(messages)
	require.NoError(t, err)

	var (
		result []*common.Message
		cursor string
		iter   int
	)
	for {
		var (
			items []*common.Message
			err   error
		)

		items, cursor, err = p.MessageByChatID(chatID, cursor, pageSize)
		require.NoError(t, err)
		result = append(result, items...)

		iter++
		if cursor == "" || iter > count {
			break
		}
	}
	require.Empty(t, cursor) // for loop should exit because of cursor being empty
	require.Len(t, result, count)
	// Verify the order.
	expectedClocks := make([]uint64, len(clockValues))
	copy(expectedClocks, clockValues)
	sort.Slice(expectedClocks, func(i, j int) bool {
		return expectedClocks[i] > expectedClocks[j]
	})
	resultClocks := make([]uint64, 0, len(clockValues))
	for _, m := range result {
		resultClocks = append(resultClocks, m.Clock)
	}
	require.EqualValues(t, expectedClocks, resultClocks)
}

func TestDeleteMessageByID(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)
	id := "1"

	err = insertMinimalMessage(p, id)
	require.NoError(t, err)

	m, err := p.MessageByID(id)
	require.NoError(t, err)
	require.Equal(t, id, m.ID)

	err = p.DeleteMessage(m.ID)
	require.NoError(t, err)

	_, err = p.MessageByID(id)
	require.EqualError(t, err, "record not found")
}

func TestDeleteMessagesByChatID(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	err = insertMinimalMessage(p, "1")
	require.NoError(t, err)

	err = insertMinimalMessage(p, "2")
	require.NoError(t, err)

	m, _, err := p.MessageByChatID(testPublicChatID, "", 10)
	require.NoError(t, err)
	require.Equal(t, 2, len(m))

	err = p.DeleteMessagesByChatID(testPublicChatID)
	require.NoError(t, err)

	m, _, err = p.MessageByChatID(testPublicChatID, "", 10)
	require.NoError(t, err)
	require.Equal(t, 0, len(m))

}

func TestMarkMessageSeen(t *testing.T) {
	chatID := "test-chat"
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)
	id := "1"

	err = insertMinimalMessage(p, id)
	require.NoError(t, err)

	m, err := p.MessageByID(id)
	require.NoError(t, err)
	require.False(t, m.Seen)

	count, countWithMention, err := p.MarkMessagesSeen(chatID, []string{m.ID})
	require.NoError(t, err)
	require.Equal(t, uint64(1), count)
	require.Equal(t, uint64(0), countWithMention)

	m, err = p.MessageByID(id)
	require.NoError(t, err)
	require.True(t, m.Seen)
}

func TestUpdateMessageOutgoingStatus(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)
	id := "1"

	err = insertMinimalMessage(p, id)
	require.NoError(t, err)

	err = p.UpdateMessageOutgoingStatus(id, "new-status")
	require.NoError(t, err)

	m, err := p.MessageByID(id)
	require.NoError(t, err)
	require.Equal(t, "new-status", m.OutgoingStatus)
}

func TestMessagesIDsByType(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	ids, err := p.RawMessagesIDsByType(protobuf.ApplicationMetadataMessage_CHAT_MESSAGE)
	require.NoError(t, err)
	require.Empty(t, ids)

	err = p.SaveRawMessage(minimalRawMessage("chat-message-id", protobuf.ApplicationMetadataMessage_CHAT_MESSAGE))
	require.NoError(t, err)
	ids, err = p.RawMessagesIDsByType(protobuf.ApplicationMetadataMessage_CHAT_MESSAGE)
	require.NoError(t, err)
	require.Equal(t, 1, len(ids))
	require.Equal(t, "chat-message-id", ids[0])

	ids, err = p.RawMessagesIDsByType(protobuf.ApplicationMetadataMessage_EMOJI_REACTION)
	require.NoError(t, err)
	require.Empty(t, ids)

	err = p.SaveRawMessage(minimalRawMessage("emoji-message-id", protobuf.ApplicationMetadataMessage_EMOJI_REACTION))
	require.NoError(t, err)
	ids, err = p.RawMessagesIDsByType(protobuf.ApplicationMetadataMessage_EMOJI_REACTION)
	require.NoError(t, err)
	require.Equal(t, 1, len(ids))
	require.Equal(t, "emoji-message-id", ids[0])
}

func TestExpiredMessagesIDs(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	ids, err := p.ExpiredMessagesIDs(messageResendMaxCount)
	require.NoError(t, err)
	require.Empty(t, ids)

	//save expired emoji message
	rawEmojiReaction := minimalRawMessage("emoji-message-id", protobuf.ApplicationMetadataMessage_EMOJI_REACTION)
	rawEmojiReaction.Sent = false
	err = p.SaveRawMessage(rawEmojiReaction)
	require.NoError(t, err)

	//make sure it appered in expired emoji reactions list
	ids, err = p.ExpiredMessagesIDs(messageResendMaxCount)
	require.NoError(t, err)
	require.Equal(t, 1, len(ids))

	//save non-expired emoji reaction
	rawEmojiReaction2 := minimalRawMessage("emoji-message-id2", protobuf.ApplicationMetadataMessage_EMOJI_REACTION)
	rawEmojiReaction2.Sent = true
	err = p.SaveRawMessage(rawEmojiReaction2)
	require.NoError(t, err)

	//make sure it didn't appear in expired emoji reactions list
	ids, err = p.ExpiredMessagesIDs(messageResendMaxCount)
	require.NoError(t, err)
	require.Equal(t, 1, len(ids))
}

func TestPersistenceEmojiReactions(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)
	// reverse order as we use DESC
	id1 := "1"
	id2 := "2"
	id3 := "3"

	from1 := "from-1"
	from2 := "from-2"
	from3 := "from-3"

	chatID := testPublicChatID

	err = insertMinimalMessage(p, id1)
	require.NoError(t, err)

	err = insertMinimalMessage(p, id2)
	require.NoError(t, err)

	err = insertMinimalMessage(p, id3)
	require.NoError(t, err)

	// Insert normal emoji reaction
	require.NoError(t, p.SaveEmojiReaction(&EmojiReaction{
		EmojiReaction: protobuf.EmojiReaction{
			Clock:     1,
			MessageId: id3,
			ChatId:    chatID,
			Type:      protobuf.EmojiReaction_SAD,
		},
		LocalChatID: chatID,
		From:        from1,
	}))

	// Insert retracted emoji reaction
	require.NoError(t, p.SaveEmojiReaction(&EmojiReaction{
		EmojiReaction: protobuf.EmojiReaction{
			Clock:     1,
			MessageId: id3,
			ChatId:    chatID,
			Type:      protobuf.EmojiReaction_SAD,
			Retracted: true,
		},
		LocalChatID: chatID,
		From:        from2,
	}))

	// Insert retracted emoji reaction out of pagination
	require.NoError(t, p.SaveEmojiReaction(&EmojiReaction{
		EmojiReaction: protobuf.EmojiReaction{
			Clock:     1,
			MessageId: id1,
			ChatId:    chatID,
			Type:      protobuf.EmojiReaction_SAD,
		},
		LocalChatID: chatID,
		From:        from2,
	}))

	// Insert retracted emoji reaction out of pagination
	require.NoError(t, p.SaveEmojiReaction(&EmojiReaction{
		EmojiReaction: protobuf.EmojiReaction{
			Clock:     1,
			MessageId: id1,
			ChatId:    chatID,
			Type:      protobuf.EmojiReaction_SAD,
		},
		LocalChatID: chatID,
		From:        from3,
	}))

	// Wrong local chat id
	require.NoError(t, p.SaveEmojiReaction(&EmojiReaction{
		EmojiReaction: protobuf.EmojiReaction{
			Clock:     1,
			MessageId: id1,
			ChatId:    chatID,
			Type:      protobuf.EmojiReaction_LOVE,
		},
		LocalChatID: "wrong-chat-id",
		From:        from3,
	}))

	reactions, err := p.EmojiReactionsByChatID(chatID, "", 1)
	require.NoError(t, err)
	require.Len(t, reactions, 1)
	require.Equal(t, id3, reactions[0].MessageId)

	// Try with a cursor
	_, cursor, err := p.MessageByChatID(chatID, "", 1)
	require.NoError(t, err)

	reactions, err = p.EmojiReactionsByChatID(chatID, cursor, 2)
	require.NoError(t, err)
	require.Len(t, reactions, 2)
	require.Equal(t, id1, reactions[0].MessageId)
	require.Equal(t, id1, reactions[1].MessageId)
}

func openTestDB() (*sql.DB, error) {
	dbPath, err := ioutil.TempFile("", "status-go-test-db-")
	if err != nil {
		return nil, err
	}
	return sqlite.Open(dbPath.Name(), "", sqlite.ReducedKDFIterationsNumber)
}

func insertMinimalMessage(p *sqlitePersistence, id string) error {
	return p.SaveMessages([]*common.Message{{
		ID:          id,
		LocalChatID: testPublicChatID,
		ChatMessage: protobuf.ChatMessage{Text: "some-text"},
		From:        testPK,
	}})
}

func insertMinimalDeletedMessage(p *sqlitePersistence, id string) error {
	return p.SaveMessages([]*common.Message{{
		ID:          id,
		Deleted:     true,
		LocalChatID: testPublicChatID,
		ChatMessage: protobuf.ChatMessage{Text: "some-text"},
		From:        testPK,
	}})
}

func insertMinimalDeletedForMeMessage(p *sqlitePersistence, id string) error {
	return p.SaveMessages([]*common.Message{{
		ID:           id,
		DeletedForMe: true,
		LocalChatID:  testPublicChatID,
		ChatMessage:  protobuf.ChatMessage{Text: "some-text"},
		From:         testPK,
	}})
}

func insertDiscordMessageWithAttachments(p *sqlitePersistence, id string, discordMessageID string) error {
	err := insertMinimalDiscordMessage(p, id, discordMessageID)
	if err != nil {
		return err
	}

	attachment := &protobuf.DiscordMessageAttachment{
		Id:        "1",
		MessageId: discordMessageID,
		Url:       "https://does-not-exist.com",
		Payload:   []byte{1, 2, 3, 4},
	}

	attachment2 := &protobuf.DiscordMessageAttachment{
		Id:        "2",
		MessageId: discordMessageID,
		Url:       "https://does-not-exist.com",
		Payload:   []byte{5, 6, 7, 8},
	}

	return p.SaveDiscordMessageAttachments([]*protobuf.DiscordMessageAttachment{
		attachment,
		attachment2,
	})
}

func insertMinimalDiscordMessage(p *sqlitePersistence, id string, discordMessageID string) error {
	discordMessage := &protobuf.DiscordMessage{
		Id:        discordMessageID,
		Type:      "Default",
		Timestamp: "123456",
		Content:   "This is the message",
		Author: &protobuf.DiscordMessageAuthor{
			Id: "2",
		},
		Reference: &protobuf.DiscordMessageReference{},
	}

	err := p.SaveDiscordMessage(discordMessage)
	if err != nil {
		return err
	}

	return p.SaveMessages([]*common.Message{{
		ID:          id,
		LocalChatID: testPublicChatID,
		From:        testPK,
		ChatMessage: protobuf.ChatMessage{
			Text:        "some-text",
			ContentType: protobuf.ChatMessage_DISCORD_MESSAGE,
			ChatId:      testPublicChatID,
			Payload: &protobuf.ChatMessage_DiscordMessage{
				DiscordMessage: discordMessage,
			},
		},
	}})
}

func minimalRawMessage(id string, messageType protobuf.ApplicationMetadataMessage_Type) *common.RawMessage {
	return &common.RawMessage{
		ID:          id,
		LocalChatID: "test-chat",
		MessageType: messageType,
	}
}

// Regression test making sure that if audio_duration_ms is null, no error is thrown
func TestMessagesAudioDurationMsNull(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)
	id := "message-id-1"

	err = insertMinimalMessage(p, id)
	require.NoError(t, err)

	_, err = p.db.Exec("UPDATE user_messages SET audio_duration_ms = NULL")
	require.NoError(t, err)

	m, err := p.MessagesByIDs([]string{id})
	require.NoError(t, err)
	require.Len(t, m, 1)

	m, _, err = p.MessageByChatID(testPublicChatID, "", 10)
	require.NoError(t, err)
	require.Len(t, m, 1)
}

func TestSaveChat(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	chat := CreatePublicChat("test-chat", &testTimeSource{})
	chat.LastMessage = &common.Message{}
	err = p.SaveChat(*chat)
	require.NoError(t, err)

	retrievedChat, err := p.Chat(chat.ID)
	require.NoError(t, err)
	require.Equal(t, chat, retrievedChat)
}

func TestSaveMentions(t *testing.T) {
	chatID := testPublicChatID
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	pkString := types.EncodeHex(crypto.FromECDSAPub(&key.PublicKey))

	message := common.Message{
		ID:          "1",
		LocalChatID: chatID,
		ChatMessage: protobuf.ChatMessage{Text: "some-text"},
		From:        testPK,
		Mentions:    []string{pkString},
	}

	err = p.SaveMessages([]*common.Message{&message})
	require.NoError(t, err)

	retrievedMessages, _, err := p.MessageByChatID(chatID, "", 10)
	require.NoError(t, err)
	require.Len(t, retrievedMessages, 1)
	require.Len(t, retrievedMessages[0].Mentions, 1)
	require.Equal(t, retrievedMessages[0].Mentions, message.Mentions)
}

func TestSqlitePersistence_GetWhenChatIdentityLastPublished(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	chatID := "0xabcd1234"
	hash := []byte{0x1}
	now := time.Now().Unix()

	err = p.SaveWhenChatIdentityLastPublished(chatID, hash)
	require.NoError(t, err)

	ts, actualHash, err := p.GetWhenChatIdentityLastPublished(chatID)
	require.NoError(t, err)

	// Check that the save happened in the last 2 seconds
	diff := ts - now
	require.LessOrEqual(t, diff, int64(2))

	require.True(t, bytes.Equal(hash, actualHash))

	// Require unsaved values to be zero
	ts2, actualHash2, err := p.GetWhenChatIdentityLastPublished("0xdeadbeef")
	require.NoError(t, err)
	require.Exactly(t, int64(0), ts2)
	require.Nil(t, actualHash2)
}

func TestSaveContactChatIdentity(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	contactID := types.EncodeHex(crypto.FromECDSAPub(&key.PublicKey))

	err = p.SaveContact(&Contact{ID: contactID}, nil)
	require.NoError(t, err)

	jpegType := []byte{0xff, 0xd8, 0xff, 0x1}
	identityImages := make(map[string]*protobuf.IdentityImage)
	identityImages["large"] = &protobuf.IdentityImage{
		Payload:    jpegType,
		SourceType: protobuf.IdentityImage_RAW_PAYLOAD,
		ImageType:  protobuf.ImageType_PNG,
	}

	identityImages["small"] = &protobuf.IdentityImage{
		Payload:    jpegType,
		SourceType: protobuf.IdentityImage_RAW_PAYLOAD,
		ImageType:  protobuf.ImageType_PNG,
	}

	toArrayOfPointers := func(array []protobuf.SocialLink) (result []*protobuf.SocialLink) {
		result = make([]*protobuf.SocialLink, len(array))
		for i := range array {
			result[i] = &array[i]
		}
		return
	}

	chatIdentity := &protobuf.ChatIdentity{
		Clock:  1,
		Images: identityImages,
		SocialLinks: toArrayOfPointers([]protobuf.SocialLink{
			{
				Text: "Personal Site",
				Url:  "status.im",
			},
			{
				Text: "Twitter",
				Url:  "Status_ico",
			},
		}),
	}

	clockUpdated, imagesUpdated, err := p.SaveContactChatIdentity(contactID, chatIdentity)
	require.NoError(t, err)
	require.True(t, clockUpdated)
	require.True(t, imagesUpdated)

	// Save again same clock and data
	clockUpdated, imagesUpdated, err = p.SaveContactChatIdentity(contactID, chatIdentity)
	require.NoError(t, err)
	require.False(t, clockUpdated)
	require.False(t, imagesUpdated)

	// Save again newer clock and no images
	chatIdentity.Clock = 2
	chatIdentity.Images = make(map[string]*protobuf.IdentityImage)
	clockUpdated, imagesUpdated, err = p.SaveContactChatIdentity(contactID, chatIdentity)
	require.NoError(t, err)
	require.True(t, clockUpdated)
	require.False(t, imagesUpdated)

	contacts, err := p.Contacts()
	require.NoError(t, err)
	require.Len(t, contacts, 1)

	require.Len(t, contacts[0].Images, 2)
	require.Len(t, contacts[0].SocialLinks, 2)
	require.Equal(t, "Personal Site", contacts[0].SocialLinks[0].Text)
	require.Equal(t, "status.im", contacts[0].SocialLinks[0].URL)
	require.Equal(t, "Twitter", contacts[0].SocialLinks[1].Text)
	require.Equal(t, "Status_ico", contacts[0].SocialLinks[1].URL)
}

func TestSaveLinks(t *testing.T) {
	chatID := testPublicChatID
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	require.NoError(t, err)

	message := common.Message{
		ID:          "1",
		LocalChatID: chatID,
		ChatMessage: protobuf.ChatMessage{Text: "some-text"},
		From:        testPK,
		Links:       []string{"https://github.com/status-im/status-mobile"},
	}

	err = p.SaveMessages([]*common.Message{&message})
	require.NoError(t, err)

	retrievedMessages, _, err := p.MessageByChatID(chatID, "", 10)
	require.NoError(t, err)
	require.Len(t, retrievedMessages, 1)
	require.Len(t, retrievedMessages[0].Links, 1)
	require.Equal(t, retrievedMessages[0].Links, message.Links)

}

func TestHideMessage(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)
	chatID := testPublicChatID
	message := &common.Message{
		ID:          "id-1",
		LocalChatID: chatID,
		ChatMessage: protobuf.ChatMessage{
			Text:  "content-1",
			Clock: uint64(1),
		},
		From: "1",
	}

	messages := []*common.Message{message}

	err = p.SaveMessages(messages)
	require.NoError(t, err)

	err = p.HideMessage(message.ID)
	require.NoError(t, err)

	var actualHidden, actualSeen bool
	err = p.db.QueryRow("SELECT hide, seen FROM user_messages WHERE id = ?", message.ID).Scan(&actualHidden, &actualSeen)

	require.NoError(t, err)
	require.True(t, actualHidden)
	require.True(t, actualSeen)
}

func TestDeactivatePublicChat(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)
	publicChatID := "public-chat-id"
	var currentClockValue uint64 = 10

	timesource := &testTimeSource{}
	lastMessage := common.Message{
		ID:          "0x01",
		LocalChatID: publicChatID,
		ChatMessage: protobuf.ChatMessage{Text: "some-text"},
		From:        testPK,
	}
	lastMessage.Clock = 20

	require.NoError(t, p.SaveMessages([]*common.Message{&lastMessage}))

	publicChat := CreatePublicChat(publicChatID, timesource)
	publicChat.LastMessage = &lastMessage
	publicChat.UnviewedMessagesCount = 1

	err = p.DeactivateChat(publicChat, currentClockValue, true)

	// It does not set deleted at for a public chat
	require.NoError(t, err)
	require.Equal(t, uint64(0), publicChat.DeletedAtClockValue)

	// It sets the lastMessage to nil
	require.Nil(t, publicChat.LastMessage)

	// It sets unviewed messages count
	require.Equal(t, uint(0), publicChat.UnviewedMessagesCount)

	// It sets active as false
	require.False(t, publicChat.Active)

	// It deletes messages
	messages, _, err := p.MessageByChatID(publicChatID, "", 10)
	require.NoError(t, err)
	require.Len(t, messages, 0)

	// Reload chat to make sure it has been save
	dbChat, err := p.Chat(publicChatID)

	require.NoError(t, err)
	require.NotNil(t, dbChat)

	// Same checks on the chat pulled from the db
	// It does not set deleted at for a public chat
	require.NoError(t, err)
	require.Equal(t, uint64(0), dbChat.DeletedAtClockValue)

	// It sets the lastMessage to nil
	require.Nil(t, dbChat.LastMessage)

	// It sets unviewed messages count
	require.Equal(t, uint(0), dbChat.UnviewedMessagesCount)

	// It sets active as false
	require.False(t, dbChat.Active)
}

func TestDeactivateOneToOneChat(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	pkString := types.EncodeHex(crypto.FromECDSAPub(&key.PublicKey))

	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)
	var currentClockValue uint64 = 10

	timesource := &testTimeSource{}

	chat := CreateOneToOneChat(pkString, &key.PublicKey, timesource)

	lastMessage := common.Message{
		ID:          "0x01",
		LocalChatID: chat.ID,
		ChatMessage: protobuf.ChatMessage{Text: "some-text"},
		From:        testPK,
	}
	lastMessage.Clock = 20

	require.NoError(t, p.SaveMessages([]*common.Message{&lastMessage}))

	chat.LastMessage = &lastMessage
	chat.UnviewedMessagesCount = 1

	err = p.DeactivateChat(chat, currentClockValue, true)

	// It does set deleted at for a public chat
	require.NoError(t, err)
	require.NotEqual(t, uint64(0), chat.DeletedAtClockValue)

	// It sets the lastMessage to nil
	require.Nil(t, chat.LastMessage)

	// It sets unviewed messages count
	require.Equal(t, uint(0), chat.UnviewedMessagesCount)

	// It sets active as false
	require.False(t, chat.Active)

	// It deletes messages
	messages, _, err := p.MessageByChatID(chat.ID, "", 10)
	require.NoError(t, err)
	require.Len(t, messages, 0)

	// Reload chat to make sure it has been save
	dbChat, err := p.Chat(chat.ID)

	require.NoError(t, err)
	require.NotNil(t, dbChat)

	// Same checks on the chat pulled from the db
	// It does set deleted at for a public chat
	require.NoError(t, err)
	require.NotEqual(t, uint64(0), dbChat.DeletedAtClockValue)

	// It sets the lastMessage to nil
	require.Nil(t, dbChat.LastMessage)

	// It sets unviewed messages count
	require.Equal(t, uint(0), dbChat.UnviewedMessagesCount)

	// It sets active as false
	require.False(t, dbChat.Active)
}

func TestConfirmations(t *testing.T) {
	dataSyncID1 := []byte("datsync-id-1")
	dataSyncID2 := []byte("datsync-id-2")
	dataSyncID3 := []byte("datsync-id-3")
	dataSyncID4 := []byte("datsync-id-3")

	messageID1 := []byte("message-id-1")
	messageID2 := []byte("message-id-2")

	publicKey1 := []byte("pk-1")
	publicKey2 := []byte("pk-2")
	publicKey3 := []byte("pk-3")

	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	confirmation1 := &common.RawMessageConfirmation{
		DataSyncID: dataSyncID1,
		MessageID:  messageID1,
		PublicKey:  publicKey1,
	}

	// Same datasyncID and same messageID, different pubkey
	confirmation2 := &common.RawMessageConfirmation{
		DataSyncID: dataSyncID2,
		MessageID:  messageID1,
		PublicKey:  publicKey2,
	}

	// Different datasyncID and same messageID, different pubkey
	confirmation3 := &common.RawMessageConfirmation{
		DataSyncID: dataSyncID3,
		MessageID:  messageID1,
		PublicKey:  publicKey3,
	}

	// Same dataSyncID, different messageID
	confirmation4 := &common.RawMessageConfirmation{
		DataSyncID: dataSyncID4,
		MessageID:  messageID2,
		PublicKey:  publicKey1,
	}

	require.NoError(t, p.InsertPendingConfirmation(confirmation1))
	require.NoError(t, p.InsertPendingConfirmation(confirmation2))
	require.NoError(t, p.InsertPendingConfirmation(confirmation3))
	require.NoError(t, p.InsertPendingConfirmation(confirmation4))

	// We confirm the first datasync message, no confirmations
	messageID, err := p.MarkAsConfirmed(dataSyncID1, false)
	require.NoError(t, err)
	require.Nil(t, messageID)

	// We confirm the second datasync message, no confirmations
	messageID, err = p.MarkAsConfirmed(dataSyncID2, false)
	require.NoError(t, err)
	require.Nil(t, messageID)

	// We confirm the third datasync message, messageID1 should be confirmed
	messageID, err = p.MarkAsConfirmed(dataSyncID3, false)
	require.NoError(t, err)
	require.Equal(t, messageID, types.HexBytes(messageID1))
}

func TestConfirmationsAtLeastOne(t *testing.T) {
	dataSyncID1 := []byte("datsync-id-1")
	dataSyncID2 := []byte("datsync-id-2")
	dataSyncID3 := []byte("datsync-id-3")

	messageID1 := []byte("message-id-1")

	publicKey1 := []byte("pk-1")
	publicKey2 := []byte("pk-2")
	publicKey3 := []byte("pk-3")

	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	confirmation1 := &common.RawMessageConfirmation{
		DataSyncID: dataSyncID1,
		MessageID:  messageID1,
		PublicKey:  publicKey1,
	}

	// Same datasyncID and same messageID, different pubkey
	confirmation2 := &common.RawMessageConfirmation{
		DataSyncID: dataSyncID2,
		MessageID:  messageID1,
		PublicKey:  publicKey2,
	}

	// Different datasyncID and same messageID, different pubkey
	confirmation3 := &common.RawMessageConfirmation{
		DataSyncID: dataSyncID3,
		MessageID:  messageID1,
		PublicKey:  publicKey3,
	}

	require.NoError(t, p.InsertPendingConfirmation(confirmation1))
	require.NoError(t, p.InsertPendingConfirmation(confirmation2))
	require.NoError(t, p.InsertPendingConfirmation(confirmation3))

	// We confirm the first datasync message, messageID1 and 3 should be confirmed
	messageID, err := p.MarkAsConfirmed(dataSyncID1, true)
	require.NoError(t, err)
	require.NotNil(t, messageID)
	require.Equal(t, types.HexBytes(messageID1), messageID)
}

func TestActivityCenterReadUnread(t *testing.T) {
	nID1 := types.HexBytes([]byte("1"))
	nID2 := types.HexBytes([]byte("2"))
	//	nID3 := types.HexBytes([]byte("3"))
	//	nID4 := types.HexBytes([]byte("4"))

	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	chat := CreatePublicChat("test-chat", &testTimeSource{})
	message := &common.Message{}
	message.Text = "sample text"
	chat.LastMessage = message
	err = p.SaveChat(*chat)
	require.NoError(t, err)

	notification := &ActivityCenterNotification{
		ID:        nID1,
		Type:      ActivityCenterNotificationTypeNewOneToOne,
		ChatID:    chat.ID,
		Timestamp: 1,
	}

	err = p.SaveActivityCenterNotification(notification)
	require.NoError(t, err)

	notification = &ActivityCenterNotification{
		ID:        nID2,
		Type:      ActivityCenterNotificationTypeNewOneToOne,
		ChatID:    chat.ID,
		Timestamp: 1,
	}

	err = p.SaveActivityCenterNotification(notification)
	require.NoError(t, err)

	// Mark the notification as read
	err = p.MarkActivityCenterNotificationsRead([]types.HexBytes{nID2})
	require.NoError(t, err)

	cursor, notifications, err := p.UnreadActivityCenterNotifications(
		"",
		2,
		[]ActivityCenterType{ActivityCenterNotificationTypeNewOneToOne},
	)
	require.NoError(t, err)
	require.Empty(t, cursor)
	require.Len(t, notifications, 1)
	require.Equal(t, nID1, notifications[0].ID)

	cursor, notifications, err = p.ReadActivityCenterNotifications(
		"",
		2,
		[]ActivityCenterType{ActivityCenterNotificationTypeNewOneToOne},
	)
	require.NoError(t, err)
	require.Empty(t, cursor)
	require.Len(t, notifications, 1)
	require.Equal(t, nID2, notifications[0].ID)
}

func TestUnreadAndAcceptedActivityCenterNotificationsCount(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	chat := CreatePublicChat("test-chat", &testTimeSource{})
	message := &common.Message{}
	message.Text = "sample text"
	chat.LastMessage = message
	err = p.SaveChat(*chat)
	require.NoError(t, err)

	allNotifications := []*ActivityCenterNotification{
		{
			ID:        types.HexBytes("1"),
			Type:      ActivityCenterNotificationTypeMention,
			ChatID:    chat.ID,
			Timestamp: 1,
		},
		{
			ID:        types.HexBytes("2"),
			Type:      ActivityCenterNotificationTypeNewOneToOne,
			ChatID:    chat.ID,
			Timestamp: 1,
		},
		{
			ID:        types.HexBytes("3"),
			Type:      ActivityCenterNotificationTypeMention,
			ChatID:    chat.ID,
			Timestamp: 1,
		},
		{
			ID:        types.HexBytes("4"),
			Type:      ActivityCenterNotificationTypeMention,
			ChatID:    chat.ID,
			Timestamp: 1,
		},
		{
			ID:        types.HexBytes("5"),
			Type:      ActivityCenterNotificationTypeContactRequest,
			ChatID:    chat.ID,
			Timestamp: 1,
		},
	}

	for _, notification := range allNotifications {
		err = p.SaveActivityCenterNotification(notification)
		require.NoError(t, err)
	}

	notificationCount, err := p.UnreadAndAcceptedActivityCenterNotificationsCount(
		[]ActivityCenterType{},
	)
	require.NoError(t, err)
	require.Equal(t, notificationCount, uint64(5))

	notificationCount, err = p.UnreadAndAcceptedActivityCenterNotificationsCount(
		[]ActivityCenterType{
			ActivityCenterNotificationTypeNewOneToOne,
		},
	)
	require.NoError(t, err)
	require.Equal(t, notificationCount, uint64(1))

	notificationCount, err = p.UnreadAndAcceptedActivityCenterNotificationsCount(
		[]ActivityCenterType{
			ActivityCenterNotificationTypeNewOneToOne,
			ActivityCenterNotificationTypeContactRequest,
		},
	)
	require.NoError(t, err)
	require.Equal(t, notificationCount, uint64(2))
}

func TestActivityCenterReadUnreadFilterByTypes(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	chat := CreatePublicChat("test-chat", &testTimeSource{})
	message := &common.Message{}
	message.Text = "sample text"
	chat.LastMessage = message
	err = p.SaveChat(*chat)
	require.NoError(t, err)

	initialCursor := ""
	limit := uint64(3)

	nID1 := types.HexBytes([]byte("1"))
	nID2 := types.HexBytes([]byte("2"))
	nID3 := types.HexBytes([]byte("3"))

	allNotifications := []*ActivityCenterNotification{
		{
			ID:        nID1,
			Type:      ActivityCenterNotificationTypeMention,
			ChatID:    chat.ID,
			Timestamp: 1,
		},
		{
			ID:        nID2,
			Type:      ActivityCenterNotificationTypeNewOneToOne,
			ChatID:    chat.ID,
			Timestamp: 1,
		},
		{
			ID:        nID3,
			Type:      ActivityCenterNotificationTypeMention,
			ChatID:    chat.ID,
			Timestamp: 1,
		},
	}

	for _, notification := range allNotifications {
		err = p.SaveActivityCenterNotification(notification)
		require.NoError(t, err)
	}

	// Don't filter by type if the array of types is empty.
	_, notifications, err := p.UnreadActivityCenterNotifications(
		initialCursor,
		limit,
		[]ActivityCenterType{},
	)
	require.NoError(t, err)
	require.Len(t, notifications, 3)
	require.Equal(t, nID3, notifications[0].ID)
	require.Equal(t, nID2, notifications[1].ID)
	require.Equal(t, nID1, notifications[2].ID)

	_, notifications, err = p.UnreadActivityCenterNotifications(
		initialCursor,
		limit,
		[]ActivityCenterType{ActivityCenterNotificationTypeNewOneToOne},
	)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	require.Equal(t, nID2, notifications[0].ID)

	_, notifications, err = p.UnreadActivityCenterNotifications(
		initialCursor,
		limit,
		[]ActivityCenterType{ActivityCenterNotificationTypeMention},
	)
	require.NoError(t, err)
	require.Len(t, notifications, 2)
	require.Equal(t, nID3, notifications[0].ID)
	require.Equal(t, nID1, notifications[1].ID)

	_, notifications, err = p.UnreadActivityCenterNotifications(
		initialCursor,
		limit,
		[]ActivityCenterType{ActivityCenterNotificationTypeMention, ActivityCenterNotificationTypeNewOneToOne},
	)
	require.NoError(t, err)
	require.Len(t, notifications, 3)
	require.Equal(t, nID3, notifications[0].ID)
	require.Equal(t, nID2, notifications[1].ID)
	require.Equal(t, nID1, notifications[2].ID)

	// Mark all notifications as read.
	for _, notification := range allNotifications {
		err = p.MarkActivityCenterNotificationsRead([]types.HexBytes{notification.ID})
		require.NoError(t, err)
	}

	_, notifications, err = p.ReadActivityCenterNotifications(
		initialCursor,
		limit,
		[]ActivityCenterType{ActivityCenterNotificationTypeNewOneToOne},
	)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	require.Equal(t, nID2, notifications[0].ID)

	_, notifications, err = p.ReadActivityCenterNotifications(
		initialCursor,
		limit,
		[]ActivityCenterType{ActivityCenterNotificationTypeMention},
	)
	require.NoError(t, err)
	require.Len(t, notifications, 2)
	require.Equal(t, nID3, notifications[0].ID)
	require.Equal(t, nID1, notifications[1].ID)
}

func TestActivityCenterReadUnreadFilterByGroup(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	chat := CreatePublicChat("test-chat", &testTimeSource{})
	message := &common.Message{}
	message.Text = "sample text"
	chat.LastMessage = message
	err = p.SaveChat(*chat)
	require.NoError(t, err)

	initialCursor := ""
	limit := uint64(12)

	nID1 := types.HexBytes([]byte("1"))
	nID2 := types.HexBytes([]byte("2"))
	nID3 := types.HexBytes([]byte("3"))
	nID4 := types.HexBytes([]byte("4"))
	nID5 := types.HexBytes([]byte("5"))
	nID6 := types.HexBytes([]byte("6"))
	nID7 := types.HexBytes([]byte("7"))
	nID8 := types.HexBytes([]byte("8"))
	nID9 := types.HexBytes([]byte("9"))

	allNotifications := []*ActivityCenterNotification{
		{
			ID:        nID1,
			Type:      ActivityCenterNotificationTypeNewPrivateGroupChat,
			ChatID:    chat.ID,
			Timestamp: 1,
		},
		{
			ID:        nID2,
			Type:      ActivityCenterNotificationTypeMention,
			ChatID:    chat.ID,
			Timestamp: 1,
		},
		{
			ID:        nID3,
			Type:      ActivityCenterNotificationTypeReply,
			ChatID:    chat.ID,
			Timestamp: 1,
		},
		{
			ID:        nID4,
			Type:      ActivityCenterNotificationTypeContactRequest,
			ChatID:    chat.ID,
			Timestamp: 1,
		},
		{
			ID:        nID5,
			Type:      ActivityCenterNotificationTypeCommunityInvitation,
			ChatID:    chat.ID,
			Timestamp: 1,
		},
		{
			ID:        nID6,
			Type:      ActivityCenterNotificationTypeCommunityRequest,
			ChatID:    chat.ID,
			Timestamp: 1,
		},
		{
			ID:        nID7,
			Type:      ActivityCenterNotificationTypeCommunityMembershipRequest,
			ChatID:    chat.ID,
			Timestamp: 1,
		},
		{
			ID:        nID8,
			Type:      ActivityCenterNotificationTypeCommunityKicked,
			ChatID:    chat.ID,
			Timestamp: 1,
		},
		{
			ID:        nID9,
			Type:      ActivityCenterNotificationTypeContactVerification,
			ChatID:    chat.ID,
			Timestamp: 1,
		},
	}

	for _, notification := range allNotifications {
		err = p.SaveActivityCenterNotification(notification)
		require.NoError(t, err)
	}

	_, notifications, err := p.ActivityCenterNotificationsByGroup(
		initialCursor,
		limit,
		ActivityCenterAll,
		ActivityCenterQueryParamsReadAll,
	)
	require.NoError(t, err)
	require.Len(t, notifications, 9)
	require.Equal(t, nID9, notifications[0].ID)
	require.Equal(t, nID8, notifications[1].ID)
	require.Equal(t, nID7, notifications[2].ID)
	require.Equal(t, nID6, notifications[3].ID)
	require.Equal(t, nID5, notifications[4].ID)
	require.Equal(t, nID4, notifications[5].ID)
	require.Equal(t, nID3, notifications[6].ID)
	require.Equal(t, nID2, notifications[7].ID)
	require.Equal(t, nID1, notifications[8].ID)

	_, notifications, err = p.ActivityCenterNotificationsByGroup(
		initialCursor,
		limit,
		ActivityCenterMentions,
		ActivityCenterQueryParamsReadAll,
	)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	require.Equal(t, nID2, notifications[0].ID)

	_, notifications, err = p.ActivityCenterNotificationsByGroup(
		initialCursor,
		limit,
		ActivityCenterReplies,
		ActivityCenterQueryParamsReadAll,
	)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	require.Equal(t, nID3, notifications[0].ID)

	_, notifications, err = p.ActivityCenterNotificationsByGroup(
		initialCursor,
		limit,
		ActivityCenterMembership,
		ActivityCenterQueryParamsReadAll,
	)
	require.NoError(t, err)
	require.Len(t, notifications, 5)
	require.Equal(t, nID8, notifications[0].ID)
	require.Equal(t, nID7, notifications[1].ID)
	require.Equal(t, nID6, notifications[2].ID)
	require.Equal(t, nID5, notifications[3].ID)
	require.Equal(t, nID1, notifications[4].ID)

	_, notifications, err = p.ActivityCenterNotificationsByGroup(
		initialCursor,
		limit,
		ActivityCenterAdmin,
		ActivityCenterQueryParamsReadAll,
	)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	require.Equal(t, nID7, notifications[0].ID)

	_, notifications, err = p.ActivityCenterNotificationsByGroup(
		initialCursor,
		limit,
		ActivityCenterContactRequests,
		ActivityCenterQueryParamsReadAll,
	)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	require.Equal(t, nID4, notifications[0].ID)

	_, notifications, err = p.ActivityCenterNotificationsByGroup(
		initialCursor,
		limit,
		ActivityCenterIdentityVerification,
		ActivityCenterQueryParamsReadAll,
	)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	require.Equal(t, nID9, notifications[0].ID)

	// NOTE: ActivityCenterTransactions & ActivityCenterSystem for further proposes
}

func TestActivityCenterReadUnreadPagination(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	initialOrFinalCursor := ""

	chat := CreatePublicChat("test-chat", &testTimeSource{})
	message := &common.Message{}
	message.Text = "sample text"
	chat.LastMessage = message
	err = p.SaveChat(*chat)
	require.NoError(t, err)

	nID1 := types.HexBytes([]byte("1"))
	nID2 := types.HexBytes([]byte("2"))
	nID3 := types.HexBytes([]byte("3"))
	nID4 := types.HexBytes([]byte("4"))
	nID5 := types.HexBytes([]byte("5"))

	allNotifications := []*ActivityCenterNotification{
		{
			ID:        nID1,
			Type:      ActivityCenterNotificationTypeNewOneToOne,
			ChatID:    chat.ID,
			Timestamp: 1,
		},
		{
			ID:        nID2,
			Type:      ActivityCenterNotificationTypeNewOneToOne,
			ChatID:    chat.ID,
			Timestamp: 1,
		},
		{
			ID:        nID3,
			Type:      ActivityCenterNotificationTypeNewOneToOne,
			ChatID:    chat.ID,
			Timestamp: 1,
		},
		{
			ID:        nID4,
			Type:      ActivityCenterNotificationTypeNewOneToOne,
			ChatID:    chat.ID,
			Timestamp: 1,
		},
		{
			ID:        nID5,
			Type:      ActivityCenterNotificationTypeNewOneToOne,
			ChatID:    chat.ID,
			Timestamp: 1,
		},
	}

	for _, notification := range allNotifications {
		err = p.SaveActivityCenterNotification(notification)
		require.NoError(t, err)
	}

	// Mark the notification as read
	err = p.MarkActivityCenterNotificationsRead([]types.HexBytes{nID2})
	require.NoError(t, err)
	err = p.MarkActivityCenterNotificationsRead([]types.HexBytes{nID4})
	require.NoError(t, err)

	// Fetch UNREAD notifications, first page.
	cursor, notifications, err := p.UnreadActivityCenterNotifications(
		initialOrFinalCursor,
		1,
		[]ActivityCenterType{ActivityCenterNotificationTypeNewOneToOne},
	)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	require.Equal(t, nID5, notifications[0].ID)
	require.NotEmpty(t, cursor)

	// Fetch next pages.
	cursor, notifications, err = p.UnreadActivityCenterNotifications(
		cursor,
		1,
		[]ActivityCenterType{ActivityCenterNotificationTypeNewOneToOne},
	)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	require.Equal(t, nID3, notifications[0].ID)
	require.NotEmpty(t, cursor)

	cursor, notifications, err = p.UnreadActivityCenterNotifications(
		cursor,
		1,
		[]ActivityCenterType{ActivityCenterNotificationTypeNewOneToOne},
	)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	require.Equal(t, nID1, notifications[0].ID)
	require.Empty(t, cursor)

	// Fetch READ notifications, first page.
	cursor, notifications, err = p.ReadActivityCenterNotifications(
		initialOrFinalCursor,
		1,
		[]ActivityCenterType{ActivityCenterNotificationTypeNewOneToOne},
	)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	require.Equal(t, nID4, notifications[0].ID)
	require.NotEmpty(t, cursor)

	// Fetch next page.
	cursor, notifications, err = p.ReadActivityCenterNotifications(
		cursor,
		1,
		[]ActivityCenterType{ActivityCenterNotificationTypeNewOneToOne},
	)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	require.Equal(t, nID2, notifications[0].ID)
	require.Empty(t, cursor)
}

func TestActivityCenterPersistence(t *testing.T) {
	nID1 := types.HexBytes([]byte("1"))
	nID2 := types.HexBytes([]byte("2"))
	nID3 := types.HexBytes([]byte("3"))
	nID4 := types.HexBytes([]byte("4"))

	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	// Check we have no unseen messages
	state, err := p.GetActivityCenterState()
	require.NoError(t, err)
	require.True(t, state.HasSeen)

	chat := CreatePublicChat("test-chat", &testTimeSource{})
	message := &common.Message{}
	message.Text = "sample text"
	chat.LastMessage = message
	err = p.SaveChat(*chat)
	require.NoError(t, err)

	notification := &ActivityCenterNotification{
		ID:        nID1,
		Type:      ActivityCenterNotificationTypeNewOneToOne,
		ChatID:    chat.ID,
		Timestamp: 1,
	}
	err = p.SaveActivityCenterNotification(notification)
	require.NoError(t, err)

	cursor, notifications, err := p.ActivityCenterNotifications("", 2)
	require.NoError(t, err)
	require.Empty(t, cursor)
	require.Len(t, notifications, 1)
	require.Equal(t, chat.ID, notifications[0].ChatID)
	require.Equal(t, message, notifications[0].LastMessage)

	// Add another notification

	notification = &ActivityCenterNotification{
		ID:        nID2,
		Type:      ActivityCenterNotificationTypeNewOneToOne,
		Timestamp: 2,
	}
	err = p.SaveActivityCenterNotification(notification)
	require.NoError(t, err)

	cursor, notifications, err = p.ActivityCenterNotifications("", 1)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	require.NotEmpty(t, cursor)
	require.Equal(t, nID2, notifications[0].ID)

	// fetch next pagination

	cursor, notifications, err = p.ActivityCenterNotifications(cursor, 1)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	require.Empty(t, cursor)
	require.False(t, notifications[0].Read)
	require.Equal(t, nID1, notifications[0].ID)

	// Check count
	count, err := p.UnreadActivityCenterNotificationsCount()
	require.NoError(t, err)
	require.Equal(t, uint64(2), count)

	// Check we have unseen notifications
	state, err = p.GetActivityCenterState()
	require.NoError(t, err)
	require.False(t, state.HasSeen)

	// Mark first one as read
	require.NoError(t, p.MarkActivityCenterNotificationsRead([]types.HexBytes{nID1}))
	count, err = p.UnreadActivityCenterNotificationsCount()
	require.NoError(t, err)
	require.Equal(t, uint64(1), count)

	// Mark first one as unread
	require.NoError(t, p.MarkActivityCenterNotificationsUnread([]types.HexBytes{nID1}))
	count, err = p.UnreadActivityCenterNotificationsCount()
	require.NoError(t, err)
	require.Equal(t, uint64(2), count)

	// Mark all read
	require.NoError(t, p.MarkAllActivityCenterNotificationsRead())
	_, notifications, err = p.ActivityCenterNotifications(cursor, 2)
	require.NoError(t, err)
	require.Len(t, notifications, 2)
	require.Empty(t, cursor)
	require.True(t, notifications[0].Read)
	require.True(t, notifications[1].Read)

	// Check count
	count, err = p.UnreadActivityCenterNotificationsCount()
	require.NoError(t, err)
	require.Equal(t, uint64(0), count)

	// Mark first one as accepted

	notifications, err = p.AcceptActivityCenterNotifications([]types.HexBytes{nID1})
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	_, notifications, err = p.ActivityCenterNotifications("", 2)
	require.NoError(t, err)
	// It should not be returned anymore
	require.Len(t, notifications, 1)

	// Mark last one as dismissed
	require.NoError(t, p.DismissActivityCenterNotifications([]types.HexBytes{nID2}))
	_, notifications, err = p.ActivityCenterNotifications("", 2)
	require.NoError(t, err)

	require.Len(t, notifications, 1)
	require.True(t, notifications[0].Dismissed)

	// Insert new notification
	notification = &ActivityCenterNotification{
		ID:        nID3,
		Type:      ActivityCenterNotificationTypeNewOneToOne,
		Timestamp: 3,
	}
	err = p.SaveActivityCenterNotification(notification)
	require.NoError(t, err)

	_, notifications, err = p.ActivityCenterNotifications("", 2)
	require.NoError(t, err)

	require.Len(t, notifications, 2)

	// Insert new notification
	notification = &ActivityCenterNotification{
		ID:        nID4,
		Type:      ActivityCenterNotificationTypeNewOneToOne,
		Timestamp: 4,
	}
	err = p.SaveActivityCenterNotification(notification)
	require.NoError(t, err)

	// Mark all as dismissed
	require.NoError(t, p.DismissAllActivityCenterNotifications())
	_, notifications, err = p.ActivityCenterNotifications("", 2)
	require.NoError(t, err)

	require.Len(t, notifications, 2)
	require.True(t, notifications[0].Dismissed)
	require.True(t, notifications[1].Dismissed)

	// Check we still have unseen notifications
	state, err = p.GetActivityCenterState()
	require.NoError(t, err)
	require.False(t, state.HasSeen)

	// Mark as unseen
	err = p.MarkAsSeenActivityCenterNotifications()
	require.NoError(t, err)

	// Check we have no unseen notifications
	state, err = p.GetActivityCenterState()
	require.NoError(t, err)
	require.True(t, state.HasSeen)
}

func TestSaveCommunityChat(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	identity := &protobuf.ChatIdentity{
		DisplayName:           "community-chat-name",
		Description:           "community-chat-name-description",
		FirstMessageTimestamp: 1,
	}
	permissions := &protobuf.CommunityPermissions{
		Access: protobuf.CommunityPermissions_NO_MEMBERSHIP,
	}

	communityChat := &protobuf.CommunityChat{
		Identity:    identity,
		Permissions: permissions,
	}

	chat := CreateCommunityChat("test-or-gid", "test-chat-id", communityChat, &testTimeSource{})
	chat.LastMessage = &common.Message{}
	err = p.SaveChat(*chat)
	require.NoError(t, err)

	retrievedChat, err := p.Chat(chat.ID)
	require.NoError(t, err)
	require.Equal(t, chat, retrievedChat)
}

func TestHasPendingNotificationsForChatSanityCheck(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	result, err := p.HasPendingNotificationsForChat("test-chat-id")
	require.NoError(t, err)
	require.False(t, result)
}

func TestSaveDiscordMessageAuthor(t *testing.T) {

	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	testAuthor := &protobuf.DiscordMessageAuthor{
		Id:                 "1",
		Name:               "Testuser",
		Discriminator:      "1234",
		Nickname:           "User",
		AvatarUrl:          "http://example.com/profile.jpg",
		AvatarImagePayload: []byte{1, 2, 3},
	}

	require.NoError(t, p.SaveDiscordMessageAuthor(testAuthor))

	exists, err := p.HasDiscordMessageAuthor("1")
	require.NoError(t, err)
	require.True(t, exists)
	author, err := p.GetDiscordMessageAuthorByID("1")
	require.NoError(t, err)
	require.Equal(t, author.Id, testAuthor.Id)
	require.Equal(t, author.Name, testAuthor.Name)
}

func TestGetDiscordMessageAuthorImagePayloadByID(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	testAuthor := &protobuf.DiscordMessageAuthor{
		Id:                 "1",
		Name:               "Testuser",
		Discriminator:      "1234",
		Nickname:           "User",
		AvatarUrl:          "http://example.com/profile.jpg",
		AvatarImagePayload: []byte{1, 2, 3},
	}

	require.NoError(t, p.SaveDiscordMessageAuthor(testAuthor))

	payload, err := p.GetDiscordMessageAuthorImagePayloadByID("1")
	require.NoError(t, err)

	require.Equal(t, testAuthor.AvatarImagePayload, payload)
}

func TestSaveDiscordMessage(t *testing.T) {

	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	require.NoError(t, p.SaveDiscordMessage(&protobuf.DiscordMessage{
		Id:        "1",
		Type:      "Default",
		Timestamp: "123456",
		Content:   "This is the message",
		Author: &protobuf.DiscordMessageAuthor{
			Id: "2",
		},
		Reference: &protobuf.DiscordMessageReference{},
	}))

	require.NoError(t, err)
}

func TestSaveDiscordMessages(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	for i := 0; i < 10; i++ {
		id := strconv.Itoa(i)
		err := insertMinimalDiscordMessage(p, id, id)
		require.NoError(t, err)

		m, err := p.MessageByID(id)
		require.NoError(t, err)
		dm := m.GetDiscordMessage()
		require.NotNil(t, dm)
		require.EqualValues(t, id, dm.Id)
		require.EqualValues(t, "2", dm.Author.Id)
	}
}

func TestUpdateDiscordMessageAuthorImage(t *testing.T) {

	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	require.NoError(t, p.SaveDiscordMessageAuthor(&protobuf.DiscordMessageAuthor{
		Id:            "1",
		Name:          "Testuser",
		Discriminator: "1234",
		Nickname:      "User",
		AvatarUrl:     "http://example.com/profile.jpg",
	}))

	exists, err := p.HasDiscordMessageAuthor("1")
	require.NoError(t, err)
	require.True(t, exists)

	err = p.UpdateDiscordMessageAuthorImage("1", []byte{0, 1, 2, 3})
	require.NoError(t, err)
	payload, err := p.GetDiscordMessageAuthorImagePayloadByID("1")
	require.NoError(t, err)
	require.Equal(t, []byte{0, 1, 2, 3}, payload)
}

func TestSaveHashRatchetMessage(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	groupID1 := []byte("group-id-1")
	groupID2 := []byte("group-id-2")
	var keyID uint32 = 3

	message1 := &types.Message{
		Hash:      []byte{1},
		Sig:       []byte{2},
		TTL:       1,
		Timestamp: 2,
		Payload:   []byte{3},
	}

	require.NoError(t, p.SaveHashRatchetMessage(groupID1, keyID, message1))

	message2 := &types.Message{
		Hash:      []byte{2},
		Sig:       []byte{2},
		TTL:       1,
		Topic:     types.BytesToTopic([]byte{5}),
		Timestamp: 2,
		Payload:   []byte{3},
		Dst:       []byte{4},
		P2P:       true,
	}

	require.NoError(t, p.SaveHashRatchetMessage(groupID2, keyID, message2))

	fetchedMessages, err := p.GetHashRatchetMessages(groupID1, keyID)
	require.NoError(t, err)
	require.NotNil(t, fetchedMessages)
	require.Len(t, fetchedMessages, 1)
	require.Equal(t, fetchedMessages[0], message1)
}
