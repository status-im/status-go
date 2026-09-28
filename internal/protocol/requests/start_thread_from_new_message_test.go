package requests

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/protocol/common"
)

func TestStartThreadFromNewMessageValidate(t *testing.T) {
	validMessage := func() *common.Message {
		message := common.NewMessage()
		message.ChatId = "chat-id"
		return message
	}

	testCases := []struct {
		name    string
		request *StartThreadFromNewMessage
		err     error
	}{
		{name: "nil request", request: nil, err: ErrStartThreadFromNewMessageInvalidMessage},
		{name: "nil message", request: &StartThreadFromNewMessage{}, err: ErrStartThreadFromNewMessageInvalidMessage},
		{name: "nil chat message", request: &StartThreadFromNewMessage{Message: &common.Message{}}, err: ErrStartThreadFromNewMessageInvalidMessage},
		{name: "empty chat id", request: &StartThreadFromNewMessage{Message: common.NewMessage()}, err: ErrStartThreadFromNewMessageInvalidChatID},
		{name: "prepopulated thread id", request: &StartThreadFromNewMessage{Message: validMessage()}, err: ErrStartThreadFromNewMessageInvalidThreadID},
		{name: "prepopulated response to", request: &StartThreadFromNewMessage{Message: validMessage()}, err: ErrStartThreadFromNewMessageInvalidResponseTo},
		{name: "valid", request: &StartThreadFromNewMessage{Message: validMessage()}},
	}

	threadID := "existing-thread"
	testCases[4].request.Message.ThreadId = &threadID
	testCases[5].request.Message.ResponseTo = "existing-parent"

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.request.Validate()
			if testCase.err == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, testCase.err)
		})
	}
}

func TestStartThreadFromNewMessageUnmarshalJSON(t *testing.T) {
	var request StartThreadFromNewMessage
	err := json.Unmarshal([]byte(`{
		"message": {
			"chatId": "chat-id",
			"text": "full message",
			"ensName": "alice.eth",
			"linkPreviews": [{"url": "https://example.com", "title": "Example"}],
			"statusLinkPreviews": [{"url": "https://status.app/u/alice"}]
		},
		"threadName": "Thread name"
	}`), &request)
	require.NoError(t, err)
	require.NotNil(t, request.Message)
	require.Equal(t, "chat-id", request.Message.ChatId)
	require.Equal(t, "full message", request.Message.Text)
	require.Equal(t, "alice.eth", request.Message.EnsName)
	require.Len(t, request.Message.LinkPreviews, 1)
	require.Equal(t, "https://example.com", request.Message.LinkPreviews[0].URL)
	require.Len(t, request.Message.StatusLinkPreviews, 1)
	require.Equal(t, "https://status.app/u/alice", request.Message.StatusLinkPreviews[0].URL)
	require.Equal(t, "Thread name", request.ThreadName)
	require.NoError(t, request.Validate())
}
