package protocol

import (
	"database/sql"
	"io/ioutil"
	"math"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/stretchr/testify/require"
)

func TestTableUserMessagesAllFieldsCount(t *testing.T) {
	db := sqlitePersistence{}
	expected := len(strings.Split(db.tableUserMessagesLegacyAllFields(), ","))
	require.Equal(t, expected, db.tableUserMessagesLegacyAllFieldsCount())
}

func TestSaveMessages(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := sqlitePersistence{db: db}

	for i := 0; i < 10; i++ {
		id := strconv.Itoa(i)
		err := insertMinimalMessage(p, id)
		require.NoError(t, err)

		m, err := p.MessageByID(id)
		require.NoError(t, err)
		require.EqualValues(t, id, m.ID)
	}
}

func TestMessageByID(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := sqlitePersistence{db: db}
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
	p := sqlitePersistence{db: db}

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
	p := sqlitePersistence{db: db}
	chatID := "super-chat"
	count := 1000
	pageSize := 50

	var messages []*Message
	for i := 0; i < count; i++ {
		messages = append(messages, &Message{
			ID:          strconv.Itoa(i),
			LocalChatID: chatID,
			ChatMessage: protobuf.ChatMessage{
				Clock: uint64(i),
			},
			From: "me",
		})

		// Add some other chats.
		if count%5 == 0 {
			messages = append(messages, &Message{
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
		messages = append(messages, &Message{
			ID:          strconv.Itoa(count*2 + i),
			LocalChatID: chatID,
			ChatMessage: protobuf.ChatMessage{
				Clock: uint64(i),
			},

			From: "me",
		})
	}

	err = p.SaveMessagesLegacy(messages)
	require.NoError(t, err)

	var (
		result []*Message
		cursor string
		iter   int
	)
	for {
		var (
			items []*Message
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

func TestMessageReplies(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := sqlitePersistence{db: db}
	chatID := "super-chat"
	message1 := &Message{
		ID:          "id-1",
		LocalChatID: chatID,
		ChatMessage: protobuf.ChatMessage{
			Text:  "content-1",
			Clock: uint64(1),
		},
		From: "1",
	}
	message2 := &Message{
		ID:          "id-2",
		LocalChatID: chatID,
		ChatMessage: protobuf.ChatMessage{
			Text:       "content-2",
			Clock:      uint64(2),
			ResponseTo: "id-1",
		},

		From: "2",
	}

	message3 := &Message{
		ID:          "id-3",
		LocalChatID: chatID,
		ChatMessage: protobuf.ChatMessage{
			Text:       "content-3",
			Clock:      uint64(3),
			ResponseTo: "non-existing",
		},
		From: "3",
	}

	messages := []*Message{message1, message2, message3}

	err = p.SaveMessagesLegacy(messages)
	require.NoError(t, err)

	retrievedMessages, _, err := p.MessageByChatID(chatID, "", 10)
	require.NoError(t, err)

	require.Equal(t, "non-existing", retrievedMessages[0].ResponseTo)
	require.Nil(t, retrievedMessages[0].QuotedMessage)

	require.Equal(t, "id-1", retrievedMessages[1].ResponseTo)
	require.Equal(t, &QuotedMessage{From: "1", Text: "content-1"}, retrievedMessages[1].QuotedMessage)

	require.Equal(t, "", retrievedMessages[2].ResponseTo)
	require.Nil(t, retrievedMessages[2].QuotedMessage)
}

func TestMessageByChatIDWithTheSameClocks(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := sqlitePersistence{db: db}
	chatID := "super-chat"
	clockValues := []uint64{10, 10, 9, 9, 9, 11, 12, 11, 100000, 6, 4, 5, 5, 5, 5}
	count := len(clockValues)
	pageSize := 2

	var messages []*Message

	for i, clock := range clockValues {
		messages = append(messages, &Message{
			ID:          strconv.Itoa(i),
			LocalChatID: chatID,
			ChatMessage: protobuf.ChatMessage{
				Clock: clock,
			},
			From: "me",
		})
	}

	err = p.SaveMessagesLegacy(messages)
	require.NoError(t, err)

	var (
		result []*Message
		cursor string
		iter   int
	)
	for {
		var (
			items []*Message
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
	p := sqlitePersistence{db: db}
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
	p := sqlitePersistence{db: db}

	err = insertMinimalMessage(p, "1")
	require.NoError(t, err)

	err = insertMinimalMessage(p, "2")
	require.NoError(t, err)

	m, _, err := p.MessageByChatID("chat-id", "", 10)
	require.NoError(t, err)
	require.Equal(t, 2, len(m))

	err = p.DeleteMessagesByChatID("chat-id")
	require.NoError(t, err)

	m, _, err = p.MessageByChatID("chat-id", "", 10)
	require.NoError(t, err)
	require.Equal(t, 0, len(m))

}

func TestMarkMessageSeen(t *testing.T) {
	chatID := "test-chat"
	db, err := openTestDB()
	require.NoError(t, err)
	p := sqlitePersistence{db: db}
	id := "1"

	err = insertMinimalMessage(p, id)
	require.NoError(t, err)

	m, err := p.MessageByID(id)
	require.NoError(t, err)
	require.False(t, m.Seen)

	err = p.MarkMessagesSeen(chatID, []string{m.ID})
	require.NoError(t, err)

	m, err = p.MessageByID(id)
	require.NoError(t, err)
	require.True(t, m.Seen)
}

func TestUpdateMessageOutgoingStatus(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := sqlitePersistence{db: db}
	id := "1"

	err = insertMinimalMessage(p, id)
	require.NoError(t, err)

	err = p.UpdateMessageOutgoingStatus(id, "new-status")
	require.NoError(t, err)

	m, err := p.MessageByID(id)
	require.NoError(t, err)
	require.Equal(t, "new-status", m.OutgoingStatus)
}

func TestSetContactGeneratedData(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := sqlitePersistence{db: db}
	existingContact := Contact{
		ID:          "contact-one",
		Address:     "contact-address",
		Name:        "contact-name",
		Photo:       "contact-photo",
		LastUpdated: 20,
		SystemTags:  []string{"1", "2"},
		DeviceInfo: []ContactDeviceInfo{
			ContactDeviceInfo{
				InstallationID: "1",
				Timestamp:      2,
				FCMToken:       "token",
			},
			ContactDeviceInfo{
				InstallationID: "2",
				Timestamp:      3,
				FCMToken:       "token-2",
			},
		},
		TributeToTalk: "talk",
	}

	existingContactUpdate := Contact{
		ID:      "contact-one",
		Address: "contact-address",
		Alias:   "generated-name-one",
	}

	nonExistingContactUpdate := Contact{
		ID:      "contact-two",
		Address: "contact-address",
		Alias:   "generated-name-two",
	}

	err = p.SaveContact(existingContact, nil)
	require.NoError(t, err)

	err = p.SetContactsGeneratedData([]*Contact{&existingContactUpdate, &nonExistingContactUpdate}, nil)
	require.NoError(t, err)

	allContacts, err := p.Contacts()
	require.NoError(t, err)

	require.Equal(t, 2, len(allContacts))

	// Make sure it has not been modified
	require.Equal(t, int64(20), allContacts[0].LastUpdated)

	// Ensure new contact has been saved
	require.Equal(t, "contact-two", allContacts[1].ID)
}

func openTestDB() (*sql.DB, error) {
	dbPath, err := ioutil.TempFile("", "")
	if err != nil {
		return nil, err
	}
	return sqlite.Open(dbPath.Name(), "")
}

func insertMinimalMessage(p sqlitePersistence, id string) error {
	return p.SaveMessagesLegacy([]*Message{&Message{
		ID:          id,
		LocalChatID: "chat-id",
		ChatMessage: protobuf.ChatMessage{Text: "some-text"},
		From:        "me",
	}})
}
