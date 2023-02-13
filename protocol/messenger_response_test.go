package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/protocol/common"
)

func TestMessengerResponseMergeChats(t *testing.T) {
	chat1 := &Chat{ID: "1"}
	modifiedChat1 := &Chat{ID: "1", Name: "name"}
	chat2 := &Chat{ID: "3"}
	response1 := &MessengerResponse{}
	response1.AddChat(chat1)

	response2 := &MessengerResponse{}
	response2.AddChats([]*Chat{modifiedChat1, chat2})

	require.NoError(t, response1.Merge(response2))

	require.Len(t, response1.Chats(), 2)
	require.Equal(t, modifiedChat1, response1.chats[modifiedChat1.ID])
	require.Equal(t, chat2, response1.chats[chat2.ID])
}

func TestMessengerResponseMergeMessages(t *testing.T) {
	message1 := &common.Message{ID: "1"}
	modifiedMessage1 := &common.Message{ID: "1", From: "name"}
	message2 := &common.Message{ID: "3"}
	response1 := &MessengerResponse{}
	response1.AddMessage(message1)

	response2 := &MessengerResponse{}
	response2.AddMessage(modifiedMessage1)
	response2.AddMessage(message2)

	require.NoError(t, response1.Merge(response2))

	require.Len(t, response1.Messages(), 2)
	messages := response1.Messages()
	if messages[0].ID == modifiedMessage1.ID {
		require.Equal(t, modifiedMessage1, messages[0])
		require.Equal(t, message2, messages[1])
	} else {
		require.Equal(t, modifiedMessage1, messages[1])
		require.Equal(t, message2, messages[0])
	}

}

func TestMessengerResponseMergeNotImplemented(t *testing.T) {
	response1 := &MessengerResponse{}

	response2 := &MessengerResponse{
		Invitations: []*GroupChatInvitation{{}},
	}
	require.Error(t, response1.Merge(response2))

}
