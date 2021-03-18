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
	p := NewSQLitePersistence(db)

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
	p := NewSQLitePersistence(db)

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

func TestMessageByID(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := NewSQLitePersistence(db)
	id := "1"

	err = insertMinimalMessage(p, id)
	require.NoError(t, err)

	m, err := p.MessageByID(id)
	require.NoError(t, err)
	require.EqualValues(t, id, m.ID)
}

func TestMessagesExist(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := NewSQLitePersistence(db)

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
	p := NewSQLitePersistence(db)
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
			From: "me",
		})

		// Add some other chats.
		if count%5 == 0 {
			messages = append(messages, &common.Message{
				ID:          strconv.Itoa(count + i),
				LocalChatID: "other-chat",
				ChatMessage: protobuf.ChatMessage{
					Clock: uint64(i),
				},

				From: "me",
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

			From: "me",
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

func TestMessageReplies(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := NewSQLitePersistence(db)
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

	messages := []*common.Message{message1, message2, message3}

	err = p.SaveMessages(messages)
	require.NoError(t, err)

	retrievedMessages, _, err := p.MessageByChatID(chatID, "", 10)
	require.NoError(t, err)

	require.Equal(t, "non-existing", retrievedMessages[0].ResponseTo)
	require.Nil(t, retrievedMessages[0].QuotedMessage)

	require.Equal(t, "id-1", retrievedMessages[1].ResponseTo)
	require.Equal(t, &common.QuotedMessage{From: "1", Text: "content-1"}, retrievedMessages[1].QuotedMessage)

	require.Equal(t, "", retrievedMessages[2].ResponseTo)
	require.Nil(t, retrievedMessages[2].QuotedMessage)
}

func TestMessageByChatIDWithTheSameClocks(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := NewSQLitePersistence(db)
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
			From: "me",
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
	p := NewSQLitePersistence(db)
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
	p := NewSQLitePersistence(db)

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
	p := NewSQLitePersistence(db)
	id := "1"

	err = insertMinimalMessage(p, id)
	require.NoError(t, err)

	m, err := p.MessageByID(id)
	require.NoError(t, err)
	require.False(t, m.Seen)

	count, err := p.MarkMessagesSeen(chatID, []string{m.ID})
	require.NoError(t, err)
	require.Equal(t, uint64(1), count)

	m, err = p.MessageByID(id)
	require.NoError(t, err)
	require.True(t, m.Seen)
}

func TestUpdateMessageOutgoingStatus(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := NewSQLitePersistence(db)
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
	p := NewSQLitePersistence(db)

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

func TestExpiredEmojiReactionsIDs(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := NewSQLitePersistence(db)

	ids, err := p.ExpiredEmojiReactionsIDs(emojiResendMaxCount)
	require.NoError(t, err)
	require.Empty(t, ids)

	//save expired chat message
	rawChatMessage := minimalRawMessage("chat-message-id", protobuf.ApplicationMetadataMessage_CHAT_MESSAGE)
	rawChatMessage.Sent = false
	err = p.SaveRawMessage(rawChatMessage)
	require.NoError(t, err)

	//make sure it doesn't apper in an expired emoji reactions list
	ids, err = p.ExpiredEmojiReactionsIDs(emojiResendMaxCount)
	require.NoError(t, err)
	require.Empty(t, ids)

	//save expired emoji message
	rawEmojiReaction := minimalRawMessage("emoji-message-id", protobuf.ApplicationMetadataMessage_EMOJI_REACTION)
	rawEmojiReaction.Sent = false
	err = p.SaveRawMessage(rawEmojiReaction)
	require.NoError(t, err)

	//make sure it appered in expired emoji reactions list
	ids, err = p.ExpiredEmojiReactionsIDs(emojiResendMaxCount)
	require.NoError(t, err)
	require.Equal(t, 1, len(ids))

	//save non-expired emoji reaction
	rawEmojiReaction2 := minimalRawMessage("emoji-message-id2", protobuf.ApplicationMetadataMessage_EMOJI_REACTION)
	rawEmojiReaction2.Sent = true
	err = p.SaveRawMessage(rawEmojiReaction2)
	require.NoError(t, err)

	//make sure it didn't appear in expired emoji reactions list
	ids, err = p.ExpiredEmojiReactionsIDs(emojiResendMaxCount)
	require.NoError(t, err)
	require.Equal(t, 1, len(ids))
}

func TestPersistenceEmojiReactions(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := NewSQLitePersistence(db)
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
	dbPath, err := ioutil.TempFile("", "")
	if err != nil {
		return nil, err
	}
	return sqlite.Open(dbPath.Name(), "")
}

func insertMinimalMessage(p *sqlitePersistence, id string) error {
	return p.SaveMessages([]*common.Message{{
		ID:          id,
		LocalChatID: testPublicChatID,
		ChatMessage: protobuf.ChatMessage{Text: "some-text"},
		From:        "me",
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
	p := NewSQLitePersistence(db)
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
	p := NewSQLitePersistence(db)

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
	p := NewSQLitePersistence(db)

	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	pkString := types.EncodeHex(crypto.FromECDSAPub(&key.PublicKey))

	message := common.Message{
		ID:          "1",
		LocalChatID: chatID,
		ChatMessage: protobuf.ChatMessage{Text: "some-text"},
		From:        "me",
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
	p := NewSQLitePersistence(db)

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

func TestSaveContactIdentityImage(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := NewSQLitePersistence(db)

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

	images := &protobuf.ChatIdentity{
		Clock:  1,
		Images: identityImages,
	}

	result, err := p.SaveContactChatIdentity(contactID, images)
	require.NoError(t, err)
	require.True(t, result)

	// Save again same clock, it should return false
	result, err = p.SaveContactChatIdentity(contactID, images)
	require.NoError(t, err)
	require.False(t, result)

	contacts, err := p.Contacts()
	require.NoError(t, err)
	require.Len(t, contacts, 1)

	require.Len(t, contacts[0].Images, 2)
}

func TestSaveLinks(t *testing.T) {
	chatID := testPublicChatID
	db, err := openTestDB()
	require.NoError(t, err)
	p := NewSQLitePersistence(db)

	require.NoError(t, err)

	message := common.Message{
		ID:          "1",
		LocalChatID: chatID,
		ChatMessage: protobuf.ChatMessage{Text: "some-text"},
		From:        "me",
		Links:       []string{"https://github.com/status-im/status-react"},
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
	p := NewSQLitePersistence(db)
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
	p := NewSQLitePersistence(db)
	publicChatID := "public-chat-id"
	var currentClockValue uint64 = 10

	timesource := &testTimeSource{}
	lastMessage := common.Message{
		ID:          "0x01",
		LocalChatID: publicChatID,
		ChatMessage: protobuf.ChatMessage{Text: "some-text"},
		From:        "me",
	}
	lastMessage.Clock = 20

	require.NoError(t, p.SaveMessages([]*common.Message{&lastMessage}))

	publicChat := CreatePublicChat(publicChatID, timesource)
	publicChat.LastMessage = &lastMessage
	publicChat.UnviewedMessagesCount = 1

	err = p.DeactivateChat(publicChat, currentClockValue)

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
	p := NewSQLitePersistence(db)
	var currentClockValue uint64 = 10

	timesource := &testTimeSource{}

	chat := CreateOneToOneChat(pkString, &key.PublicKey, timesource)

	lastMessage := common.Message{
		ID:          "0x01",
		LocalChatID: chat.ID,
		ChatMessage: protobuf.ChatMessage{Text: "some-text"},
		From:        "me",
	}
	lastMessage.Clock = 20

	require.NoError(t, p.SaveMessages([]*common.Message{&lastMessage}))

	chat.LastMessage = &lastMessage
	chat.UnviewedMessagesCount = 1

	err = p.DeactivateChat(chat, currentClockValue)

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
	p := NewSQLitePersistence(db)

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
	p := NewSQLitePersistence(db)

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
