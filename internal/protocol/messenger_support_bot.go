package protocol

import (
	"fmt"
	"os"
	"strings"

	"github.com/status-im/status-go/internal/db/multiaccounts/settings"
	"github.com/status-im/status-go/internal/protocol/requests"
)

const (
	supportBotExistingUserMessage = "Hi, I just upgraded to the latest version of Status."
	supportBotNewUserMessage      = "Hi, I'm a new Status user."
)

var supportBotChatKey = os.Getenv("STATUS_SUPPORT_BOT_CHAT_KEY")

func (m *Messenger) sendSupportBotContactRequest() error {
	if m.shouldAbortSupportBotContactRequest() {
		return nil
	}

	state, err := m.settings.SupportBotContactRequestState()
	if err != nil {
		return err
	}
	if state == settings.SupportBotContactRequestStateDone {
		return nil
	}

	chatID, err := requests.ConvertCompressedToLegacyKey(m.config.supportBotChatKey)
	if err != nil {
		return err
	}
	if strings.EqualFold(chatID, m.myHexIdentity()) {
		return m.markSupportBotContactRequestDone()
	}
	if contact, ok := m.allContacts.Load(chatID); ok && contact.Added() {
		return m.markSupportBotContactRequestDone()
	}

	message, err := supportBotContactRequestMessage(state)
	if err != nil {
		return err
	}
	response, err := m.SendContactRequest(m.ctx, &requests.SendContactRequest{
		ID:      chatID,
		Message: message,
	})
	if err != nil {
		return err
	}
	if err := m.markSupportBotContactRequestDone(); err != nil {
		return err
	}

	m.PublishMessengerResponse(response)
	return nil
}

func (m *Messenger) shouldAbortSupportBotContactRequest() bool {
	select {
	case <-m.quit:
		return true
	case <-m.ctx.Done():
		return true
	default:
		return false
	}
}

func (m *Messenger) markSupportBotContactRequestDone() error {
	return m.settings.SaveSettingField(settings.SupportBotContactRequestState, settings.SupportBotContactRequestStateDone)
}

func supportBotContactRequestMessage(state int64) (string, error) {
	switch state {
	case settings.SupportBotContactRequestStatePendingExisting:
		return supportBotExistingUserMessage, nil
	case settings.SupportBotContactRequestStatePendingNew:
		return supportBotNewUserMessage, nil
	default:
		return "", fmt.Errorf("invalid support bot contact request state: %d", state)
	}
}
