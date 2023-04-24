package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
)

func TestUpdateChatUnviewedCounts(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	chat := CreatePublicChat(testPublicChatID, &testTimeSource{})
	err = p.SaveChat(*chat)
	require.NoError(t, err)

	unviewedMessages, unviewedMentions, firstUnviewedMessageID, err := p.getChatUnviewedCounts(chat.ID, nil)
	require.NoError(t, err)
	require.Equal(t, uint(0), unviewedMessages)
	require.Equal(t, uint(0), unviewedMentions)
	require.Equal(t, FirstUnviewedMessageNone, firstUnviewedMessageID)

	retrievedChat, err := p.Chat(chat.ID)
	require.NoError(t, err)
	require.Equal(t, uint(0), retrievedChat.UnviewedMessagesCount)
	require.Equal(t, uint(0), retrievedChat.UnviewedMentionsCount)
	require.Nil(t, retrievedChat.FirstUnviewedMessage)

	messages := []*common.Message{
		{
			ChatMessage: protobuf.ChatMessage{
				Clock: 1,
			},
			ID:           "1",
			LocalChatID:  chat.ID,
			Seen:         true,
			Mentioned:    true,
			Replied:      false,
			Deleted:      false,
			DeletedForMe: false,
		},
		{
			ChatMessage: protobuf.ChatMessage{
				Clock: 2,
			},
			ID:           "2",
			LocalChatID:  chat.ID,
			Seen:         false,
			Mentioned:    false,
			Replied:      false,
			Deleted:      true,
			DeletedForMe: false,
		},
		{
			ChatMessage: protobuf.ChatMessage{
				Clock: 3,
			},
			ID:           "3",
			LocalChatID:  chat.ID,
			Seen:         false,
			Mentioned:    true,
			Replied:      false,
			Deleted:      false,
			DeletedForMe: false,
		},
		{
			ChatMessage: protobuf.ChatMessage{
				Clock: 4,
			},
			ID:           "4",
			LocalChatID:  chat.ID,
			Seen:         false,
			Mentioned:    false,
			Replied:      false,
			Deleted:      false,
			DeletedForMe: false,
		},
	}

	err = p.SaveMessages(messages)
	require.NoError(t, err)

	err = p.updateChatUnviewedCounts(chat.ID, nil)
	require.NoError(t, err)

	unviewedMessages, unviewedMentions, firstUnviewedMessageID, err = p.getChatUnviewedCounts(chat.ID, nil)
	require.NoError(t, err)
	require.Equal(t, uint(2), unviewedMessages)
	require.Equal(t, uint(1), unviewedMentions)
	require.Equal(t, "3", firstUnviewedMessageID)

	retrievedChat, err = p.Chat(chat.ID)
	require.NoError(t, err)
	require.Equal(t, uint(2), retrievedChat.UnviewedMessagesCount)
	require.Equal(t, uint(1), retrievedChat.UnviewedMentionsCount)
	require.NotNil(t, retrievedChat.FirstUnviewedMessage)
	require.Equal(t, "3", retrievedChat.FirstUnviewedMessage.ID)
}
