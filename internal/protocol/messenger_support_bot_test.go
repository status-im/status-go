package protocol

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/db/multiaccounts/settings"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	"github.com/status-im/status-go/internal/protocol/requests"
	"github.com/status-im/status-go/pkg/messaging"
)

func TestSendSupportBotContactRequest(t *testing.T) {
	testCases := []struct {
		name    string
		state   int64
		message string
	}{
		{
			name:    "existing user",
			state:   settings.SupportBotContactRequestStatePendingExisting,
			message: supportBotExistingUserMessage,
		},
		{
			name:    "new user",
			state:   settings.SupportBotContactRequestStatePendingNew,
			message: supportBotNewUserMessage,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			m := newSupportBotTestMessenger(t)
			require.NoError(t, m.settings.SaveSettingField(settings.SupportBotContactRequestState, testCase.state))

			require.NoError(t, m.sendSupportBotContactRequest())

			chatID, err := requests.ConvertCompressedToLegacyKey(testPK)
			require.NoError(t, err)
			messages, _, err := m.persistence.MessageByChatID(chatID, "", 10)
			require.NoError(t, err)
			contactRequest := FindFirstByContentType(messages, protobuf.ChatMessage_CONTACT_REQUEST)
			require.NotNil(t, contactRequest)
			require.Equal(t, testCase.message, contactRequest.Text)

			state, err := m.settings.SupportBotContactRequestState()
			require.NoError(t, err)
			require.Equal(t, settings.SupportBotContactRequestStateDone, state)

			require.NoError(t, m.settings.SaveSettingField(settings.SupportBotContactRequestState, testCase.state))
			require.NoError(t, m.sendSupportBotContactRequest())

			messagesAfterRetry, _, err := m.persistence.MessageByChatID(chatID, "", 10)
			require.NoError(t, err)
			require.Len(t, messagesAfterRetry, len(messages))

			state, err = m.settings.SupportBotContactRequestState()
			require.NoError(t, err)
			require.Equal(t, settings.SupportBotContactRequestStateDone, state)
		})
	}
}

func TestMessengerStartSendsSupportBotContactRequest(t *testing.T) {
	messagingEnv, err := messaging.NewTestMessagingEnvironment()
	require.NoError(t, err)
	require.NoError(t, messagingEnv.Setup(t))

	m, err := newRunningTestMessenger(t, messagingEnv, testMessengerConfig{
		extraOptions: []Option{
			WithEnableSupportBotContactRequest(true),
			withSupportBotChatKey(testPK),
		},
	})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		state, err := m.settings.SupportBotContactRequestState()
		return err == nil && state == settings.SupportBotContactRequestStateDone
	}, 5*time.Second, 50*time.Millisecond)

	chatID, err := requests.ConvertCompressedToLegacyKey(testPK)
	require.NoError(t, err)
	messages, _, err := m.persistence.MessageByChatID(chatID, "", 10)
	require.NoError(t, err)
	contactRequest := FindFirstByContentType(messages, protobuf.ChatMessage_CONTACT_REQUEST)
	require.NotNil(t, contactRequest)
	require.Equal(t, supportBotExistingUserMessage, contactRequest.Text)
}

func TestSupportBotContactRequestMessage(t *testing.T) {
	message, err := supportBotContactRequestMessage(settings.SupportBotContactRequestStatePendingExisting)
	require.NoError(t, err)
	require.Equal(t, supportBotExistingUserMessage, message)

	message, err = supportBotContactRequestMessage(settings.SupportBotContactRequestStatePendingNew)
	require.NoError(t, err)
	require.Equal(t, supportBotNewUserMessage, message)

	_, err = supportBotContactRequestMessage(settings.SupportBotContactRequestStateDone)
	require.Error(t, err)
}

func newSupportBotTestMessenger(t *testing.T) *Messenger {
	t.Helper()

	messagingEnv, err := messaging.NewTestMessagingEnvironment()
	require.NoError(t, err)
	require.NoError(t, messagingEnv.Setup(t))

	m, err := newRunningTestMessenger(t, messagingEnv, testMessengerConfig{
		extraOptions: []Option{withSupportBotChatKey(testPK)},
	})
	require.NoError(t, err)
	return m
}
