package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/protocol/common"
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

func TestMessengerResponseMergeThreads(t *testing.T) {
	thread1 := &Thread{ThreadID: "thread-1", ChatID: "chat-1", Name: "First"}
	modifiedThread1 := &Thread{ThreadID: "thread-1", ChatID: "chat-1", Name: "Renamed"}
	thread2 := &Thread{ThreadID: "thread-2", ChatID: "chat-1", Name: "Second"}

	response1 := &MessengerResponse{}
	response1.AddThread(thread1)

	response2 := &MessengerResponse{}
	response2.AddThread(modifiedThread1)
	response2.AddThread(thread2)

	require.NoError(t, response1.Merge(response2))

	require.Len(t, response1.Threads(), 2)
	require.Equal(t, modifiedThread1, response1.threads[threadKey(modifiedThread1.ChatID, modifiedThread1.ThreadID)])
	require.Equal(t, thread2, response1.threads[threadKey(thread2.ChatID, thread2.ThreadID)])
}

func TestMessengerResponseMarshalJSONIncludesThreads(t *testing.T) {
	response := &MessengerResponse{}
	response.AddThread(&Thread{ThreadID: "thread-1", ChatID: "chat-1", Name: "Thread"})

	encoded, err := response.MarshalJSON()
	require.NoError(t, err)
	require.Contains(t, string(encoded), `"threads"`)
}

func TestMessengerResponseMergeNotImplemented(t *testing.T) {
	response1 := &MessengerResponse{}

	response2 := &MessengerResponse{
		Invitations: []*GroupChatInvitation{{}},
	}
	require.Error(t, response1.Merge(response2))

}
