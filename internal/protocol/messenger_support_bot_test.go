package protocol

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/db/multiaccounts/settings"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	"github.com/status-im/status-go/internal/protocol/requests"
	"github.com/status-im/status-go/pkg/messaging"
)

func TestSupportBotChatKeyFromEnv(t *testing.T) {
	t.Setenv("STATUS_SUPPORT_BOT_CHAT_KEY", testPK)
	require.Equal(t, testPK, supportBotChatKeyFromEnv())

	require.NoError(t, os.Unsetenv("STATUS_SUPPORT_BOT_CHAT_KEY"))
	require.Equal(t, defaultSupportBotChatKey, supportBotChatKeyFromEnv())
}

func TestSendSupportBotContactRequest(t *testing.T) {
	testCases := []struct {
		name                  string
		state                 int64
		expectsContactRequest bool
	}{
		{
			name:                  "existing user",
			state:                 settings.SupportBotContactRequestStatePendingExisting,
			expectsContactRequest: false,
		},
		{
			name:                  "new user",
			state:                 settings.SupportBotContactRequestStatePendingNew,
			expectsContactRequest: true,
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
			if testCase.expectsContactRequest {
				require.NotNil(t, contactRequest)
				require.Equal(t, supportBotNewUserMessage, contactRequest.Text)
			} else {
				require.Nil(t, contactRequest)
			}

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

func TestMessengerStartSkipsExistingUserSupportBotContactRequest(t *testing.T) {
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
	require.Nil(t, contactRequest)
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
